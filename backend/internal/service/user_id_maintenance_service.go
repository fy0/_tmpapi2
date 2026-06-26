package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	userIDMaintenanceConfirmation = "CHANGE_USER_ID"
	userIDMaintenanceAdvisoryKey  = int64(0x757365725f6964)
)

// UserIDMaintenanceStatus describes the current users.id sequence state.
type UserIDMaintenanceStatus struct {
	MaxUserID    int64  `json:"max_user_id"`
	NextUserID   int64  `json:"next_user_id"`
	SequenceName string `json:"sequence_name"`
}

// SetUserNextIDResult is returned after advancing the users.id sequence.
type SetUserNextIDResult struct {
	MaxUserID    int64 `json:"max_user_id"`
	PreviousNext int64 `json:"previous_next_user_id"`
	NextUserID   int64 `json:"next_user_id"`
}

// ChangeUserIDRequest contains the service-level request for a true user id migration.
type ChangeUserIDRequest struct {
	OldUserID      int64
	NewUserID      int64
	OperatorUserID int64
	Confirmation   string
}

// ChangeUserIDResult describes a completed user id migration.
type ChangeUserIDResult struct {
	OldUserID       int64            `json:"old_user_id"`
	NewUserID       int64            `json:"new_user_id"`
	UpdatedRows     int64            `json:"updated_rows"`
	UpdatedByTarget map[string]int64 `json:"updated_by_target"`
	NextUserID      int64            `json:"next_user_id"`
}

type userIDMaintenanceFKRef struct {
	Schema string
	Table  string
	Column string
}

type userIDMaintenanceColumnRef struct {
	Schema string
	Table  string
	Column string
}

type userIDMaintenanceCacheRefs struct {
	SubscriptionGroupIDs []int64
	QuotaPlatforms       []string
}

// UserIDMaintenanceService performs admin-only maintenance on users.id.
type UserIDMaintenanceService struct {
	db                   *sql.DB
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCache         BillingCache
}

func NewUserIDMaintenanceService(
	db *sql.DB,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCache BillingCache,
) *UserIDMaintenanceService {
	return &UserIDMaintenanceService{
		db:                   db,
		authCacheInvalidator: authCacheInvalidator,
		billingCache:         billingCache,
	}
}

func (s *UserIDMaintenanceService) GetStatus(ctx context.Context) (*UserIDMaintenanceStatus, error) {
	if s == nil || s.db == nil {
		return nil, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable")
	}
	return s.getStatus(ctx, s.db)
}

