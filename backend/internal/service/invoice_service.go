package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	InvoiceStatusPending  = "pending"
	InvoiceStatusIssued   = "issued"
	InvoiceStatusRejected = "rejected"

	defaultInvoiceStorageDir = "data/invoices"
	maxInvoiceUploadBytes    = 20 << 20
)

var (
	ErrInvoiceNotFound            = infraerrors.NotFound("INVOICE_NOT_FOUND", "invoice request not found")
	ErrInvoiceRechargeUnavailable = infraerrors.Conflict("INVOICE_RECHARGE_UNAVAILABLE", "selected recharge is not available for invoice")
	ErrInvoiceFileUnavailable     = infraerrors.NotFound("INVOICE_FILE_UNAVAILABLE", "invoice file is not available")
	ErrInvoiceFileTooLarge        = infraerrors.BadRequest("INVOICE_FILE_TOO_LARGE", "invoice file is too large")
)

type InvoiceRepository interface {
	GetSummary(ctx context.Context, userID int64) (*InvoiceSummary, error)
	ListAvailableRecharges(ctx context.Context, userID int64) ([]InvoiceRecharge, error)
	CreateRequest(ctx context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error)
	ListUserInvoices(ctx context.Context, userID int64, params InvoiceListParams) ([]InvoiceRequest, int64, error)
	ListAdminInvoices(ctx context.Context, params InvoiceListParams) ([]InvoiceRequest, int64, error)
	GetByID(ctx context.Context, id int64) (*InvoiceRequest, error)
	GetByIDForUser(ctx context.Context, userID, id int64) (*InvoiceRequest, error)
	UpdateIssuedFile(ctx context.Context, id, adminID int64, file InvoiceStoredFile) (*InvoiceRequest, error)
}

type InvoiceService struct {
	repo       InvoiceRepository
	storageDir string
}

func NewInvoiceService(repo InvoiceRepository) *InvoiceService {
	return &InvoiceService{
		repo:       repo,
		storageDir: defaultInvoiceStorageDir,
	}
}

type InvoiceSummary struct {
	AvailableAmount float64 `json:"available_amount"`
	PendingAmount   float64 `json:"pending_amount"`
	IssuedAmount    float64 `json:"issued_amount"`
}

type InvoiceRecharge struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Type      string     `json:"type"`
	Value     float64    `json:"value"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type InvoiceRequest struct {
	ID           int64             `json:"id"`
	UserID       int64             `json:"user_id"`
	UserEmail    string            `json:"user_email,omitempty"`
	Status       string            `json:"status"`
	InvoiceTitle string            `json:"invoice_title"`
	TaxNo        string            `json:"tax_no"`
	Amount       float64           `json:"amount"`
	Note         string            `json:"note"`
	AdminNote    string            `json:"admin_note"`
	FileName     string            `json:"file_name,omitempty"`
	FilePath     string            `json:"-"`
	ContentType  string            `json:"content_type,omitempty"`
	FileSize     int64             `json:"file_size,omitempty"`
	UploadedBy   *int64            `json:"uploaded_by,omitempty"`
	IssuedAt     *time.Time        `json:"issued_at,omitempty"`
	RejectedAt   *time.Time        `json:"rejected_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Recharges    []InvoiceRecharge `json:"recharges,omitempty"`
}

type CreateInvoiceRequestInput struct {
	UserID        int64
	InvoiceTitle  string
	TaxNo         string
	Note          string
	RedeemCodeIDs []int64
}

type InvoiceListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

type UploadInvoiceFileInput struct {
	InvoiceID   int64
	AdminID     int64
	FileName    string
	ContentType string
	Size        int64
	Reader      io.Reader
}

type InvoiceStoredFile struct {
	FileName    string
	FilePath    string
	ContentType string
	FileSize    int64
}

type InvoiceFile struct {
	FileName    string
	FilePath    string
	ContentType string
	FileSize    int64
}

func (s *InvoiceService) GetSummary(ctx context.Context, userID int64) (*InvoiceSummary, error) {
	return s.repo.GetSummary(ctx, userID)
}

func (s *InvoiceService) ListAvailableRecharges(ctx context.Context, userID int64) ([]InvoiceRecharge, error) {
	items, err := s.repo.ListAvailableRecharges(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []InvoiceRecharge{}, nil
	}
	return items, nil
}

func (s *InvoiceService) CreateRequest(ctx context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	input.InvoiceTitle = strings.TrimSpace(input.InvoiceTitle)
	input.TaxNo = strings.TrimSpace(input.TaxNo)
	input.Note = strings.TrimSpace(input.Note)
	input.RedeemCodeIDs = normalizeInvoiceRedeemIDs(input.RedeemCodeIDs)

	if input.UserID <= 0 {
		return nil, infraerrors.BadRequest("INVOICE_USER_INVALID", "user is required")
	}
	if input.InvoiceTitle == "" {
		return nil, infraerrors.BadRequest("INVOICE_TITLE_REQUIRED", "invoice title is required")
	}
	if len(input.InvoiceTitle) > 200 {
		return nil, infraerrors.BadRequest("INVOICE_TITLE_TOO_LONG", "invoice title is too long")
	}
	if len(input.TaxNo) > 100 {
		return nil, infraerrors.BadRequest("INVOICE_TAX_NO_TOO_LONG", "tax number is too long")
	}
	if len(input.Note) > 1000 {
		return nil, infraerrors.BadRequest("INVOICE_NOTE_TOO_LONG", "invoice note is too long")
	}
	if len(input.RedeemCodeIDs) == 0 {
		return nil, infraerrors.BadRequest("INVOICE_RECHARGES_REQUIRED", "at least one recharge is required")
	}

	return s.repo.CreateRequest(ctx, input)
}

