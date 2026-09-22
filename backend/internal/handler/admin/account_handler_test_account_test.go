package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type recordingAccountTestService struct {
	accountID int64
	modelID   string
	prompt    string
	mode      string
}

func (s *recordingAccountTestService) TestAccountConnection(_ *gin.Context, accountID int64, modelID string, prompt string, mode string, opts ...service.AccountTestOptions) error {
	s.accountID = accountID
	s.modelID = modelID
	s.prompt = prompt
	s.mode = mode
	return nil
}

func (s *recordingAccountTestService) ProbeOpenAIAPIKeyResponsesSupport(context.Context, int64) {}

func (s *recordingAccountTestService) FetchUpstreamSupportedModels(context.Context, *service.Account) ([]string, error) {
	return nil, nil
}

func TestAccountHandlerTestPassesPromptToAccountTestService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testService := &recordingAccountTestService{}
	handler := &AccountHandler{accountTestService: testService}
	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/test", handler.Test)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/accounts/37/test",
		bytes.NewBufferString(`{"model_id":"claude-sonnet-4-6","prompt":"只回答 37 - 14 的结果","mode":"default"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, int64(37), testService.accountID)
	require.Equal(t, "claude-sonnet-4-6", testService.modelID)
	require.Equal(t, "只回答 37 - 14 的结果", testService.prompt)
	require.Equal(t, service.AccountTestModeDefault, testService.mode)
}

func TestNewAccountHandlerPreservesNilAccountTestService(t *testing.T) {
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Nil(t, handler.accountTestService)
}
