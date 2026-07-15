package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyService_RejectsV15AuthSnapshotAfterForkAndUpstreamFieldsMerged(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-v15", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  15,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v15 auth snapshot to be rejected after fork and upstream fields were merged")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyAuthSnapshotPreservesMergedImageGenerationFields(t *testing.T) {
	groupID := int64(9)
	redirectGroupID := int64(23)
	webSearchPrice := 0.025
	svc := &APIKeyService{}

	snapshot := svc.snapshotFromAPIKey(context.Background(), &APIKey{
		ID:      1,
		UserID:  2,
		GroupID: &groupID,
		Status:  StatusActive,
		User: &User{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Concurrency: 3,
		},
		Group: &Group{
			ID:                                      groupID,
			Platform:                                PlatformOpenAI,
			Status:                                  StatusActive,
			AllowImageGeneration:                    true,
			AllowBatchImageGeneration:               true,
			VideoRateIndependent:                    true,
			VideoRateMultiplier:                     1.25,
			ResponsesImageGenerationRedirectGroupID: &redirectGroupID,
			WebSearchPricePerCall:                   &webSearchPrice,
			PeakRateEnabled:                         true,
			PeakRateMultiplier:                      1.5,
		},
	})

	restored := svc.snapshotToAPIKey("sk-cache-contract", snapshot)
	require.NotNil(t, restored)
	require.NotNil(t, restored.Group)
	require.True(t, restored.Group.Hydrated)
	require.True(t, restored.Group.AllowImageGeneration)
	require.True(t, restored.Group.AllowBatchImageGeneration)
	require.True(t, restored.Group.VideoRateIndependent)
	require.Equal(t, 1.25, restored.Group.VideoRateMultiplier)
	require.Equal(t, &redirectGroupID, restored.Group.ResponsesImageGenerationRedirectGroupID)
	require.Equal(t, &webSearchPrice, restored.Group.WebSearchPricePerCall)
	require.True(t, restored.Group.PeakRateEnabled)
	require.Equal(t, 1.5, restored.Group.PeakRateMultiplier)
}
