//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release            *GitHubRelease
	fetchLatestCalls   int
	downloadFileCalls  int
	fetchChecksumCalls int
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	s.fetchLatestCalls++
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	s.downloadFileCalls++
	panic("DownloadFile should not be called when updates are disabled")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	s.fetchChecksumCalls++
	panic("FetchChecksumFile should not be called when updates are disabled")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	githubClient := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v9.9.9",
			Name:    "v9.9.9",
		},
	}
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		githubClient,
		"9.9.9",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
	require.Zero(t, githubClient.fetchLatestCalls)
	require.Zero(t, githubClient.downloadFileCalls)
	require.Zero(t, githubClient.fetchChecksumCalls)
}

func TestUpdateServiceCheckUpdatePinnedAndOffline(t *testing.T) {
	githubClient := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v9.9.9",
			Name:    "v9.9.9",
		},
	}
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		githubClient,
		"9.9.9",
		"release",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "v0.1.138", info.CurrentVersion)
	require.Equal(t, "v0.1.138", info.LatestVersion)
	require.False(t, info.HasUpdate)
	require.Nil(t, info.ReleaseInfo)
	require.False(t, info.Cached)
	require.Equal(t, "release", info.BuildType)
	require.Zero(t, githubClient.fetchLatestCalls)
}
