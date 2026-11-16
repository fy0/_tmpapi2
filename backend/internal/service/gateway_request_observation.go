package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// GatewayRequestMetadata is observation-only. Never use it as a scheduling,
// cache, idempotency or authorization key. A conversation is scoped by the
// enclosing log row's user_id AND api_key_id, not by this client-supplied string.
// Zero values deliberately describe unknown historical data.
type GatewayRequestMetadata struct {
	DownstreamRequestID  string                   `json:"downstream_request_id,omitempty"`
	Attempts             []GatewayAttemptMetadata `json:"attempts,omitempty"`
	DownstreamStatusCode int                      `json:"downstream_status_code,omitempty"`
	ConversationID       string                   `json:"conversation_id,omitempty"`
	ConversationIDSource string                   `json:"conversation_id_source,omitempty"`
	RequestBodyBytes     int                      `json:"request_body_bytes,omitempty"`
	RequestStartedAt     *time.Time               `json:"request_started_at,omitempty"`
	RequestEndedAt       *time.Time               `json:"request_ended_at,omitempty"`
	AttemptCount         int                      `json:"attempt_count,omitempty"`
	RecoveryOutcome      string                   `json:"recovery_outcome,omitempty"`
}

// GatewayAttemptMetadata deliberately excludes request/response content and secrets.
// Unknown failed-attempt usage is not zero cost: its accounting state stays unknown.
type GatewayAttemptMetadata struct {
	Attempt        int       `json:"attempt"`
	AccountID      int64     `json:"account_id"`
	Model          string    `json:"model,omitempty"`
	UpstreamStatus int       `json:"upstream_status"`
	FailureKind    string    `json:"failure_kind,omitempty"`
	RuleID         int64     `json:"rule_id,omitempty"`
	Action         string    `json:"action,omitempty"`
	Outcome        string    `json:"outcome"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at"`
	CostAccounting string    `json:"cost_accounting,omitempty"`
	InputTokens    int64     `json:"input_tokens,omitempty"`
	OutputTokens   int64     `json:"output_tokens,omitempty"`
}

type gatewayObservationContextKey struct{}

type gatewayRequestObservation struct {
	mu   sync.Mutex
	meta GatewayRequestMetadata
	// HTTP, SSE and WebSocket recovery share these downstream-request counters.
	retries          int
	retryLimit       int
	limitInitialized bool
	deadline         time.Time
	semanticOutput   bool
	turn             int
}

func gatewayObservation(ctx context.Context) *gatewayRequestObservation {
	if ctx == nil {
		return nil
	}
	observation, _ := ctx.Value(gatewayObservationContextKey{}).(*gatewayRequestObservation)
	return observation
}

// BindGatewayRequestObservation runs before c.Next(), including requests which
// fail authentication, never select an account, or are cancelled while queued.
func BindGatewayRequestObservation(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	observation := ensureGatewayObservation(c.Request)
	observation.mu.Lock()
	if observation.meta.ConversationID == "" {
		if id := ExtractClientSessionID(c); id != "" {
			observation.meta.ConversationID = id
			observation.meta.ConversationIDSource = "header.client_session"
		}
	}
	observation.mu.Unlock()
}

func ensureGatewayObservation(req *http.Request) *gatewayRequestObservation {
	if req == nil {
		return nil
	}
	if existing := gatewayObservation(req.Context()); existing != nil {
		return existing
	}
	now := time.Now().UTC()
	observation := &gatewayRequestObservation{meta: GatewayRequestMetadata{RequestStartedAt: &now}}
	observation.meta.DownstreamRequestID, _ = req.Context().Value(ctxkey.RequestID).(string)
	id, source := resolveGatewayConversation(req.Header, nil)
	observation.meta.ConversationID, observation.meta.ConversationIDSource = id, source
	*req = *req.WithContext(context.WithValue(req.Context(), gatewayObservationContextKey{}, observation))
	return observation
}

