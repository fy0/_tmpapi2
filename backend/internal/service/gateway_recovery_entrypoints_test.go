package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func gatewayRecoveryTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 2
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
	return cfg
}

func gatewayRecoveryTestAccount() *Account {
	return &Account{ID: 87, Name: "recovery-test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 2,
		Credentials: map[string]any{"api_key": "sk-unit-test", "base_url": "http://upstream.example"}, Extra: map[string]any{}}
}

const recoveryCompletedSSE = "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_recovered\",\"model\":\"gpt-5\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":12,\"output_tokens\":3}}}\n\n"

func TestGatewayRecoveryRealHTTPEntrypoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"native", "passthrough", "chat_completions", "ws_http_bridge"} {
		t.Run(mode, func(t *testing.T) {
			upstream := &httpUpstreamSequenceRecorder{responses: []*http.Response{recoveryResponse(499, "", false), recoveryResponse(200, recoveryCompletedSSE, true)}}
			cfg := gatewayRecoveryTestConfig()
			cfg.Gateway.OpenAIWS.Enabled = false
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, cache: &stubGatewayCache{}, toolCorrector: NewCodexToolCorrector()}
			account := gatewayRecoveryTestAccount()
			if mode == "passthrough" {
				account.Extra["openai_passthrough"] = true
			}
			body := []byte(`{"model":"gpt-5","input":"hello","stream":true,"prompt_cache_key":"cache-original"}`)
			if mode == "chat_completions" {
				body = []byte(`{"model":"gpt-5","messages":[{"role":"user","content":"hello"}],"stream":true}`)
			}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body)))
			c.Request.Header.Set("conversation_id", "conversation-real-entry")
			c.Request.Header.Set("session_id", "session-original")
			BindGatewayRequestObservation(c)
			CaptureGatewayRequestBody(c.Request, body)
			var result *OpenAIForwardResult
			var err error
			switch mode {
			case "chat_completions":
				result, err = svc.ForwardAsChatCompletions(c.Request.Context(), c, account, body, "", "")
			case "ws_http_bridge":
				result, err = svc.proxyOpenAIWSHTTPBridgeTurn(c.Request.Context(), c, account, "sk-unit-test", body, len(body), "gpt-5", "", "", "", "", 1, func(message []byte) error { _, e := rec.Write(message); return e })
			default:
				result, err = svc.Forward(c.Request.Context(), c, account, body)
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, 2, upstream.callCount)
			require.Equal(t, upstream.bodies[0], upstream.bodies[1], "prepared request must be byte-identical")
			require.Equal(t, upstream.reqs[0].Header, upstream.reqs[1].Header)
			meta := GatewayRequestMetadataFromContext(c.Request.Context())
			require.Equal(t, "conversation-real-entry", meta.ConversationID)
			require.Equal(t, 2, meta.AttemptCount)
			require.Len(t, meta.Attempts, 2)
			require.Equal(t, 499, meta.Attempts[0].UpstreamStatus)
			require.Equal(t, "unknown", meta.Attempts[0].CostAccounting)
			require.Equal(t, 12, result.Usage.InputTokens)
			require.NotContains(t, rec.Body.String(), "499")
		})
	}
}

func TestGatewayRecoveryRealWSV2Entrypoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"handshake_499", "failed_before_output", "failed_after_tool"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			var mu sync.Mutex
			payloads := []map[string]any{}
			headers := []http.Header{}
			upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := calls.Add(1)
				mu.Lock()
				headers = append(headers, r.Header.Clone())
				mu.Unlock()
				if attempt == 1 && mode == "handshake_499" {
					w.WriteHeader(499)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				var payload map[string]any
				if err = conn.ReadJSON(&payload); err != nil {
					return
				}
				mu.Lock()
				payloads = append(payloads, payload)
				mu.Unlock()
				if attempt == 1 {
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.created","response":{"id":"failed-response-id"}}`))
					if mode == "failed_after_tool" {
						_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.output_item.added","item":{"id":"tool-1","type":"function_call","name":"charge","call_id":"c1","arguments":""}}`))
					}
					_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.failed","response":{"status":"failed","error":{"type":"server_error","message":"temporary"}}}`))
					return
				}
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.completed","response":{"id":"resp_recovered","model":"gpt-5","status":"completed","output":[],"usage":{"input_tokens":12,"output_tokens":3}}}`))
			}))
			defer upstream.Close()
			cfg := gatewayRecoveryTestConfig()
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, toolCorrector: NewCodexToolCorrector(), openaiWSResolver: NewOpenAIWSProtocolResolver(cfg)}
			account := gatewayRecoveryTestAccount()
			account.Credentials["base_url"] = upstream.URL
			account.Extra["responses_websockets_v2_enabled"] = true
			body := []byte(`{"model":"gpt-5","input":"hello","stream":true,"prompt_cache_key":"unchanged-cache","store":true}`)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body)))
			c.Request.Header.Set("conversation_id", "ws-conversation")
			c.Request.Header.Set("session_id", "ws-session")
			BindGatewayRequestObservation(c)
			CaptureGatewayRequestBody(c.Request, body)
			result, err := svc.Forward(c.Request.Context(), c, account, body)
			if mode == "failed_after_tool" {
				require.Error(t, err)
				require.Equal(t, int32(1), calls.Load())
				require.Contains(t, rec.Body.String(), "function_call")
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, int32(2), calls.Load())
				require.NotContains(t, rec.Body.String(), "failed-response-id")
				require.Contains(t, rec.Body.String(), "resp_recovered")
				mu.Lock()
				defer mu.Unlock()
				if len(payloads) == 2 {
					require.Equal(t, payloads[0], payloads[1])
				}
				require.Equal(t, headers[0].Get("session_id"), headers[1].Get("session_id"))
				require.Equal(t, headers[0].Get("conversation_id"), headers[1].Get("conversation_id"))
				metadata := GatewayRequestMetadataFromContext(c.Request.Context())
				require.Equal(t, 2, metadata.AttemptCount)
				require.Len(t, metadata.Attempts, 2)
			}
		})
	}
}

