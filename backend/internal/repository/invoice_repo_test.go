package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInvoiceRepositoryGetProfileMissing(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewInvoiceRepository(db)
	mock.ExpectQuery(`SELECT\s+invoice_title,\s+tax_no,\s+updated_at`).
		WithArgs(int64(7)).
		WillReturnError(sql.ErrNoRows)

	profile, err := repo.GetProfile(context.Background(), 7)
	require.NoError(t, err)
	require.Nil(t, profile)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInvoiceRepositoryCreateRequestUpsertsProfile(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	repo := NewInvoiceRepository(db)
	input := service.CreateInvoiceRequestInput{
		UserID:        7,
		InvoiceTitle:  "ACME Ltd",
		TaxNo:         "TAX-123",
		Note:          "monthly",
		RedeemCodeIDs: []int64{11},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+rc\.id,\s+rc\.code,\s+rc\.type,\s+rc\.value,\s+rc\.used_at,\s+rc\.created_at`).
		WithArgs(sqlmock.AnyArg(), input.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "type", "value", "used_at", "created_at"}).
			AddRow(int64(11), "redeem-11", "balance", float64(20), now, now))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO invoice_requests (user_id, status, invoice_title, tax_no, amount, note)")).
		WithArgs(input.UserID, service.InvoiceStatusPending, input.InvoiceTitle, input.TaxNo, float64(20), input.Note).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO invoice_profiles (user_id, invoice_title, tax_no)")).
		WithArgs(input.UserID, input.InvoiceTitle, input.TaxNo).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO invoice_request_redeem_codes (invoice_request_id, redeem_code_id, amount)")).
		WithArgs(int64(99), int64(11), float64(20)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT\s+ir\.id,`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_email", "status", "invoice_title", "tax_no", "amount", "note",
			"admin_note", "file_name", "file_path", "content_type", "file_size", "uploaded_by",
			"issued_at", "rejected_at", "created_at", "updated_at",
		}).AddRow(
			int64(99), input.UserID, "user@example.com", service.InvoiceStatusPending, input.InvoiceTitle,
			input.TaxNo, float64(20), input.Note, "", "", "", "", int64(0), nil, nil, nil, now, now,
		))
	mock.ExpectQuery(`SELECT\s+irc\.invoice_request_id,\s+rc\.id,\s+rc\.code,\s+rc\.type,\s+irc\.amount,\s+rc\.used_at,\s+rc\.created_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"invoice_request_id", "id", "code", "type", "amount", "used_at", "created_at"}).
			AddRow(int64(99), int64(11), "redeem-11", "balance", float64(20), now, now))

	invoice, err := repo.CreateRequest(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, int64(99), invoice.ID)
	require.Equal(t, input.InvoiceTitle, invoice.InvoiceTitle)
	require.Equal(t, input.TaxNo, invoice.TaxNo)
	require.Len(t, invoice.Recharges, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