// CaptureGatewayRequestBody observes original inbound bytes before model/account
// selection or upstream normalization. It retains no body, only byte length and
// an allowlisted identifier. It never changes headers or scheduling inputs.
func CaptureGatewayRequestBody(req *http.Request, body []byte) {
	observation := ensureGatewayObservation(req)
	if observation == nil {
		return
	}
	id, source := resolveGatewayConversation(req.Header, body)
	observation.mu.Lock()
	if id != "" {
		observation.meta.ConversationID, observation.meta.ConversationIDSource = id, source
	}
	observation.meta.RequestBodyBytes = len(body)
	meta := observation.meta
	observation.mu.Unlock()
	requestID, _ := req.Context().Value(ctxkey.RequestID).(string)
	log := logger.FromContext(req.Context()).With(
		zap.String("conversation_id", meta.ConversationID),
		zap.String("conversation_id_source", meta.ConversationIDSource),
	)
	log.Debug("gateway.request_correlated", zap.String("request_id", requestID), zap.Int("request_body_bytes", len(body)))
	*req = *req.WithContext(logger.IntoContext(req.Context(), log))
}

func resolveGatewayConversation(headers http.Header, body []byte) (string, string) {
	for _, name := range []string{"conversation_id", "conversation-id", "X-Conversation-ID"} {
		if id := sanitizeSessionID(headers.Get(name)); id != "" {
			return id, "header." + strings.ToLower(name)
		}
	}
	validJSON := gjson.ValidBytes(body)
	if validJSON {
		for _, name := range []string{"conversation_id", "conversation.id"} {
			value := gjson.GetBytes(body, name)
			if value.Type == gjson.String {
				if id := sanitizeSessionID(value.String()); id != "" {
					return id, "body." + name
				}
			}
		}
	}
	for _, name := range clientSessionIDHeaders {
		if id := sanitizeSessionID(headers.Get(name)); id != "" {
			return id, "header." + strings.ToLower(name)
		}
	}
	if !validJSON {
		return "", ""
	}
	for _, name := range []string{"session_id", "metadata.session_id"} {
		value := gjson.GetBytes(body, name)
		if value.Type == gjson.String {
			if id := sanitizeSessionID(value.String()); id != "" {
				return id, "body." + name
			}
		}
	}
	if value := gjson.GetBytes(body, "prompt_cache_key"); value.Type == gjson.String {
		if id := sanitizeSessionID(value.String()); id != "" {
			return id, "fallback.prompt_cache_key"
		}
	}
	return "", ""
}

func GatewayRequestMetadataFromContext(ctx context.Context) GatewayRequestMetadata {
	observation := gatewayObservation(ctx)
	if observation == nil {
		return GatewayRequestMetadata{}
	}
	observation.mu.Lock()
	defer observation.mu.Unlock()
	meta := observation.meta
	meta.Attempts = append([]GatewayAttemptMetadata(nil), meta.Attempts...)
	return meta
}

// CopyGatewayRequestObservation freezes metadata before an asynchronous usage
// task is enqueued. A later WebSocket turn cannot change the completed turn's ID.
func CopyGatewayRequestObservation(parent, destination context.Context) context.Context {
	if destination == nil {
		destination = context.Background()
	}
	if gatewayObservation(parent) == nil {
		return destination
	}
	meta := GatewayRequestMetadataFromContext(parent)
	if meta.RequestEndedAt == nil {
		now := time.Now().UTC()
		meta.RequestEndedAt = &now
	}
	return context.WithValue(destination, gatewayObservationContextKey{}, &gatewayRequestObservation{meta: meta})
}