func (s *UserIDMaintenanceService) SetNextUserID(ctx context.Context, nextUserID int64) (*SetUserNextIDResult, error) {
	if s == nil || s.db == nil {
		return nil, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable")
	}
	if nextUserID <= 0 {
		return nil, infraerrors.BadRequest("USER_ID_INVALID_NEXT_ID", "next user id must be positive")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin user id maintenance transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	if err := s.lock(ctx, tx); err != nil {
		return nil, err
	}

	status, err := s.getStatus(ctx, tx)
	if err != nil {
		return nil, err
	}
	if nextUserID <= status.NextUserID {
		return nil, infraerrors.BadRequest("USER_ID_NEXT_ID_NOT_INCREASED", "next user id must be greater than current next user id").WithMetadata(map[string]string{
			"current_next_user_id": fmt.Sprintf("%d", status.NextUserID),
		})
	}
	if nextUserID <= status.MaxUserID {
		return nil, infraerrors.BadRequest("USER_ID_NEXT_ID_BELOW_MAX", "next user id must be greater than current max user id").WithMetadata(map[string]string{
			"max_user_id": fmt.Sprintf("%d", status.MaxUserID),
		})
	}

	if err := s.setSequenceNext(ctx, tx, status.SequenceName, nextUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit set next user id: %w", err)
	}

	slog.Info("admin.user_id_next_id_updated",
		"audit", true,
		"previous_next_user_id", status.NextUserID,
		"next_user_id", nextUserID,
		"max_user_id", status.MaxUserID,
	)

	return &SetUserNextIDResult{
		MaxUserID:    status.MaxUserID,
		PreviousNext: status.NextUserID,
		NextUserID:   nextUserID,
	}, nil
}

func (s *UserIDMaintenanceService) ChangeUserID(ctx context.Context, req ChangeUserIDRequest) (*ChangeUserIDResult, error) {
	if s == nil || s.db == nil {
		return nil, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable")
	}
	if req.Confirmation != userIDMaintenanceConfirmation {
		return nil, infraerrors.BadRequest("USER_ID_CHANGE_CONFIRMATION_REQUIRED", "confirmation text is invalid")
	}
	if req.OldUserID <= 0 || req.NewUserID <= 0 {
		return nil, infraerrors.BadRequest("USER_ID_INVALID_ID", "user ids must be positive")
	}
	if req.OldUserID == req.NewUserID {
		return nil, infraerrors.BadRequest("USER_ID_UNCHANGED", "new user id must be different")
	}
	if req.OperatorUserID > 0 && req.OperatorUserID == req.OldUserID {
		return nil, infraerrors.Forbidden("USER_ID_SELF_CHANGE_FORBIDDEN", "admin cannot change their own user id")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin user id change transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	if err := s.lock(ctx, tx); err != nil {
		return nil, err
	}

	sourceEmail, err := s.lockUserRow(ctx, tx, req.OldUserID)
	if err != nil {
		return nil, err
	}
	if exists, err := s.userExists(ctx, tx, req.NewUserID); err != nil {
		return nil, err
	} else if exists {
		return nil, infraerrors.Conflict("USER_ID_TARGET_EXISTS", "target user id already exists")
	}

	nonFKRefs, err := s.existingNonFKRefs(ctx, tx, req.NewUserID)
	if err != nil {
		return nil, err
	}
	if len(nonFKRefs) > 0 {
		return nil, infraerrors.Conflict("USER_ID_TARGET_REFERENCED", "target user id is already referenced by existing records").WithMetadata(map[string]string{
			"references": strings.Join(nonFKRefs, ","),
		})
	}

	cacheRefs, err := s.collectCacheRefs(ctx, tx, req.OldUserID)
	if err != nil {
		return nil, err
	}

	columns, err := s.usersColumns(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, infraerrors.InternalServer("USER_ID_USERS_COLUMNS_NOT_FOUND", "users columns not found")
	}

	placeholderEmail := fmt.Sprintf("__id_migration_%d_%d@id-migration.invalid", req.NewUserID, time.Now().UnixNano())
	if err := s.insertPlaceholderUser(ctx, tx, req.OldUserID, req.NewUserID, placeholderEmail, columns); err != nil {
		return nil, err
	}

	updatedByTarget := map[string]int64{}
	totalUpdated := int64(0)
	addCount := func(ref userIDMaintenanceColumnRef, n int64) {
		key := ref.Schema + "." + ref.Table + "." + ref.Column
		updatedByTarget[key] += n
		totalUpdated += n
	}

	fkRefs, err := s.userFKRefs(ctx, tx)
	if err != nil {
		return nil, err
	}
	for _, ref := range fkRefs {
		if ref.Table == "users" && ref.Column == "id" {
			continue
		}
		affected, err := updateColumnRef(ctx, tx, userIDMaintenanceColumnRef(ref), req.OldUserID, req.NewUserID)
		if err != nil {
			return nil, fmt.Errorf("update fk ref %s.%s.%s: %w", ref.Schema, ref.Table, ref.Column, err)
		}
		addCount(userIDMaintenanceColumnRef(ref), affected)
	}

	nonFKAllowlist := userIDMaintenanceNonFKRefs()
	fkSet := make(map[string]struct{}, len(fkRefs))
	for _, ref := range fkRefs {
		fkSet[columnRefKey(userIDMaintenanceColumnRef(ref))] = struct{}{}
	}
	for _, ref := range nonFKAllowlist {
		if _, ok := fkSet[columnRefKey(ref)]; ok {
			continue
		}
		if ok, err := s.columnExists(ctx, tx, ref); err != nil {
			return nil, err
		} else if !ok {
			continue
		}
		affected, err := updateColumnRef(ctx, tx, ref, req.OldUserID, req.NewUserID)
		if err != nil {
			return nil, fmt.Errorf("update non-fk ref %s.%s.%s: %w", ref.Schema, ref.Table, ref.Column, err)
		}
		addCount(ref, affected)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, req.OldUserID); err != nil {
		return nil, fmt.Errorf("delete old user row: %w", err)
	}
	if err := s.restoreUserEmail(ctx, tx, req.NewUserID, sourceEmail); err != nil {
		return nil, err
	}

	status, err := s.getStatus(ctx, tx)
	if err != nil {
		return nil, err
	}
	nextUserID := status.NextUserID
	if req.NewUserID >= status.NextUserID {
		nextUserID = req.NewUserID + 1
		if err := s.setSequenceNext(ctx, tx, status.SequenceName, nextUserID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit change user id: %w", err)
	}

	s.invalidateCaches(ctx, req.OldUserID, req.NewUserID, cacheRefs)
	slog.Info("admin.user_id_changed",
		"audit", true,
		"operator_user_id", req.OperatorUserID,
		"old_user_id", req.OldUserID,
		"new_user_id", req.NewUserID,
		"updated_rows", totalUpdated,
		"next_user_id", nextUserID,
	)

	return &ChangeUserIDResult{
		OldUserID:       req.OldUserID,
		NewUserID:       req.NewUserID,
		UpdatedRows:     totalUpdated,
		UpdatedByTarget: updatedByTarget,
		NextUserID:      nextUserID,
	}, nil
}

func rollbackUnlessCommitted(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type dbtx interface {
	queryer
	execer
}

func (s *UserIDMaintenanceService) getStatus(ctx context.Context, q queryer) (*UserIDMaintenanceStatus, error) {
	sequenceName, err := s.sequenceName(ctx, q)
	if err != nil {
		return nil, err
	}
	var maxID int64
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM users`).Scan(&maxID); err != nil {
		return nil, fmt.Errorf("get max user id: %w", err)
	}

	var lastValue int64
	var isCalled bool
	query := fmt.Sprintf(`SELECT last_value, is_called FROM %s`, quoteSequenceName(sequenceName))
	if err := q.QueryRowContext(ctx, query).Scan(&lastValue, &isCalled); err != nil {
		return nil, fmt.Errorf("get user id sequence state: %w", err)
	}
	nextID := lastValue
	if isCalled {
		nextID = lastValue + 1
	}
	if nextID < 1 {
		nextID = 1
	}

	return &UserIDMaintenanceStatus{
		MaxUserID:    maxID,
		NextUserID:   nextID,
		SequenceName: sequenceName,
	}, nil
}

func (s *UserIDMaintenanceService) sequenceName(ctx context.Context, q queryer) (string, error) {
	var seq sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT pg_get_serial_sequence('users', 'id')`).Scan(&seq); err != nil {
		return "", fmt.Errorf("get users id sequence name: %w", err)
	}
	if !seq.Valid || strings.TrimSpace(seq.String) == "" {
		return "", infraerrors.InternalServer("USER_ID_SEQUENCE_NOT_FOUND", "users.id sequence not found")
	}
	return seq.String, nil
}

func (s *UserIDMaintenanceService) setSequenceNext(ctx context.Context, e execer, sequenceName string, nextUserID int64) error {
	if nextUserID <= 0 {
		return infraerrors.BadRequest("USER_ID_INVALID_NEXT_ID", "next user id must be positive")
	}
	_, err := e.ExecContext(ctx, `SELECT setval($1::regclass, $2, true)`, sequenceName, nextUserID-1)
	if err != nil {
		return fmt.Errorf("set users id sequence: %w", err)
	}
	return nil
}

func (s *UserIDMaintenanceService) lock(ctx context.Context, e execer) error {
	if _, err := e.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, userIDMaintenanceAdvisoryKey); err != nil {
		return fmt.Errorf("acquire user id maintenance advisory lock: %w", err)
	}
	return nil
}

func (s *UserIDMaintenanceService) lockUserRow(ctx context.Context, q queryer, userID int64) (string, error) {
	var id int64
	var email string
	err := q.QueryRowContext(ctx, `SELECT id, email FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id, &email)
	if err == sql.ErrNoRows {
		return "", infraerrors.NotFound("USER_ID_SOURCE_NOT_FOUND", "source user not found")
	}
	if err != nil {
		return "", fmt.Errorf("lock source user: %w", err)
	}
	return email, nil
}

func (s *UserIDMaintenanceService) userExists(ctx context.Context, q queryer, userID int64) (bool, error) {
	var exists bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return exists, nil
}

func (s *UserIDMaintenanceService) usersColumns(ctx context.Context, q queryer) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'users'
  AND column_name <> 'id'
ORDER BY ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("list users columns: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		out = append(out, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *UserIDMaintenanceService) insertPlaceholderUser(
	ctx context.Context,
	e execer,
	oldUserID int64,
	newUserID int64,
	placeholderEmail string,
	columns []string,
) error {
	if len(columns) == 0 {
		return infraerrors.InternalServer("USER_ID_USERS_COLUMNS_NOT_FOUND", "users columns not found")
	}
	insertCols := make([]string, 0, len(columns)+1)
	selectExprs := make([]string, 0, len(columns)+1)
	insertCols = append(insertCols, quoteIdent("id"))
	selectExprs = append(selectExprs, "$2")
	for _, col := range columns {
		insertCols = append(insertCols, quoteIdent(col))
		if col == "email" {
			selectExprs = append(selectExprs, "$3")
			continue
		}
		selectExprs = append(selectExprs, quoteIdent(col))
	}
	query := fmt.Sprintf(
		`INSERT INTO users (%s) SELECT %s FROM users WHERE id = $1`,
		strings.Join(insertCols, ", "),
		strings.Join(selectExprs, ", "),
	)
	result, err := e.ExecContext(ctx, query, oldUserID, newUserID, placeholderEmail)
	if err != nil {
		return fmt.Errorf("insert placeholder user: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected != 1 {
		return infraerrors.NotFound("USER_ID_SOURCE_NOT_FOUND", "source user not found")
	}
	return nil
}

func (s *UserIDMaintenanceService) restoreUserEmail(ctx context.Context, e execer, newUserID int64, sourceEmail string) error {
	result, err := e.ExecContext(ctx, `UPDATE users SET email = $1 WHERE id = $2`, sourceEmail, newUserID)
	if err != nil {
		return fmt.Errorf("restore new user email: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected != 1 {
		return infraerrors.InternalServer("USER_ID_RESTORE_EMAIL_FAILED", "failed to restore user email")
	}
	return nil
}

func (s *UserIDMaintenanceService) userFKRefs(ctx context.Context, q queryer) ([]userIDMaintenanceFKRef, error) {
	rows, err := q.QueryContext(ctx, `
SELECT
  nsp.nspname AS table_schema,
  cls.relname AS table_name,
  att.attname AS column_name
FROM pg_constraint con
JOIN pg_class cls ON cls.oid = con.conrelid
JOIN pg_namespace nsp ON nsp.oid = cls.relnamespace
JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ANY(con.conkey)
WHERE con.contype = 'f'
  AND con.confrelid = 'public.users'::regclass
  AND array_length(con.conkey, 1) = 1
  AND array_length(con.confkey, 1) = 1
  AND con.confkey[1] = (
    SELECT attnum
    FROM pg_attribute
    WHERE attrelid = 'public.users'::regclass
      AND attname = 'id'
      AND NOT attisdropped
  )
ORDER BY nsp.nspname, cls.relname, att.attname`)
	if err != nil {
		return nil, fmt.Errorf("list user fk refs: %w", err)
	}
	defer rows.Close()

	var refs []userIDMaintenanceFKRef
	for rows.Next() {
		var ref userIDMaintenanceFKRef
		if err := rows.Scan(&ref.Schema, &ref.Table, &ref.Column); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}

func (s *UserIDMaintenanceService) existingNonFKRefs(ctx context.Context, q dbtx, userID int64) ([]string, error) {
	refs := userIDMaintenanceNonFKRefs()
	var found []string
	for _, ref := range refs {
		ok, err := s.columnExists(ctx, q, ref)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		query := fmt.Sprintf(
			`SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1 LIMIT 1)`,
			quoteQualifiedName(ref.Schema, ref.Table),
			quoteIdent(ref.Column),
		)
		var exists bool
		if err := q.QueryRowContext(ctx, query, userID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check non-fk target ref %s.%s.%s: %w", ref.Schema, ref.Table, ref.Column, err)
		}
		if exists {
			found = append(found, ref.Schema+"."+ref.Table+"."+ref.Column)
		}
	}
	sort.Strings(found)
	return found, nil
}

func (s *UserIDMaintenanceService) columnExists(ctx context.Context, q queryer, ref userIDMaintenanceColumnRef) (bool, error) {
	var exists bool
	if err := q.QueryRowContext(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM information_schema.columns
  WHERE table_schema = $1
    AND table_name = $2
    AND column_name = $3
)`, ref.Schema, ref.Table, ref.Column).Scan(&exists); err != nil {
		return false, fmt.Errorf("check column exists %s.%s.%s: %w", ref.Schema, ref.Table, ref.Column, err)
	}
	return exists, nil
}

func updateColumnRef(ctx context.Context, e execer, ref userIDMaintenanceColumnRef, oldUserID, newUserID int64) (int64, error) {
	query := fmt.Sprintf(
		`UPDATE %s SET %s = $1 WHERE %s = $2`,
		quoteQualifiedName(ref.Schema, ref.Table),
		quoteIdent(ref.Column),
		quoteIdent(ref.Column),
	)
	result, err := e.ExecContext(ctx, query, newUserID, oldUserID)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return affected, nil
}

func (s *UserIDMaintenanceService) collectCacheRefs(ctx context.Context, q queryer, userID int64) (userIDMaintenanceCacheRefs, error) {
	groups, err := queryUserIDMaintenanceSubscriptionGroupIDs(ctx, q, userID)
	if err != nil {
		return userIDMaintenanceCacheRefs{}, fmt.Errorf("list subscription groups before user id change: %w", err)
	}
	platforms, err := queryUserIDMaintenancePlatformQuotaPlatforms(ctx, q, userID)
	if err != nil {
		return userIDMaintenanceCacheRefs{}, fmt.Errorf("list platform quotas before user id change: %w", err)
	}
	return userIDMaintenanceCacheRefs{
		SubscriptionGroupIDs: groups,
		QuotaPlatforms:       platforms,
	}, nil
}

func (s *UserIDMaintenanceService) invalidateCaches(ctx context.Context, oldUserID, newUserID int64, refs userIDMaintenanceCacheRefs) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, oldUserID)
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, newUserID)
	}
	if s.billingCache != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		for _, uid := range []int64{oldUserID, newUserID} {
			if err := s.billingCache.InvalidateUserBalance(cacheCtx, uid); err != nil {
				slog.Warn("admin.user_id_maintenance.invalidate_balance_failed", "user_id", uid, "err", err)
			}
			for _, gid := range refs.SubscriptionGroupIDs {
				if err := s.billingCache.InvalidateSubscriptionCache(cacheCtx, uid, gid); err != nil {
					slog.Warn("admin.user_id_maintenance.invalidate_subscription_failed", "user_id", uid, "group_id", gid, "err", err)
				}
			}
			for _, platform := range refs.QuotaPlatforms {
				if err := s.billingCache.DeleteUserPlatformQuotaCache(cacheCtx, uid, platform); err != nil {
					slog.Warn("admin.user_id_maintenance.invalidate_platform_quota_failed", "user_id", uid, "platform", platform, "err", err)
				}
			}
		}
	}
}

