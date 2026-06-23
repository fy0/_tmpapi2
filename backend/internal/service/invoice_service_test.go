package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type invoiceServiceSettingRepoStub struct {
	values map[string]string
}

func (s *invoiceServiceSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *invoiceServiceSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *invoiceServiceSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *invoiceServiceSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *invoiceServiceSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *invoiceServiceSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *invoiceServiceSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type invoiceRepoStub struct {
	createInput CreateInvoiceRequestInput
	profile     *InvoiceProfile
	summary     *InvoiceSummary
}

func (r *invoiceRepoStub) GetSummary(context.Context, int64) (*InvoiceSummary, error) {
	if r.summary != nil {
		return r.summary, nil
	}
	return &InvoiceSummary{}, nil
}

func (r *invoiceRepoStub) GetProfile(context.Context, int64) (*InvoiceProfile, error) {
	return r.profile, nil
}

func (r *invoiceRepoStub) ListAvailableRecharges(context.Context, int64) ([]InvoiceRecharge, error) {
	return nil, nil
}

func (r *invoiceRepoStub) CreateRequest(_ context.Context, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	r.createInput = input
	return &InvoiceRequest{
		ID:           1,
		UserID:       input.UserID,
		Status:       InvoiceStatusPending,
		InvoiceTitle: input.InvoiceTitle,
		Amount:       50,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (r *invoiceRepoStub) ListUserInvoices(context.Context, int64, InvoiceListParams) ([]InvoiceRequest, int64, error) {
	return nil, 0, nil
}

func (r *invoiceRepoStub) ListAdminInvoices(context.Context, InvoiceListParams) ([]InvoiceRequest, int64, error) {
	return nil, 0, nil
}

func (r *invoiceRepoStub) GetByID(context.Context, int64) (*InvoiceRequest, error) {
	return nil, ErrInvoiceNotFound
}

func (r *invoiceRepoStub) GetByIDForUser(context.Context, int64, int64) (*InvoiceRequest, error) {
	return nil, ErrInvoiceNotFound
}

func (r *invoiceRepoStub) UpdateIssuedFile(context.Context, int64, int64, InvoiceStoredFile) (*InvoiceRequest, error) {
	return nil, ErrInvoiceNotFound
}

func (r *invoiceRepoStub) ClearIssuedFile(context.Context, int64) (*InvoiceRequest, error) {
	return nil, ErrInvoiceNotFound
}

func TestInvoiceServiceCreateRequestPassesAmountLimits(t *testing.T) {
	repo := &invoiceRepoStub{}
	settings := &invoiceServiceSettingRepoStub{values: map[string]string{
		SettingMinInvoiceAmount: "25.5",
		SettingMaxInvoiceAmount: "200.5",
	}}
	svc := NewInvoiceService(repo, settings)

	_, err := svc.CreateRequest(context.Background(), CreateInvoiceRequestInput{
		UserID:        7,
		InvoiceTitle:  "ACME",
		RedeemCodeIDs: []int64{1, 2},
	})
	if err != nil {
		t.Fatalf("CreateRequest returned error: %v", err)
	}
	if repo.createInput.MinAmount != 25.5 {
		t.Fatalf("MinAmount = %v, want 25.5", repo.createInput.MinAmount)
	}
	if repo.createInput.MaxAmount != 200.5 {
		t.Fatalf("MaxAmount = %v, want 200.5", repo.createInput.MaxAmount)
	}
}

func TestInvoiceServiceGetSummaryIncludesAmountLimits(t *testing.T) {
	repo := &invoiceRepoStub{summary: &InvoiceSummary{AvailableAmount: 10}}
	settings := &invoiceServiceSettingRepoStub{values: map[string]string{
		SettingMinInvoiceAmount: "30",
		SettingMaxInvoiceAmount: "300",
	}}
	svc := NewInvoiceService(repo, settings)

	summary, err := svc.GetSummary(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetSummary returned error: %v", err)
	}
	if summary.MinInvoiceAmount != 30 {
		t.Fatalf("MinInvoiceAmount = %v, want 30", summary.MinInvoiceAmount)
	}
	if summary.MaxInvoiceAmount != 300 {
		t.Fatalf("MaxInvoiceAmount = %v, want 300", summary.MaxInvoiceAmount)
	}
}

func TestInvoiceServiceGetProfileReturnsEmptyProfileWhenMissing(t *testing.T) {
	svc := NewInvoiceService(&invoiceRepoStub{}, nil)

	profile, err := svc.GetProfile(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}
	if profile.InvoiceTitle != "" || profile.TaxNo != "" || profile.UpdatedAt != nil {
		t.Fatalf("profile = %#v, want empty profile", profile)
	}
}

func TestInvoiceServiceUpdateInvoiceSettings(t *testing.T) {
	settings := &invoiceServiceSettingRepoStub{values: map[string]string{}}
	svc := NewInvoiceService(&invoiceRepoStub{}, settings)

	updated, err := svc.UpdateInvoiceSettings(context.Background(), InvoiceSettings{
		MinInvoiceAmount: 12.345,
		MaxInvoiceAmount: 99.999,
	})
	if err != nil {
		t.Fatalf("UpdateInvoiceSettings returned error: %v", err)
	}
	if updated.MinInvoiceAmount != 12.35 {
		t.Fatalf("MinInvoiceAmount = %v, want 12.35", updated.MinInvoiceAmount)
	}
	if settings.values[SettingMinInvoiceAmount] != "12.35" {
		t.Fatalf("stored setting = %q, want 12.35", settings.values[SettingMinInvoiceAmount])
	}
	if updated.MaxInvoiceAmount != 100 {
		t.Fatalf("MaxInvoiceAmount = %v, want 100", updated.MaxInvoiceAmount)
	}
	if settings.values[SettingMaxInvoiceAmount] != "100" {
		t.Fatalf("stored max setting = %q, want 100", settings.values[SettingMaxInvoiceAmount])
	}

	if _, err := svc.UpdateInvoiceSettings(context.Background(), InvoiceSettings{MinInvoiceAmount: -1}); err == nil {
		t.Fatal("UpdateInvoiceSettings with negative amount should fail")
	}
	if _, err := svc.UpdateInvoiceSettings(context.Background(), InvoiceSettings{MaxInvoiceAmount: -1}); err == nil {
		t.Fatal("UpdateInvoiceSettings with negative max amount should fail")
	}
	if _, err := svc.UpdateInvoiceSettings(context.Background(), InvoiceSettings{MinInvoiceAmount: 50, MaxInvoiceAmount: 10}); infraerrors.Reason(err) != "INVOICE_AMOUNT_RANGE_INVALID" {
		t.Fatalf("UpdateInvoiceSettings invalid range reason = %q, want INVOICE_AMOUNT_RANGE_INVALID", infraerrors.Reason(err))
	}
}
