package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
)

func recoveryTestContext(t *testing.T, body string, rules ...*model.ErrorPassthroughRule) (*gin.Context, *http.Request, *Account, gatewayRecoveryOptions) {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	c.Request.Header.Set("Conversation_id", "conversation-test")
	BindGatewayRequestObservation(c)
	CaptureGatewayRequestBody(c.Request, []byte(body))
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, "https://upstream.invalid/v1/responses", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-secret-do-not-log")
	request.Header.Set("session_id", "unchanged-session")
	if len(rules) > 0 {
		svc := &ErrorPassthroughService{}
		svc.setLocalCache(rules)
		BindErrorPassthroughService(c, svc)
	}
	account := &Account{ID: 19, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	options := defaultGatewayRecoveryOptions()
	options.delays = []time.Duration{time.Nanosecond, time.Nanosecond}
	options.jitter = false
	return c, request, account, options
}

func recoveryResponse(status int, body string, sse bool) *http.Response {
	headers := make(http.Header)
	if sse {
		headers.Set("Content-Type", "text/event-stream")
	} else {
		headers.Set("Content-Type", "application/json")
	}
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}
}

func recoveryRule(action string, code int) *model.ErrorPassthroughRule {
	return &model.ErrorPassthroughRule{ID: 7, Name: "recovery-test", Enabled: true, Action: action, ErrorCodes: []int{code}, MatchMode: model.MatchModeAny, PassthroughCode: true, PassthroughBody: true}
}

func TestGatewayRecoveryHTTP499DefaultAndRequestIdentity(t *testing.T) {
	for _, firstBody := range []string{"", "<html>499 upstream closed</html>", `{"error":{"message":"upstream closed"}}`} {
		t.Run(firstBody, func(t *testing.T) {
			body := `{"model":"test-model","input":"do not log this","prompt_cache_key":"cache-original","previous_response_id":"resp-original"}`
			c, request, account, options := recoveryTestContext(t, body)
			calls := 0
			response, err := recoverOpenAIUpstream(c, request, account, func(req *http.Request) (*http.Response, error) {
				calls++
				got, _ := io.ReadAll(req.Body)
				_ = req.Body.Close()
				if string(got) != body || !reflect.DeepEqual(req.Header, request.Header) {
					t.Fatal("recovery changed body or headers")
				}
				if calls == 1 {
					return recoveryResponse(499, firstBody, false), nil
				}
				return recoveryResponse(200, `{"id":"success","output":[]}`, false), nil
			}, options)
			if err != nil || calls != 2 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			_ = response.Body.Close()
			meta := GatewayRequestMetadataFromContext(c.Request.Context())
			if meta.AttemptCount != 2 || len(meta.Attempts) != 2 || meta.Attempts[0].UpstreamStatus != 499 {
				t.Fatalf("metadata: %+v", meta)
			}
			if meta.Attempts[0].CostAccounting != "unknown" {
				t.Fatal("failed upstream cost must not be invented as zero")
			}
			encoded := GatewayRequestMetadataJSON(meta)
			if encoded == nil || strings.Contains(*encoded, "test-secret") || strings.Contains(*encoded, "do not log this") {
				t.Fatal("unsafe observation contents")
			}
		})
	}
}

