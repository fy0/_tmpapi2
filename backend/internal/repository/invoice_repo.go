package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type invoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) service.InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) GetSummary(ctx context.Context, userID int64) (*service.InvoiceSummary, error) {
	const query = `
SELECT
	COALESCE((
		SELECT SUM(rc.value)
		FROM redeem_codes rc
		LEFT JOIN invoice_request_redeem_codes irc ON irc.redeem_code_id = rc.id AND irc.released_at IS NULL
		WHERE rc.used_by = $1
		  AND rc.status = 'used'
		  AND rc.value > 0
		  AND rc.type IN ('balance', 'admin_balance')
		  AND irc.redeem_code_id IS NULL
	), 0) AS available_amount,
	COALESCE((
		SELECT SUM(ir.amount)
		FROM invoice_requests ir
		WHERE ir.user_id = $1 AND ir.status = 'pending'
	), 0) AS pending_amount,
	COALESCE((
		SELECT SUM(ir.amount)
		FROM invoice_requests ir
		WHERE ir.user_id = $1 AND ir.status = 'issued'
	), 0) AS issued_amount`

	var summary service.InvoiceSummary
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&summary.AvailableAmount, &summary.PendingAmount, &summary.IssuedAmount); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *invoiceRepository) GetProfile(ctx context.Context, userID int64) (*service.InvoiceProfile, error) {
	const query = `
SELECT invoice_title, tax_no, updated_at
FROM invoice_profiles
WHERE user_id = $1`

	var profile service.InvoiceProfile
	var updatedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&profile.InvoiceTitle, &profile.TaxNo, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if updatedAt.Valid {
		profile.UpdatedAt = &updatedAt.Time
	}
	return &profile, nil
}

func (r *invoiceRepository) ListAvailableRecharges(ctx context.Context, userID int64) ([]service.InvoiceRecharge, error) {
	const query = `
SELECT rc.id, rc.code, rc.type, rc.value, rc.used_at, rc.created_at
FROM redeem_codes rc
LEFT JOIN invoice_request_redeem_codes irc ON irc.redeem_code_id = rc.id AND irc.released_at IS NULL
WHERE rc.used_by = $1
  AND rc.status = 'used'
  AND rc.value > 0
  AND rc.type IN ('balance', 'admin_balance')
  AND irc.redeem_code_id IS NULL
ORDER BY rc.used_at DESC NULLS LAST, rc.id DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.InvoiceRecharge
	for rows.Next() {
		recharge, err := scanInvoiceRecharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, recharge)
	}
	return out, rows.Err()
}

func (r *invoiceRepository) CreateRequest(ctx context.Context, input service.CreateInvoiceRequestInput) (*service.InvoiceRequest, error) {
	if len(input.RedeemCodeIDs) == 0 {
		return nil, service.ErrInvoiceRechargeUnavailable
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	recharges, err := r.lockAvailableRecharges(ctx, tx, input.UserID, input.RedeemCodeIDs)
	if err != nil {
		return nil, err
	}
	if len(recharges) != len(input.RedeemCodeIDs) {
		return nil, service.ErrInvoiceRechargeUnavailable
	}

	var amount float64
	for _, recharge := range recharges {
		amount += recharge.Value
	}
	if amount <= 0 {
		return nil, service.ErrInvoiceRechargeUnavailable
	}
	if input.MinAmount > 0 && amount < input.MinAmount {
		return nil, service.ErrInvoiceAmountBelowMinimum.WithMetadata(map[string]string{
			"minimum": strconv.FormatFloat(input.MinAmount, 'f', -1, 64),
			"amount":  strconv.FormatFloat(amount, 'f', -1, 64),
		})
	}
	if input.MaxAmount > 0 && amount > input.MaxAmount {
		return nil, service.ErrInvoiceAmountAboveMaximum.WithMetadata(map[string]string{
			"maximum": strconv.FormatFloat(input.MaxAmount, 'f', -1, 64),
			"amount":  strconv.FormatFloat(amount, 'f', -1, 64),
		})
	}

	var requestID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO invoice_requests (user_id, status, invoice_title, tax_no, amount, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`,
		input.UserID,
		service.InvoiceStatusPending,
		input.InvoiceTitle,
		input.TaxNo,
		amount,
		input.Note,
	).Scan(&requestID)
	if err != nil {
		return nil, translateInvoiceError(err)
	}

	if err := r.upsertInvoiceProfile(ctx, tx, input.UserID, input.InvoiceTitle, input.TaxNo); err != nil {
		return nil, translateInvoiceError(err)
	}

	for _, recharge := range recharges {
		_, err = tx.ExecContext(ctx, `
INSERT INTO invoice_request_redeem_codes (invoice_request_id, redeem_code_id, amount)
VALUES ($1, $2, $3)`,
			requestID,
			recharge.ID,
			recharge.Value,
		)
		if err != nil {
			return nil, translateInvoiceError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, translateInvoiceError(err)
	}
	return r.GetByID(ctx, requestID)
}

