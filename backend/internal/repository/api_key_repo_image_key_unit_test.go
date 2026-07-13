package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryImageKeyLifecycle(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "image-key-lifecycle@test.com")

	group, err := client.Group.Create().
		SetName("image-key-lifecycle").
		SetPlatform(service.PlatformOpenAI).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		SetAllowImageGeneration(true).
		Save(ctx)
	require.NoError(t, err)

	first := &service.APIKey{
		UserID:     user.ID,
		Key:        "sk-image-key-first",
		Name:       "First image key",
		GroupID:    &group.ID,
		Status:     service.StatusActive,
		IsImageKey: true,
	}
	second := &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-image-key-second",
		Name:    "Second image key",
		GroupID: &group.ID,
		Status:  service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Create(ctx, second))

	candidate, err := repo.GetImageKeyCandidate(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, candidate)
	require.Equal(t, first.ID, candidate.ID)
	require.True(t, candidate.IsImageKey)

	require.NoError(t, repo.SetImageKey(ctx, user.ID, second.ID))
	candidate, err = repo.GetImageKeyCandidate(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, candidate)
	require.Equal(t, second.ID, candidate.ID)
	require.True(t, candidate.IsImageKey)

	keyService := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil)
	resolved, err := keyService.ResolveImageKey(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Equal(t, second.ID, resolved.ID)

	previous, err := repo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.False(t, previous.IsImageKey)

	candidate.IsImageKey = false
	require.NoError(t, repo.Update(ctx, candidate))
	updated, err := repo.GetByID(ctx, second.ID)
	require.NoError(t, err)
	require.False(t, updated.IsImageKey)
}