// BeginGatewayRequestTurn resets a WS turn in-place, retaining the context pointer
// held by admission/usage code. Frozen prior-turn snapshots are independent.
func BeginGatewayRequestTurn(c *gin.Context, turn int, body []byte) {
	if c == nil || c.Request == nil || turn <= 0 {
		return
	}
	observation := ensureGatewayObservation(c.Request)
	observation.mu.Lock()
	if observation.turn == turn {
		observation.mu.Unlock()
		return
	}
	now := time.Now().UTC()
	requestID := observation.meta.DownstreamRequestID
	observation.meta = GatewayRequestMetadata{DownstreamRequestID: requestID, RequestStartedAt: &now}
	observation.retries, observation.retryLimit = 0, 0
	observation.limitInitialized, observation.semanticOutput = false, false
	observation.deadline, observation.turn = time.Time{}, turn
	observation.mu.Unlock()
	CaptureGatewayRequestBody(c.Request, body)
}

// CompleteGatewayRequestObservation freezes the outcome visible at this boundary.
// A committed 200/101 stream can still have a protocol-level failure.
func CompleteGatewayRequestObservation(c *gin.Context) GatewayRequestMetadata {
	if c == nil || c.Request == nil {
		return GatewayRequestMetadata{}
	}
	meta := GatewayRequestMetadataFromContext(c.Request.Context())
	outcome := meta.RecoveryOutcome
	if c.Request.Context().Err() != nil {
		outcome = "cancelled"
	} else if _, failed := GetOpsStreamError(c); failed {
		outcome = "failed_after_output"
	} else if c.Writer != nil && c.Writer.Status() < 400 && (outcome == "forwarded_after_retry" || outcome == "retrying") {
		outcome = "recovered"
	} else if outcome == "" && meta.AttemptCount > 0 {
		outcome = "forwarded"
	}
	return FinishGatewayRequestObservation(c, outcome)
}

func beginGatewayAttempt(c *gin.Context) (int, time.Time) {
	now := time.Now().UTC()
	if c == nil || c.Request == nil {
		return 1, now
	}
	observation := ensureGatewayObservation(c.Request)
	observation.mu.Lock()
	defer observation.mu.Unlock()
	observation.meta.AttemptCount++
	return observation.meta.AttemptCount, now
}

func SetGatewayRecoveryOutcome(c *gin.Context, outcome string) {
	if c == nil || c.Request == nil {
		return
	}
	observation := ensureGatewayObservation(c.Request)
	observation.mu.Lock()
	observation.meta.RecoveryOutcome = outcome
	observation.mu.Unlock()
}

func FinishGatewayRequestObservation(c *gin.Context, outcome string) GatewayRequestMetadata {
	if c == nil || c.Request == nil {
		return GatewayRequestMetadata{}
	}
	observation := ensureGatewayObservation(c.Request)
	now := time.Now().UTC()
	observation.mu.Lock()
	defer observation.mu.Unlock()
	observation.meta.RequestEndedAt = &now
	if c.Writer != nil {
		observation.meta.DownstreamStatusCode = c.Writer.Status()
	}
	if outcome != "" {
		observation.meta.RecoveryOutcome = outcome
	}
	meta := observation.meta
	meta.Attempts = append([]GatewayAttemptMetadata(nil), meta.Attempts...)
	return meta
}

// GatewayRequestMetadataJSON only serializes the fixed observation schema.
// Credentials, arbitrary client metadata, and conversation content cannot enter.
func GatewayRequestMetadataJSON(meta GatewayRequestMetadata) *string {
	if meta.RequestStartedAt == nil && meta.ConversationID == "" && meta.AttemptCount == 0 {
		return nil
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil
	}
	value := string(data)
	return &value
}

func DecodeGatewayRequestMetadata(data, conversationID, source string) GatewayRequestMetadata {
	var meta GatewayRequestMetadata
	if data != "" {
		_ = json.Unmarshal([]byte(data), &meta)
	}
	// Indexed columns are authoritative, including NULL/empty historical IDs.
	meta.ConversationID = conversationID
	meta.ConversationIDSource = source
	return meta
}