func (r *invoiceRepository) upsertInvoiceProfile(ctx context.Context, tx *sql.Tx, userID int64, invoiceTitle, taxNo string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO invoice_profiles (user_id, invoice_title, tax_no)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET invoice_title = EXCLUDED.invoice_title,
    tax_no = EXCLUDED.tax_no,
    updated_at = NOW()`,
		userID,
		invoiceTitle,
		taxNo,
	)
	return err
}

func (r *invoiceRepository) ListUserInvoices(ctx context.Context, userID int64, params service.InvoiceListParams) ([]service.InvoiceRequest, int64, error) {
	where := []string{"ir.user_id = $1"}
	args := []any{userID}
	nextArg := 2
	applyInvoiceListFilters(&where, &args, &nextArg, params, false)
	return r.listInvoices(ctx, where, args, params)
}

func (r *invoiceRepository) ListAdminInvoices(ctx context.Context, params service.InvoiceListParams) ([]service.InvoiceRequest, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	nextArg := 1
	applyInvoiceListFilters(&where, &args, &nextArg, params, true)
	return r.listInvoices(ctx, where, args, params)
}

func (r *invoiceRepository) GetByID(ctx context.Context, id int64) (*service.InvoiceRequest, error) {
	return r.getInvoice(ctx, "ir.id = $1", id)
}

func (r *invoiceRepository) GetByIDForUser(ctx context.Context, userID, id int64) (*service.InvoiceRequest, error) {
	return r.getInvoice(ctx, "ir.id = $1 AND ir.user_id = $2", id, userID)
}

func (r *invoiceRepository) UpdateIssuedFile(ctx context.Context, id, adminID int64, file service.InvoiceStoredFile) (*service.InvoiceRequest, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE invoice_requests
SET status = $2,
    file_name = $3,
    file_path = $4,
    content_type = $5,
    file_size = $6,
    uploaded_by = $7,
    issued_at = NOW(),
    rejected_at = NULL,
    withdrawn_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status IN ($8, $9)`,
		id,
		service.InvoiceStatusIssued,
		file.FileName,
		file.FilePath,
		file.ContentType,
		file.FileSize,
		adminID,
		service.InvoiceStatusPending,
		service.InvoiceStatusIssued,
	)
	if err != nil {
		return nil, translateInvoiceError(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrInvoiceNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *invoiceRepository) ClearIssuedFile(ctx context.Context, id int64) (*service.InvoiceRequest, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE invoice_requests
SET status = $2,
    file_name = '',
    file_path = '',
    content_type = '',
    file_size = 0,
    uploaded_by = NULL,
    issued_at = NULL,
    rejected_at = NULL,
    withdrawn_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND file_name <> ''
  AND file_path <> ''`,
		id,
		service.InvoiceStatusPending,
	)
	if err != nil {
		return nil, translateInvoiceError(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrInvoiceFileUnavailable
	}
	return r.GetByID(ctx, id)
}

func (r *invoiceRepository) WithdrawRequest(ctx context.Context, userID, id int64, now time.Time) (*service.InvoiceRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
SELECT status, created_at
FROM invoice_requests
WHERE id = $1 AND user_id = $2
FOR UPDATE`,
		id,
		userID,
	).Scan(&status, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrInvoiceNotFound
		}
		return nil, err
	}
	if status != service.InvoiceStatusPending {
		return nil, service.ErrInvoiceWithdrawUnavailable
	}
	if now.After(createdAt.Add(service.InvoiceWithdrawWindow)) {
		return nil, service.ErrInvoiceWithdrawExpired
	}

	res, err := tx.ExecContext(ctx, `
UPDATE invoice_requests
SET status = $3,
    withdrawn_at = $4,
    updated_at = $4
WHERE id = $1 AND user_id = $2 AND status = $5`,
		id,
		userID,
		service.InvoiceStatusWithdrawn,
		now,
		service.InvoiceStatusPending,
	)
	if err != nil {
		return nil, translateInvoiceError(err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrInvoiceWithdrawUnavailable
	}

	_, err = tx.ExecContext(ctx, `
UPDATE invoice_request_redeem_codes
SET released_at = $2
WHERE invoice_request_id = $1 AND released_at IS NULL`,
		id,
		now,
	)
	if err != nil {
		return nil, translateInvoiceError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, translateInvoiceError(err)
	}
	return r.GetByID(ctx, id)
}

