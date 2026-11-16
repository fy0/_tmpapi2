package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayObservationOriginalIdentityAndFallback(t *testing.T) {
	for _, test := range []struct{ name, header, body, want, source string }{
		{"explicit_header", "real-conversation", `{"conversation_id":"body-id","prompt_cache_key":"cache"}`, "real-conversation", "header.conversation_id"},
		{"explicit_body", "", `{"conversation_id":"body-id","prompt_cache_key":"cache"}`, "body-id", "body.conversation_id"},
		{"fallback", "", `{"prompt_cache_key":"cache-only"}`, "cache-only", "fallback.prompt_cache_key"},
		{"unknown", "", `{"input":"no stable ID"}`, "", ""},
		{"untrusted_control", "", `{"conversation_id":"bad\nidentity"}`, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(test.body))
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.RequestID, "downstream-request"))
			if test.header != "" {
				c.Request.Header.Set("conversation_id", test.header)
			}
			BindGatewayRequestObservation(c)
			CaptureGatewayRequestBody(c.Request, []byte(test.body))
			meta := GatewayRequestMetadataFromContext(c.Request.Context())
			require.Equal(t, test.want, meta.ConversationID)
			require.Equal(t, test.source, meta.ConversationIDSource)
			require.EqualValues(t, len(test.body), meta.RequestBodyBytes)
			require.Equal(t, "downstream-request", meta.DownstreamRequestID)
			require.NotNil(t, meta.RequestStartedAt)
			require.Equal(t, test.header, c.Request.Header.Get("conversation_id"), "observation never changes routing headers")
		})
	}
}

func TestGatewayObservationWSTurnSnapshotsDoNotAlias(t *testing.T) {
	c, _, _, _ := recoveryTestContext(t, `{"conversation_id":"turn-one"}`)
	c.Request.Header.Del("conversation_id")
	BeginGatewayRequestTurn(c, 1, []byte(`{"conversation_id":"turn-one"}`))
	observation := ensureGatewayObservation(c.Request)
	observation.meta.Attempts = []GatewayAttemptMetadata{{Attempt: 1, AccountID: 19, Outcome: "retrying"}}
	observation.meta.AttemptCount = 1
	observation.retries = 1
	observation.semanticOutput = true
	observation.deadline = time.Now().Add(time.Second)
	snapshot := CopyGatewayRequestObservation(c.Request.Context(), context.Background())
	BeginGatewayRequestTurn(c, 2, []byte(`{"conversation_id":"turn-two"}`))
	current := GatewayRequestMetadataFromContext(c.Request.Context())
	frozen := GatewayRequestMetadataFromContext(snapshot)
	require.Equal(t, "turn-two", current.ConversationID)
	require.Empty(t, current.Attempts)
	require.Zero(t, current.AttemptCount)
	require.Equal(t, "turn-one", frozen.ConversationID)
	require.Len(t, frozen.Attempts, 1)
	require.False(t, observation.semanticOutput)
	require.Zero(t, observation.retries)
	require.True(t, observation.deadline.IsZero())
	frozen.Attempts[0].Outcome = "mutated-copy"
	require.Equal(t, "retrying", GatewayRequestMetadataFromContext(snapshot).Attempts[0].Outcome)
	BeginGatewayRequestTurn(c, 2, []byte(`{"conversation_id":"must-not-reset"}`))
	require.Equal(t, "turn-two", GatewayRequestMetadataFromContext(c.Request.Context()).ConversationID)
}