func TestGatewayRecoveryRulesAndExhaustion(t *testing.T) {
	code := 503
	message := "configured failure"
	rewrite := recoveryRule(model.ErrorActionRewrite, 499)
	rewrite.PassthroughCode, rewrite.PassthroughBody, rewrite.ResponseCode, rewrite.CustomMessage = false, false, &code, &message
	legacy := *rewrite
	legacy.Action = ""
	priorityPass := recoveryRule(model.ErrorActionPassthrough, 499)
	priorityPass.Priority = -10
	unrelated := recoveryRule(model.ErrorActionPassthrough, 418)
	for _, test := range []struct {
		name          string
		rules         []*model.ErrorPassthroughRule
		calls, status int
		message       string
	}{
		{"default", nil, 3, 503, ""},
		{"unrelated_rule", []*model.ErrorPassthroughRule{unrelated}, 3, 503, ""},
		{"passthrough_empty_body", []*model.ErrorPassthroughRule{recoveryRule(model.ErrorActionPassthrough, 499)}, 1, 499, ""},
		{"rewrite_no_recursive_503_retry", []*model.ErrorPassthroughRule{rewrite, recoveryRule(model.ErrorActionRetry, 503)}, 1, 503, message},
		{"legacy_rewrite_no_retry", []*model.ErrorPassthroughRule{&legacy}, 1, 503, message},
		{"explicit_retry", []*model.ErrorPassthroughRule{recoveryRule(model.ErrorActionRetry, 499)}, 3, 503, ""},
		{"first_priority_match_only", []*model.ErrorPassthroughRule{rewrite, priorityPass}, 1, 499, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, request, account, options := recoveryTestContext(t, `{"model":"m","input":"x"}`, test.rules...)
			calls := 0
			_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) { calls++; return recoveryResponse(499, "", false), nil }, options)
			var final *UpstreamFailoverError
			if !errors.As(err, &final) || !final.RecoveryHandled || final.ShouldRetryNextAccount() {
				t.Fatalf("expected terminal handled error, got %v", err)
			}
			if calls != test.calls || final.ClientStatusCode != test.status {
				t.Fatalf("calls=%d status=%d", calls, final.ClientStatusCode)
			}
			if test.message != "" && final.ClientMessage != test.message {
				t.Fatalf("message=%q", final.ClientMessage)
			}
			if test.calls == 3 && final.ResponseHeaders.Get("Retry-After") != "1" {
				t.Fatal("missing exhaustion retry hint")
			}
			if c.Writer.Written() {
				t.Fatal("decision committed downstream prematurely")
			}
		})
	}
}

func TestGatewayRecoveryTransientTransportAndStatus(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		err    error
	}{
		{"502", 502, nil}, {"503", 503, nil}, {"504", 504, nil}, {"524", 524, nil},
		{"reset", 0, syscall.ECONNRESET}, {"unexpected_eof", 0, io.ErrUnexpectedEOF},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
			calls := 0
			resp, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if test.err != nil {
						return nil, test.err
					}
					return recoveryResponse(test.status, "", false), nil
				}
				return recoveryResponse(200, `{}`, false), nil
			}, options)
			if err != nil || calls != 2 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			_ = resp.Body.Close()
			meta := GatewayRequestMetadataFromContext(c.Request.Context())
			if meta.Attempts[0].UpstreamStatus != test.status {
				t.Fatal("original status lost")
			}
		})
	}
}

func TestGatewayRecoveryStreamPrefixAndSemanticBarrier(t *testing.T) {
	created := "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"failed-id\"}}\n\n"
	failed := "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"error\":{\"type\":\"server_error\",\"message\":\"temporary\"}}}\n\n"
	delta := "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"visible\"}\n\n"
	tool := "event: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"name\":\"charge\",\"arguments\":\"\"}}\n\n"
	for _, test := range []struct {
		name, first string
		retry       bool
	}{
		{"metadata_then_transient", created + failed, true},
		{"text_then_failed", created + delta + failed, false},
		{"tool_start_then_failed", created + tool + failed, false},
		{"reasoning_then_failed", created + strings.ReplaceAll(delta, "output_text", "reasoning_text") + failed, false},
		{"metadata_then_truncated", created, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, request, account, options := recoveryTestContext(t, `{"model":"m","stream":true}`)
			calls := 0
			resp, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return recoveryResponse(200, test.first, true), nil
				}
				return recoveryResponse(200, strings.ReplaceAll(created, "failed-id", "success-id")+delta, true), nil
			}, options)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if test.retry {
				if calls != 2 || bytes.Contains(got, []byte("failed-id")) {
					t.Fatalf("failed metadata leaked: calls=%d body=%s", calls, got)
				}
			} else if calls != 1 || string(got) != test.first {
				t.Fatalf("replayed or changed committed output: calls=%d", calls)
			}
		})
	}
}

