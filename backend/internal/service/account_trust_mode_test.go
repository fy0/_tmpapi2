//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func trustedAccountForTest() *Account {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	return &Account{
		ID:                      4101,
		Platform:                PlatformAnthropic,
		Type:                    AccountTypeAPIKey,
		Status:                  StatusActive,
		Schedulable:             true,
		ExpiresAt:               &past,
		AutoPauseOnExpired:      true,
		RateLimitResetAt:        &future,
		OverloadUntil:           &future,
		TempUnschedulableUntil:  &future,
		TempUnschedulableReason: "automatic cooldown",
		Credentials: map[string]any{
			"temp_unschedulable_enabled": true,
		},
		Extra: map[string]any{
			AccountTrustModeExtraKey: true,
			"quota_limit":            10.0,
			"quota_used":             10.0,
			modelRateLimitsKey: map[string]any{
				"claude-sonnet-4-5": map[string]any{
					"rate_limit_reset_at": future.Format(time.RFC3339),
				},
			},
		},
	}
}

func TestAccountTrustModeBypassesAutomaticSchedulingInterruptions(t *testing.T) {
	account := trustedAccountForTest()

	require.True(t, account.IsTrustModeEnabled())
	require.True(t, account.IsSchedulable())
	require.True(t, account.IsSchedulableForModel("claude-sonnet-4-5"))
	require.False(t, account.IsRateLimited())
	require.False(t, account.IsOverloaded())
	require.False(t, account.IsTempUnschedulableEnabled())
	require.Zero(t, account.GetRateLimitRemainingTimeWithContext(context.Background(), "claude-sonnet-4-5"))

	account.Status = StatusError
	require.True(t, account.IsSchedulable(), "a stale automatic error must not interrupt a trusted account")
}

func TestAccountTrustModePreservesAdministratorControls(t *testing.T) {
	t.Run("manual pause", func(t *testing.T) {
		account := trustedAccountForTest()
		account.Schedulable = false
		require.False(t, account.IsSchedulable())
	})

	t.Run("disabled status", func(t *testing.T) {
		account := trustedAccountForTest()
		account.Status = StatusDisabled
		require.False(t, account.IsSchedulable())
	})

	t.Run("inactive status", func(t *testing.T) {
		account := trustedAccountForTest()
		account.Status = "inactive"
		require.False(t, account.IsSchedulable())
	})
}

type trustModeMutationRepo struct {
	AccountRepository
	setErrorCalls     int
	setRateCalls      int
	setModelRateCalls int
	setOverloadCalls  int
	setTempCalls      int
}

type trustModeRuntimeBlocker struct {
	blockCalls int
	clearedIDs []int64
}

func (b *trustModeRuntimeBlocker) BlockAccountScheduling(*Account, time.Time, string) {
	b.blockCalls++
}

func (b *trustModeRuntimeBlocker) ClearAccountSchedulingBlock(accountID int64) {
	b.clearedIDs = append(b.clearedIDs, accountID)
}

type trustModeGatewayCache struct {
	GatewayCache
	deleteCalls int
}

func (c *trustModeGatewayCache) DeleteSessionAccountID(context.Context, int64, string) error {
	c.deleteCalls++
	return nil
}

func (r *trustModeMutationRepo) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

func (r *trustModeMutationRepo) SetRateLimited(context.Context, int64, time.Time) error {
	r.setRateCalls++
	return nil
}

func (r *trustModeMutationRepo) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	r.setModelRateCalls++
	return nil
}

func (r *trustModeMutationRepo) SetOverloaded(context.Context, int64, time.Time) error {
	r.setOverloadCalls++
	return nil
}

func (r *trustModeMutationRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.setTempCalls++
	return nil
}

