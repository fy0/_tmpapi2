//go:build unit

package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCreateTestPayloadUsesNormalizedPrompt(t *testing.T) {
	payload, err := createTestPayload("claude-sonnet-4-6", "  custom Claude prompt  ")
	require.NoError(t, err)

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	require.Equal(t, "custom Claude prompt", gjson.GetBytes(body, "messages.0.content.0.text").String())
	require.Equal(t, "ephemeral", gjson.GetBytes(body, "messages.0.content.0.cache_control.type").String())
	require.Equal(t, claudeCodeSystemPrompt, gjson.GetBytes(body, "system.0.text").String())
}

func TestCreateTestPayloadDefaultsBlankPrompt(t *testing.T) {
	for _, prompt := range []string{"", " \t\r\n "} {
		payload, err := createTestPayload("claude-sonnet-4-6", prompt)
		require.NoError(t, err)

		body, err := json.Marshal(payload)
		require.NoError(t, err)
		require.Equal(t, defaultAccountTestPrompt, gjson.GetBytes(body, "messages.0.content.0.text").String())
	}
}

func TestCreateOpenAITestPayloadUsesPromptAndDefaultsBlank(t *testing.T) {
	customBody, err := json.Marshal(createOpenAITestPayload("gpt-5.4", true, "  custom OpenAI prompt  "))
	require.NoError(t, err)
	require.Equal(t, "custom OpenAI prompt", gjson.GetBytes(customBody, "input.0.content.0.text").String())
	require.False(t, gjson.GetBytes(customBody, "store").Bool())

	blankBody, err := json.Marshal(createOpenAITestPayload("gpt-5.4", false, " \n "))
	require.NoError(t, err)
	require.Equal(t, defaultAccountTestPrompt, gjson.GetBytes(blankBody, "input.0.content.0.text").String())
}

func TestAccountTestServiceClaudeAPIKeyRoutesPreservePrompt(t *testing.T) {
	for _, platform := range []string{PlatformAnthropic, PlatformAntigravity} {
		t.Run(platform, func(t *testing.T) {
			account := &Account{
				ID:          101,
				Platform:    platform,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Credentials: map[string]any{
					"api_key":  "test-key",
					"base_url": "https://anthropic.example.com",
				},
			}
			repo := &mockAccountRepoForGemini{accountsByID: map[int64]*Account{account.ID: account}}
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}}
			svc := &AccountTestService{
				accountRepo:  repo,
				httpUpstream: upstream,
				cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
					Enabled: false,
				}}},
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/101/test", nil)

			err := svc.TestAccountConnection(c, account.ID, "claude-sonnet-4-6", "  routed Claude prompt  ", AccountTestModeDefault)
			require.NoError(t, err)
			require.Equal(t, "routed Claude prompt", gjson.GetBytes(upstream.lastBody, "messages.0.content.0.text").String())
		})
	}
}

func TestAccountTestServiceBedrockPreservesPrompt(t *testing.T) {
	account := &Account{
		ID:          102,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode":  "apikey",
			"api_key":    "bedrock-test-key",
			"aws_region": "us-east-1",
		},
	}
	repo := &mockAccountRepoForGemini{accountsByID: map[int64]*Account{account.ID: account}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"content":[{"text":"ok"}]}`)),
	}}
	svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/102/test", nil)

	err := svc.TestAccountConnection(c, account.ID, "claude-sonnet-4-6", "  Bedrock prompt  ", AccountTestModeDefault)
	require.NoError(t, err)
	require.Equal(t, "Bedrock prompt", gjson.GetBytes(upstream.lastBody, "messages.0.content.0.text").String())
}

func TestAccountTestServiceVertexPreservesPrompt(t *testing.T) {
	account := &Account{
		ID:          103,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeServiceAccount,
		Concurrency: 1,
		Credentials: map[string]any{
			"service_account_json": map[string]any{
				"type":           "service_account",
				"project_id":     "test-project",
				"private_key_id": "test-key-id",
				"private_key":    "-----BEGIN PRIVATE KEY-----\nunused\n-----END PRIVATE KEY-----\n",
				"client_email":   "svc@test-project.iam.gserviceaccount.com",
			},
			"location": "us-east5",
		},
	}
	repo := &mockAccountRepoForGemini{accountsByID: map[int64]*Account{account.ID: account}}
	cache := newClaudeTokenCacheStub()
	key, err := parseVertexServiceAccountKey(account)
	require.NoError(t, err)
	cache.tokens[vertexServiceAccountCacheKey(account, key)] = "vertex-access-token"
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}}
	svc := &AccountTestService{
		accountRepo:         repo,
		claudeTokenProvider: NewClaudeTokenProvider(repo, cache, nil),
		httpUpstream:        upstream,
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/103/test", nil)

	err = svc.TestAccountConnection(c, account.ID, "claude-sonnet-4-6", "  Vertex prompt  ", AccountTestModeDefault)
	require.NoError(t, err)
	require.Equal(t, "Vertex prompt", gjson.GetBytes(upstream.lastBody, "messages.0.content.0.text").String())
}