func (s *InvoiceService) ListUserInvoices(ctx context.Context, userID int64, params InvoiceListParams) ([]InvoiceRequest, int64, error) {
	params = normalizeInvoiceListParams(params)
	items, total, err := s.repo.ListUserInvoices(ctx, userID, params)
	if err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []InvoiceRequest{}
	}
	return items, total, nil
}

func (s *InvoiceService) ListAdminInvoices(ctx context.Context, params InvoiceListParams) ([]InvoiceRequest, int64, error) {
	params = normalizeInvoiceListParams(params)
	items, total, err := s.repo.ListAdminInvoices(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []InvoiceRequest{}
	}
	return items, total, nil
}

func (s *InvoiceService) GetAdminInvoice(ctx context.Context, id int64) (*InvoiceRequest, error) {
	if id <= 0 {
		return nil, ErrInvoiceNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *InvoiceService) GetUserInvoice(ctx context.Context, userID, id int64) (*InvoiceRequest, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrInvoiceNotFound
	}
	return s.repo.GetByIDForUser(ctx, userID, id)
}

func (s *InvoiceService) GetUserInvoiceFile(ctx context.Context, userID, id int64) (*InvoiceFile, error) {
	invoice, err := s.GetUserInvoice(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return invoiceFileFromRequest(invoice)
}

func (s *InvoiceService) GetAdminInvoiceFile(ctx context.Context, id int64) (*InvoiceFile, error) {
	invoice, err := s.GetAdminInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	return invoiceFileFromRequest(invoice)
}

func (s *InvoiceService) UploadIssuedFile(ctx context.Context, input UploadInvoiceFileInput) (*InvoiceRequest, error) {
	if input.InvoiceID <= 0 {
		return nil, ErrInvoiceNotFound
	}
	if input.AdminID <= 0 {
		return nil, infraerrors.BadRequest("INVOICE_ADMIN_INVALID", "admin user is required")
	}
	if input.Reader == nil {
		return nil, infraerrors.BadRequest("INVOICE_FILE_REQUIRED", "invoice file is required")
	}
	if input.Size > maxInvoiceUploadBytes {
		return nil, ErrInvoiceFileTooLarge
	}

	if _, err := s.repo.GetByID(ctx, input.InvoiceID); err != nil {
		return nil, err
	}

	safeName := sanitizeInvoiceFileName(input.FileName)
	targetDir := filepath.Join(s.storageDir, fmt.Sprintf("%d", input.InvoiceID))
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return nil, fmt.Errorf("create invoice storage directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeName))
	size, err := writeInvoiceFile(targetPath, input.Reader)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, err
	}

	file := InvoiceStoredFile{
		FileName:    safeName,
		FilePath:    targetPath,
		ContentType: strings.TrimSpace(input.ContentType),
		FileSize:    size,
	}
	updated, err := s.repo.UpdateIssuedFile(ctx, input.InvoiceID, input.AdminID, file)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, err
	}
	return updated, nil
}

func normalizeInvoiceRedeemIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeInvoiceListParams(params InvoiceListParams) InvoiceListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 1000 {
		params.PageSize = 1000
	}
	params.Status = strings.TrimSpace(params.Status)
	switch params.Status {
	case "", InvoiceStatusPending, InvoiceStatusIssued, InvoiceStatusRejected:
	default:
		params.Status = ""
	}
	params.Keyword = strings.TrimSpace(params.Keyword)
	return params
}

func invoiceFileFromRequest(invoice *InvoiceRequest) (*InvoiceFile, error) {
	if invoice == nil || invoice.Status != InvoiceStatusIssued || strings.TrimSpace(invoice.FileName) == "" {
		return nil, ErrInvoiceFileUnavailable
	}
	if strings.TrimSpace(invoice.FileName) == "" || strings.TrimSpace(invoiceFilePath(invoice)) == "" {
		return nil, ErrInvoiceFileUnavailable
	}
	if _, err := os.Stat(invoiceFilePath(invoice)); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrInvoiceFileUnavailable
		}
		return nil, fmt.Errorf("stat invoice file: %w", err)
	}
	return &InvoiceFile{
		FileName:    invoice.FileName,
		FilePath:    invoiceFilePath(invoice),
		ContentType: invoice.ContentType,
		FileSize:    invoice.FileSize,
	}, nil
}

func invoiceFilePath(invoice *InvoiceRequest) string {
	if invoice == nil {
		return ""
	}
	return invoice.FilePath
}

func sanitizeInvoiceFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "invoice"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			_, _ = b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			_, _ = b.WriteRune(r)
		case r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
		case r == '.', r == '-', r == '_', r == ' ', r == '(', r == ')':
			_, _ = b.WriteRune(r)
		default:
			_ = b.WriteByte('_')
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" || out == "." {
		out = "invoice"
	}
	if len(out) > 120 {
		ext := filepath.Ext(out)
		stem := strings.TrimSuffix(out, ext)
		if len(ext) > 20 {
			ext = ""
		}
		limit := 120 - len(ext)
		if limit < 1 {
			limit = 120
		}
		if len(stem) > limit {
			stem = stem[:limit]
		}
		out = stem + ext
	}
	return out
}

func writeInvoiceFile(path string, r io.Reader) (int64, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return 0, fmt.Errorf("create invoice file: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, io.LimitReader(r, maxInvoiceUploadBytes+1))
	if err != nil {
		return 0, fmt.Errorf("write invoice file: %w", err)
	}
	if n > maxInvoiceUploadBytes {
		return 0, ErrInvoiceFileTooLarge
	}
	if n == 0 {
		return 0, infraerrors.BadRequest("INVOICE_FILE_EMPTY", "invoice file is empty")
	}
	return n, nil
}