func TestRateLimitServiceTrustModeSkipsAutomaticMutations(t *testing.T) {
	repo := &trustModeMutationRepo{}
	blocker := &trustModeRuntimeBlocker{}
	svc := &RateLimitService{accountRepo: repo, runtimeBlocker: blocker}
	account := trustedAccountForTest()
	account.Platform = PlatformOpenAI
	account.Type = AccountTypeOAuth
	account.Credentials["temp_unschedulable_rules"] = []any{
		map[string]any{
			"error_code":       http.StatusBadRequest,
			"keywords":         []any{"bad request"},
			"duration_minutes": 10,
		},
	}

	for _, statusCode := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		529,
	} {
		require.False(t, svc.HandleUpstreamError(
			context.Background(), account, statusCode, http.Header{}, []byte(`{"error":"failure"}`), "gpt-5.4",
		))
	}
	require.Equal(t, ErrorPolicySkipped, svc.CheckErrorPolicy(
		context.Background(), account, http.StatusBadRequest, []byte("bad request"), "gpt-5.4",
	))
	require.False(t, svc.HandleTempUnschedulable(
		context.Background(), account, http.StatusBadRequest, []byte("bad request"), "gpt-5.4",
	))
	require.False(t, svc.HandleOpenAIImageRateLimit(
		context.Background(), account, http.StatusTooManyRequests, http.Header{}, []byte("rate limited"),
	))
	require.False(t, svc.HandleUpstreamModelNotFound(
		context.Background(), account, "gpt-5.4", http.StatusNotFound, []byte("model not found"),
	))
	require.False(t, svc.HandleStreamTimeout(context.Background(), account, "gpt-5.4"))

	require.Zero(t, repo.setErrorCalls)
	require.Zero(t, repo.setRateCalls)
	require.Zero(t, repo.setModelRateCalls)
	require.Zero(t, repo.setOverloadCalls)
	require.Zero(t, repo.setTempCalls)
	require.Zero(t, blocker.blockCalls)
}

func TestAntigravityTrustModeKeepsStickySessionAndSkipsModelCooldown(t *testing.T) {
	cache := &trustModeGatewayCache{}
	repo := &trustModeMutationRepo{}
	svc := &AntigravityGatewayService{cache: cache}
	account := trustedAccountForTest()
	account.Platform = PlatformAntigravity

	handled := svc.setAntigravityModelRateLimits(
		context.Background(), repo, account, "gemini-3-flash", "[test]", http.StatusTooManyRequests, time.Now().Add(time.Minute), false,
	)
	svc.clearStickySession(context.Background(), account, 7, "sticky-session")

	require.True(t, handled)
	require.Zero(t, repo.setModelRateCalls)
	require.Zero(t, cache.deleteCalls)
}

type trustModeAdminRepo struct {
	AccountRepository
	account                  *Account
	updateCalls              int
	clearRateLimitCalls      int
	clearAntigravityCalls    int
	clearModelRateLimitCalls int
	clearTempUnschedCalls    int
}