func queryUserIDMaintenanceSubscriptionGroupIDs(ctx context.Context, q queryer, userID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT group_id FROM user_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func queryUserIDMaintenancePlatformQuotaPlatforms(ctx context.Context, q queryer, userID int64) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT platform FROM user_platform_quotas WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var platforms []string
	for rows.Next() {
		var platform string
		if err := rows.Scan(&platform); err != nil {
			return nil, err
		}
		platforms = append(platforms, platform)
	}
	return platforms, rows.Err()
}

func userIDMaintenanceNonFKRefs() []userIDMaintenanceColumnRef {
	return []userIDMaintenanceColumnRef{
		{Schema: "public", Table: "orphan_allowed_groups_audit", Column: "user_id"},
		{Schema: "public", Table: "ops_error_logs", Column: "user_id"},
		{Schema: "public", Table: "ops_error_logs", Column: "resolved_by_user_id"},
		{Schema: "public", Table: "ops_error_logs", Column: "deleted_key_owner_user_id"},
		{Schema: "public", Table: "ops_retry_attempts", Column: "requested_by_user_id"},
		{Schema: "public", Table: "ops_alert_silences", Column: "created_by"},
		{Schema: "public", Table: "usage_dashboard_hourly_users", Column: "user_id"},
		{Schema: "public", Table: "usage_dashboard_daily_users", Column: "user_id"},
		{Schema: "public", Table: "ops_system_logs", Column: "user_id"},
		{Schema: "public", Table: "ops_system_log_cleanup_audits", Column: "operator_id"},
		{Schema: "public", Table: "payment_orders", Column: "user_id"},
		{Schema: "public", Table: "auth_identity_migration_reports", Column: "resolved_by_user_id"},
		{Schema: "public", Table: "user_external_identities", Column: "user_id"},
		{Schema: "public", Table: "channel_monitors", Column: "created_by"},
		{Schema: "public", Table: "deleted_api_key_audits", Column: "user_id"},
		{Schema: "public", Table: "support_tickets", Column: "user_id"},
		{Schema: "public", Table: "support_tickets", Column: "created_by"},
		{Schema: "public", Table: "support_tickets", Column: "updated_by"},
		{Schema: "public", Table: "support_ticket_messages", Column: "author_id"},
	}
}

func quoteQualifiedName(schema, table string) string {
	return quoteIdent(schema) + "." + quoteIdent(table)
}

func quoteSequenceName(name string) string {
	parts := strings.Split(name, ".")
	for i := range parts {
		parts[i] = quoteIdent(parts[i])
	}
	return strings.Join(parts, ".")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func columnRefKey(ref userIDMaintenanceColumnRef) string {
	return ref.Schema + "." + ref.Table + "." + ref.Column
}