func (r *invoiceRepository) lockAvailableRecharges(ctx context.Context, tx *sql.Tx, userID int64, ids []int64) ([]service.InvoiceRecharge, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT rc.id, rc.code, rc.type, rc.value, rc.used_at, rc.created_at
FROM redeem_codes rc
LEFT JOIN invoice_request_redeem_codes irc ON irc.redeem_code_id = rc.id AND irc.released_at IS NULL
WHERE rc.id = ANY($1)
  AND rc.used_by = $2
  AND rc.status = 'used'
  AND rc.value > 0
  AND rc.type IN ('balance', 'admin_balance')
  AND irc.redeem_code_id IS NULL
ORDER BY rc.id ASC
FOR UPDATE OF rc`,
		pq.Array(ids),
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.InvoiceRecharge, 0, len(ids))
	for rows.Next() {
		recharge, err := scanInvoiceRecharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, recharge)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *invoiceRepository) listInvoices(ctx context.Context, where []string, args []any, params service.InvoiceListParams) ([]service.InvoiceRequest, int64, error) {
	whereSQL := strings.Join(where, " AND ")
	countQuery := "SELECT COUNT(*) FROM invoice_requests ir LEFT JOIN users u ON u.id = ir.user_id WHERE " + whereSQL
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []service.InvoiceRequest{}, 0, nil
	}

	page, pageSize := normalizedInvoicePage(params)
	offset := (page - 1) * pageSize
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, pageSize, offset)
	limitArg := len(listArgs) - 1
	offsetArg := len(listArgs)

	query := invoiceSelectSQL() + " WHERE " + whereSQL + fmt.Sprintf(" ORDER BY ir.created_at DESC, ir.id DESC LIMIT $%d OFFSET $%d", limitArg, offsetArg)
	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.InvoiceRequest, 0, pageSize)
	for rows.Next() {
		item, err := scanInvoiceRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.attachRecharges(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *invoiceRepository) getInvoice(ctx context.Context, condition string, args ...any) (*service.InvoiceRequest, error) {
	query := invoiceSelectSQL() + " WHERE " + condition
	item, err := scanInvoiceRequest(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrInvoiceNotFound
		}
		return nil, err
	}
	items := []service.InvoiceRequest{*item}
	if err := r.attachRecharges(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *invoiceRepository) attachRecharges(ctx context.Context, items []service.InvoiceRequest) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	byID := make(map[int64]*service.InvoiceRequest, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
		byID[items[i].ID] = &items[i]
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT irc.invoice_request_id, rc.id, rc.code, rc.type, irc.amount, rc.used_at, rc.created_at
FROM invoice_request_redeem_codes irc
JOIN redeem_codes rc ON rc.id = irc.redeem_code_id
WHERE irc.invoice_request_id = ANY($1)
ORDER BY rc.used_at DESC NULLS LAST, rc.id DESC`,
		pq.Array(ids),
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var invoiceID int64
		var recharge service.InvoiceRecharge
		var usedAt sql.NullTime
		if err := rows.Scan(
			&invoiceID,
			&recharge.ID,
			&recharge.Code,
			&recharge.Type,
			&recharge.Value,
			&usedAt,
			&recharge.CreatedAt,
		); err != nil {
			return err
		}
		if usedAt.Valid {
			recharge.UsedAt = &usedAt.Time
		}
		if item := byID[invoiceID]; item != nil {
			item.Recharges = append(item.Recharges, recharge)
		}
	}
	return rows.Err()
}

