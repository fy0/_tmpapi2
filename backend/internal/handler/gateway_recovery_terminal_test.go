package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Services return the decided error without committing a response. Verify the
// client boundary still writes the configured error in each protocol, without
// exposing buffered failure metadata or rematching the rewritten status.
func TestGatewayRecoveryTerminalReachesHTTPClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, anthropic := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			for _, status := range []int{http.StatusBadRequest, http.StatusServiceUnavailable} {
				t.Run(fmt.Sprintf("anthropic=%t/stream=%t/status=%d", anthropic, stream, status), func(t *testing.T) {
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
					if stream {
						c.Header("Content-Type", "text/event-stream")
						c.Writer.WriteHeaderNow()
					}
					terminal := &service.UpstreamFailoverError{
						StatusCode: http.StatusBadGateway, RecoveryHandled: true,
						ClientStatusCode: status, ClientMessage: "decided client error",
						NextAccountAction: service.NextAccountStop,
						ResponseHeaders:   http.Header{"Retry-After": []string{"1"}},
						ResponseBody:      []byte(`{"response":{"id":"failed-attempt-must-not-leak"}}`),
					}
					h := &OpenAIGatewayHandler{}
					if anthropic {
						h.handleAnthropicFailoverExhausted(c, terminal, stream)
					} else {
						h.handleFailoverExhausted(c, terminal, stream)
					}
					require.False(t, terminal.ShouldRetryNextAccount())
					require.Contains(t, rec.Body.String(), "decided client error")
					require.NotContains(t, rec.Body.String(), "failed-attempt-must-not-leak")
					if stream {
						require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
					} else {
						require.Equal(t, "1", rec.Header().Get("Retry-After"))
						require.Equal(t, status, rec.Code)
						require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
					}
				})
			}
		}
	}
}