func TestGatewayRecoveryDoesNotRetryDeterministicStreamFailure(t *testing.T) {
	for _, typ := range []string{"invalid_request_error", "authentication_error", "content_policy_violation", "unknown_failure"} {
		t.Run(typ, func(t *testing.T) {
			c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
			calls := 0
			resp, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
				calls++
				return recoveryResponse(200, `data: {"type":"response.failed","response":{"error":{"type":"`+typ+`","message":"not 499, just text"}}}`+"\n\n", true), nil
			}, options)
			if calls != 1 {
				t.Fatalf("deterministic error replayed: calls=%d", calls)
			}
			if typ == "authentication_error" {
				// Keep credential side effects with the original caller, without
				// losing the event or performing a default recovery retry.
				if err != nil || resp == nil {
					t.Fatalf("missing legacy credential response: %v", err)
				}
				body, readErr := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if readErr != nil || !strings.Contains(string(body), typ) {
					t.Fatalf("credential event lost: %s %v", body, readErr)
				}
			} else if err == nil {
				t.Fatal("expected terminal non-retryable error")
			}
		})
	}
}

func TestGatewayRecoveryCancellationAndZeroBudget(t *testing.T) {
	t.Run("already_cancelled", func(t *testing.T) {
		c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
		ctx, cancel := context.WithCancel(c.Request.Context())
		cancel()
		c.Request = c.Request.WithContext(ctx)
		calls := 0
		_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) { calls++; return nil, io.EOF }, options)
		if !errors.Is(err, context.Canceled) || calls != 0 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
	t.Run("cancel_during_backoff", func(t *testing.T) {
		c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		options.delays = []time.Duration{time.Second}
		calls := 0
		_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
			calls++
			time.AfterFunc(5*time.Millisecond, cancel)
			return recoveryResponse(499, "", false), nil
		}, options)
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
	t.Run("pool_zero_disables_retries", func(t *testing.T) {
		c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
		account.Credentials = map[string]any{"pool_mode": true, "pool_mode_retry_count": 0}
		calls := 0
		_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) { calls++; return recoveryResponse(499, "", false), nil }, options)
		if err == nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
}

type recoveryBlockedBody struct {
	done   chan struct{}
	closed atomic.Bool
}

func (b *recoveryBlockedBody) Read([]byte) (int, error) { <-b.done; return 0, io.ErrClosedPipe }
func (b *recoveryBlockedBody) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.done)
	}
	return nil
}

func TestGatewayRecoveryBufferBoundsAndRetryAfter(t *testing.T) {
	t.Run("buffer_size_disarms_replay", func(t *testing.T) {
		body := ": " + strings.Repeat("x", 120) + "\n\n"
		resp := recoveryResponse(200, body, true)
		options := defaultGatewayRecoveryOptions()
		options.prefixBytes = 32
		failure, barrier := inspectGatewayRecoveryPrelude(resp, options)
		got, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if failure != nil || !barrier || string(got) != body {
			t.Fatalf("failure=%v barrier=%v", failure, barrier)
		}
	})
	t.Run("buffer_timeout_closes_reader", func(t *testing.T) {
		body := &recoveryBlockedBody{done: make(chan struct{})}
		resp := recoveryResponse(200, "", true)
		resp.Body = body
		options := defaultGatewayRecoveryOptions()
		options.prefixTimeout = 5 * time.Millisecond
		failure, _ := inspectGatewayRecoveryPrelude(resp, options)
		if failure == nil || !errors.Is(failure.err, errGatewayRecoveryPreludeTimeout) || !body.closed.Load() {
			t.Fatal("deadline not enforced")
		}
	})
	t.Run("retry_after_over_budget_no_early_retry", func(t *testing.T) {
		c, request, account, options := recoveryTestContext(t, `{"input":"x"}`, recoveryRule(model.ErrorActionRetry, 429))
		options.budget = time.Second
		calls := 0
		_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
			calls++
			resp := recoveryResponse(429, "", false)
			resp.Header.Set("Retry-After", "10")
			return resp, nil
		}, options)
		var final *UpstreamFailoverError
		if !errors.As(err, &final) || calls != 1 || final.ResponseHeaders.Get("Retry-After") != "10" {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
}