func invoiceSelectSQL() string {
	return `
SELECT
	ir.id,
	ir.user_id,
	COALESCE(u.email, '') AS user_email,
	ir.status,
	ir.invoice_title,
	ir.tax_no,
	ir.amount,
	ir.note,
	ir.admin_note,
	ir.file_name,
	ir.file_path,
	ir.content_type,
	ir.file_size,
	ir.uploaded_by,
	ir.issued_at,
	ir.rejected_at,
	ir.withdrawn_at,
	ir.created_at,
	ir.updated_at
FROM invoice_requests ir
LEFT JOIN users u ON u.id = ir.user_id`
}

func applyInvoiceListFilters(where *[]string, args *[]any, nextArg *int, params service.InvoiceListParams, admin bool) {
	if params.Status != "" {
		*where = append(*where, fmt.Sprintf("ir.status = $%d", *nextArg))
		*args = append(*args, params.Status)
		(*nextArg)++
	}
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		if admin {
			*where = append(*where, fmt.Sprintf("(ir.invoice_title ILIKE $%d OR ir.tax_no ILIKE $%d OR u.email ILIKE $%d)", *nextArg, *nextArg, *nextArg))
		} else {
			*where = append(*where, fmt.Sprintf("(ir.invoice_title ILIKE $%d OR ir.tax_no ILIKE $%d)", *nextArg, *nextArg))
		}
		*args = append(*args, like)
		(*nextArg)++
	}
	if params.MinAgeHours > 0 {
		*where = append(*where, fmt.Sprintf("ir.created_at <= NOW() - ($%d::integer * INTERVAL '1 hour')", *nextArg))
		*args = append(*args, params.MinAgeHours)
		(*nextArg)++
	}
}

func normalizedInvoicePage(params service.InvoiceListParams) (int, int) {
	page := params.Page
	pageSize := params.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	return page, pageSize
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInvoiceRequest(scanner rowScanner) (*service.InvoiceRequest, error) {
	var item service.InvoiceRequest
	var uploadedBy sql.NullInt64
	var issuedAt sql.NullTime
	var rejectedAt sql.NullTime
	var withdrawnAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.UserEmail,
		&item.Status,
		&item.InvoiceTitle,
		&item.TaxNo,
		&item.Amount,
		&item.Note,
		&item.AdminNote,
		&item.FileName,
		&item.FilePath,
		&item.ContentType,
		&item.FileSize,
		&uploadedBy,
		&issuedAt,
		&rejectedAt,
		&withdrawnAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if uploadedBy.Valid {
		item.UploadedBy = &uploadedBy.Int64
	}
	if issuedAt.Valid {
		item.IssuedAt = &issuedAt.Time
	}
	if rejectedAt.Valid {
		item.RejectedAt = &rejectedAt.Time
	}
	if withdrawnAt.Valid {
		item.WithdrawnAt = &withdrawnAt.Time
	}
	return &item, nil
}

func scanInvoiceRecharge(scanner rowScanner) (service.InvoiceRecharge, error) {
	var recharge service.InvoiceRecharge
	var usedAt sql.NullTime
	err := scanner.Scan(
		&recharge.ID,
		&recharge.Code,
		&recharge.Type,
		&recharge.Value,
		&usedAt,
		&recharge.CreatedAt,
	)
	if usedAt.Valid {
		recharge.UsedAt = &usedAt.Time
	}
	return recharge, err
}

func translateInvoiceError(err error) error {
	if err == nil {
		return nil
	}
	if isUniqueConstraintViolation(err) {
		return service.ErrInvoiceRechargeUnavailable.WithCause(err)
	}
	return err
}