func TestGatewayRecoveryRealWSIngressPreservesTurnAndDropsFailedPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := gatewayRecoveryTestConfig()
	first := &openAIWSCaptureConn{events: [][]byte{[]byte(`{"type":"response.created","response":{"id":"failed-id"}}`), []byte(`{"type":"response.failed","response":{"error":{"type":"server_error","message":"temporary"}}}`)}}
	second := &openAIWSCaptureConn{events: [][]byte{[]byte(`{"type":"response.completed","response":{"id":"recovered-id","model":"gpt-5","usage":{"input_tokens":5,"output_tokens":2}}}`)}}
	dialer := &openAIWSQueueDialer{conns: []openAIWSClientConn{first, second}}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(dialer)
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: &httpUpstreamRecorder{}, cache: &stubGatewayCache{}, toolCorrector: NewCodexToolCorrector(), openaiWSResolver: NewOpenAIWSProtocolResolver(cfg), openaiWSPool: pool}
	account := gatewayRecoveryTestAccount()
	account.Extra["responses_websockets_v2_enabled"] = true
	type finished struct {
		err  error
		meta GatewayRequestMetadata
	}
	done := make(chan finished, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			done <- finished{err: err}
			return
		}
		defer conn.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_, body, err := conn.Read(ctx)
		if err != nil {
			done <- finished{err: err}
			return
		}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r.Clone(ctx)
		BindGatewayRequestObservation(c)
		err = svc.ProxyResponsesWebSocketFromClient(ctx, c, conn, account, "sk-unit-test", body, nil)
		done <- finished{err: err, meta: GatewayRequestMetadataFromContext(c.Request.Context())}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer client.CloseNow()
	body := []byte(`{"type":"response.create","model":"gpt-5","input":"hello","store":true,"prompt_cache_key":"unchanged-cache","conversation_id":"ingress-conversation"}`)
	require.NoError(t, client.Write(ctx, coderws.MessageText, body))
	_, message, err := client.Read(ctx)
	require.NoError(t, err)
	require.NotContains(t, string(message), "failed-id")
	require.Contains(t, string(message), "recovered-id")
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	select {
	case result := <-done:
		require.NoError(t, result.err)
		require.Equal(t, 2, result.meta.AttemptCount)
		require.Equal(t, "ingress-conversation", result.meta.ConversationID)
		require.Len(t, result.meta.Attempts, 2)
		require.Equal(t, 101, result.meta.Attempts[0].UpstreamStatus)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.Equal(t, 2, dialer.DialCount())
	first.mu.Lock()
	firstJSON, _ := json.Marshal(first.lastWrite)
	first.mu.Unlock()
	second.mu.Lock()
	secondJSON, _ := json.Marshal(second.lastWrite)
	second.mu.Unlock()
	require.JSONEq(t, string(firstJSON), string(secondJSON))
}

func TestGatewayRecoveryWSSafetyAndSharedBudget(t *testing.T) {
	failed := []byte(`{"type":"response.failed","response":{"error":{"type":"server_error"}}}`)
	for _, mode := range []string{"strict_affinity", "tool_started", "ordinary"} {
		t.Run(mode, func(t *testing.T) {
			c, _, account, _ := recoveryTestContext(t, `{"input":"x"}`)
			a := beginGatewayWSRecoveryAttempt(c, account, []byte(`{"input":"x"}`), mode == "strict_affinity")
			if mode == "tool_started" {
				a.observeOutput([]byte(`{"type":"response.output_item.added","item":{"type":"function_call"}}`), "response.output_item.added")
			}
			err := a.recover(nil, nil, failed, false)
			var retry *gatewayWSRecoveryRetry
			require.Equal(t, mode == "ordinary", errors.As(err, &retry))
			if mode != "ordinary" {
				var final *UpstreamFailoverError
				require.ErrorAs(t, err, &final)
				require.False(t, final.ShouldRetryNextAccount())
			}
		})
	}
}