func (r *trustModeAdminRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *trustModeAdminRepo) Update(_ context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func (r *trustModeAdminRepo) ClearRateLimit(context.Context, int64) error {
	r.clearRateLimitCalls++
	r.account.RateLimitedAt = nil
	r.account.RateLimitResetAt = nil
	r.account.OverloadUntil = nil
	return nil
}

func (r *trustModeAdminRepo) ClearAntigravityQuotaScopes(context.Context, int64) error {
	r.clearAntigravityCalls++
	delete(r.account.Extra, "antigravity_quota_scopes")
	return nil
}

func (r *trustModeAdminRepo) ClearModelRateLimits(context.Context, int64) error {
	r.clearModelRateLimitCalls++
	delete(r.account.Extra, modelRateLimitsKey)
	return nil
}

func (r *trustModeAdminRepo) ClearTempUnschedulable(context.Context, int64) error {
	r.clearTempUnschedCalls++
	r.account.TempUnschedulableUntil = nil
	r.account.TempUnschedulableReason = ""
	return nil
}

func TestUpdateAccountEnablingTrustModeRecoversAutomaticError(t *testing.T) {
	account := trustedAccountForTest()
	account.Status = StatusError
	account.Schedulable = false
	account.ErrorMessage = "automatic upstream failure"
	account.Extra[AccountTrustModeExtraKey] = false
	account.Extra["antigravity_quota_scopes"] = map[string]any{"default": true}
	repo := &trustModeAdminRepo{account: account}
	blocker := &trustModeRuntimeBlocker{}
	svc := &adminServiceImpl{accountRepo: repo, runtimeBlocker: blocker}

	updated, err := svc.UpdateAccount(context.Background(), account.ID, &UpdateAccountInput{
		Status: StatusError,
		Extra: map[string]any{
			AccountTrustModeExtraKey: true,
			modelRateLimitsKey:       account.Extra[modelRateLimitsKey],
		},
	})

	require.NoError(t, err)
	require.Equal(t, StatusActive, updated.Status)
	require.True(t, updated.Schedulable)
	require.Empty(t, updated.ErrorMessage)
	require.True(t, updated.IsTrustModeEnabled())
	require.Nil(t, updated.RateLimitResetAt)
	require.Nil(t, updated.OverloadUntil)
	require.Nil(t, updated.TempUnschedulableUntil)
	require.NotContains(t, updated.Extra, modelRateLimitsKey)
	require.NotContains(t, updated.Extra, "antigravity_quota_scopes")
	require.Equal(t, 1, repo.updateCalls)
	require.Equal(t, 1, repo.clearRateLimitCalls)
	require.Equal(t, 1, repo.clearAntigravityCalls)
	require.Equal(t, 1, repo.clearModelRateLimitCalls)
	require.Equal(t, 1, repo.clearTempUnschedCalls)
	require.Equal(t, []int64{account.ID}, blocker.clearedIDs)
}

func TestUpdateAccountEnablingTrustModePreservesManualPauseAndDisable(t *testing.T) {
	tests := []struct {
		name            string
		previousStatus  string
		submittedStatus string
		schedulable     bool
	}{
		{name: "manual pause", previousStatus: StatusActive, submittedStatus: StatusActive, schedulable: false},
		{name: "inactive", previousStatus: StatusError, submittedStatus: "inactive", schedulable: false},
		{name: "disabled", previousStatus: StatusError, submittedStatus: StatusDisabled, schedulable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiredAt := time.Now().Add(-time.Hour)
			account := &Account{
				ID:          4201,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeOAuth,
				Status:      tt.previousStatus,
				Schedulable: tt.schedulable,
				Extra:       map[string]any{AccountTrustModeExtraKey: false},
			}
			if tt.name == "manual pause" {
				account.AutoPauseOnExpired = true
				account.ExpiresAt = &expiredAt
			}
			repo := &trustModeAdminRepo{account: account}
			svc := &adminServiceImpl{accountRepo: repo}

			updated, err := svc.UpdateAccount(context.Background(), account.ID, &UpdateAccountInput{
				Status: tt.submittedStatus,
				Extra:  map[string]any{AccountTrustModeExtraKey: true},
			})

			require.NoError(t, err)
			require.Equal(t, tt.submittedStatus, updated.Status)
			require.Equal(t, tt.schedulable, updated.Schedulable)
		})
	}
}

func TestUpdateAccountDisablingTrustModeStartsWithCleanAutomaticState(t *testing.T) {
	account := trustedAccountForTest()
	repo := &trustModeAdminRepo{account: account}
	blocker := &trustModeRuntimeBlocker{}
	svc := &adminServiceImpl{accountRepo: repo, runtimeBlocker: blocker}

	updated, err := svc.UpdateAccount(context.Background(), account.ID, &UpdateAccountInput{
		Status: StatusActive,
		Extra:  map[string]any{AccountTrustModeExtraKey: false},
	})

	require.NoError(t, err)
	require.False(t, updated.IsTrustModeEnabled())
	require.Equal(t, StatusActive, updated.Status)
	require.True(t, updated.Schedulable)
	require.Nil(t, updated.RateLimitResetAt)
	require.Nil(t, updated.OverloadUntil)
	require.Nil(t, updated.TempUnschedulableUntil)
	require.Equal(t, []int64{account.ID}, blocker.clearedIDs)
}

func TestUpdateAccountRejectsInvalidTrustModeValue(t *testing.T) {
	repo := &trustModeAdminRepo{account: &Account{
		ID:          4301,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.UpdateAccount(context.Background(), repo.account.ID, &UpdateAccountInput{
		Extra: map[string]any{AccountTrustModeExtraKey: "true"},
	})

	require.Error(t, err)
	require.Zero(t, repo.updateCalls)
}
