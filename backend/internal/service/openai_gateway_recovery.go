package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// This boundary runs before endpoint-specific normalization, account penalties,
// failover and response writes. It reuses the SAME prepared request/account;
// session IDs, cache keys, continuation IDs, model and headers are never rebuilt.
// A decided terminal error stops the existing handler loop rather than nesting
// a second account-selection retry loop around an exhausted recovery budget.
func (s *OpenAIGatewayService) doOpenAIUpstreamWithRecovery(c *gin.Context, request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	// Several non-OpenAI adapters reuse this service. Keep their provider-specific
	// policy/usage semantics unchanged until they have an explicit recovery adapter.
	if account != nil && account.Platform != "" && account.Platform != PlatformOpenAI {
		return s.doOpenAIUpstream(request, proxyURL, account)
	}
	return recoverOpenAIUpstream(c, request, account, func(req *http.Request) (*http.Response, error) {
		return s.doOpenAIUpstream(req, proxyURL, account)
	}, defaultGatewayRecoveryOptions())
}

type gatewayRecoveryOptions struct {
	budget        time.Duration
	prefixTimeout time.Duration
	prefixBytes   int
	delays        []time.Duration
	jitter        bool
}

func defaultGatewayRecoveryOptions() gatewayRecoveryOptions {
	return gatewayRecoveryOptions{
		budget:        30 * time.Second,
		prefixTimeout: 30 * time.Second,
		prefixBytes:   64 << 10,
		delays:        []time.Duration{250 * time.Millisecond, 750 * time.Millisecond},
		jitter:        true,
	}
}

type gatewayRecoveryFailure struct {
	kind       string
	httpStatus int // actual HTTP status, zero only when no HTTP response exists
	status     int // rule-facing status; stream failures have a separate semantic status
	body       []byte
	headers    http.Header
	err        error
	unsafe     bool
	websocket  bool
}

var errGatewayRecoveryPreludeTimeout = errors.New("upstream first-output buffer deadline exceeded")

// The request is already in memory in the gateway. Decode only safety/model
// fields and never keep its contents in observations. Remote MCP tools can have
// side effects before a tool event becomes visible; background work is likewise
// not safely replayable without an upstream idempotency contract.
func gatewayRecoveryRequestInfo(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	if request.Body == nil || request.Body == http.NoBody {
		return "", request.Method == http.MethodGet
	}
	if request.GetBody == nil {
		return "", false
	}
	body, err := request.GetBody()
	if err != nil {
		return "", false
	}
	defer func() { _ = body.Close() }()
	var info struct {
		Model      string `json:"model"`
		Background bool   `json:"background"`
		Tools      []struct {
			Type string `json:"type"`
		} `json:"tools"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 32<<20)).Decode(&info); err != nil {
		return "", false
	}
	if info.Background {
		return sanitizeSessionID(info.Model), false
	}
	for _, tool := range info.Tools {
		if tool.Type == "mcp" {
			return sanitizeSessionID(info.Model), false
		}
	}
	return sanitizeSessionID(info.Model), true
}

func gatewayRecoveryTransient(f gatewayRecoveryFailure) bool {
	if f.unsafe {
		return false
	}
	if f.err != nil {
		if errors.Is(f.err, context.Canceled) || errors.Is(f.err, errGatewayRecoveryPreludeTimeout) {
			return false
		}
		var pluginErr *PluginTransportError
		if errors.As(f.err, &pluginErr) && pluginErr.RequestSent {
			return false
		}
		if classifyUpstreamTransportError(f.err).Persistent {
			return false
		}
		if errors.Is(f.err, io.EOF) || errors.Is(f.err, io.ErrUnexpectedEOF) || errors.Is(f.err, syscall.ECONNRESET) || errors.Is(f.err, syscall.EPIPE) {
			return true
		}
		var streamErr http2.StreamError
		if errors.As(f.err, &streamErr) {
			return true
		}
		var networkErr net.Error
		return errors.As(f.err, &networkErr) && networkErr.Timeout()
	}
	if hit, _, _ := detectOpenAICyberPolicy(f.body); hit {
		return false
	}
	if isOpenAIContextWindowError("", f.body) || isOpenAICompatibleModelNotFoundBody(f.body) || isOpenAIUpstreamAccessStateError("", f.body) || openAIStreamCredentialAuthFailure(f.body) {
		return false
	}
	for _, path := range []string{"error.type", "response.error.type", "error.code", "response.error.code"} {
		value := strings.ToLower(gjson.GetBytes(f.body, path).String())
		for _, deny := range []string{"invalid_request", "authentication", "permission", "content_policy", "safety", "context_length", "model_not_found"} {
			if strings.Contains(value, deny) {
				return false
			}
		}
	}
	// Rate limits keep the existing capacity policy unless a retry rule opts in.
	if f.status == http.StatusTooManyRequests {
		return false
	}
	if f.kind == "http" {
		switch f.httpStatus {
		case 499, 502, 503, 504, 524:
			return true
		}
		return isOpenAITransientProcessingError(f.httpStatus, "", f.body)
	}
	// A generic response.failed is NOT evidence of a transient fault. Use only
	// explicit structured error codes/types or existing narrow overload signals.
	for _, path := range []string{"error.code", "response.error.code", "error.type", "response.error.type"} {
		switch strings.ToLower(gjson.GetBytes(f.body, path).String()) {
		case "server_error", "internal_server_error", "temporarily_unavailable", "service_unavailable", "server_is_overloaded", "overloaded_error", "slow_down", "upstream_timeout":
			return true
		}
	}
	return isOpenAITransientProcessingError(http.StatusBadRequest, "", f.body)
}

func gatewayRecoveryRule(c *gin.Context, account *Account, f gatewayRecoveryFailure) *model.ErrorPassthroughRule {
	svc := getBoundErrorPassthroughService(c)
	if svc == nil || account == nil {
		return nil
	}
	return svc.MatchRule(account.Platform, f.status, f.body)
}

func gatewayRecoveryReplayAllowed(c *gin.Context, request *http.Request, observation *gatewayRequestObservation) bool {
	if request.Context().Err() != nil {
		return false
	}
	if c != nil && c.Request != nil && c.Request.Context().Err() != nil {
		return false
	}
	observation.mu.Lock()
	started := observation.semanticOutput
	observation.mu.Unlock()
	if started {
		return false
	}
	// A WS handshake is not generated content. The WS relay enforces its own
	// per-turn semantic boundary; HTTP compact keepalive comments are also safe.
	if c != nil && c.Writer != nil && c.Writer.Status() != http.StatusSwitchingProtocols && GetOpenAIClientTransport(c) != OpenAIClientTransportWS && openAIStreamClientOutputStarted(c, false) {
		return false
	}
	return true
}

func gatewayRecoveryRetryAfter(headers http.Header, body []byte) time.Duration {
	for _, raw := range []string{headers.Get("Retry-After"), gjson.GetBytes(body, "error.headers.retry-after").String(), gjson.GetBytes(body, "response.error.headers.retry-after").String()} {
		if seconds, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && seconds > 0 && seconds < 86400 {
			return time.Duration(seconds * float64(time.Second))
		}
		if date, err := http.ParseTime(raw); err == nil && time.Until(date) > 0 {
			return time.Until(date)
		}
	}
	for _, raw := range []string{headers.Get("Retry-After-Ms"), gjson.GetBytes(body, "error.headers.retry-after-ms").String()} {
		if millis, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && millis > 0 && millis < 86400000 {
			return time.Duration(millis * float64(time.Millisecond))
		}
	}
	return 0
}

func reserveGatewayRecoveryRetry(c *gin.Context, request *http.Request, account *Account, observation *gatewayRequestObservation, f gatewayRecoveryFailure, options gatewayRecoveryOptions) (time.Duration, bool) {
	observation.mu.Lock()
	defer observation.mu.Unlock()
	if !observation.limitInitialized {
		observation.retryLimit = 2
		if account != nil && account.IsPoolMode() {
			if value, exists := account.Credentials["pool_mode_retry_count"]; exists && value != nil {
				observation.retryLimit = account.GetPoolModeRetryCount()
			}
		}
		observation.deadline = time.Now().Add(options.budget)
		for _, ctx := range []context.Context{request.Context(), c.Request.Context()} {
			if deadline, ok := ctx.Deadline(); ok && deadline.Before(observation.deadline) {
				observation.deadline = deadline
			}
		}
		observation.limitInitialized = true
	}
	if observation.retries >= observation.retryLimit || observation.meta.AttemptCount-1 >= observation.retryLimit || len(options.delays) == 0 {
		return 0, false
	}
	index := observation.retries
	if index >= len(options.delays) {
		index = len(options.delays) - 1
	}
	delay := options.delays[index]
	if options.jitter {
		delay += time.Duration(rand.Float64() * 0.15 * float64(delay))
	}
	if retryAfter := gatewayRecoveryRetryAfter(f.headers, f.body); retryAfter > delay {
		delay = retryAfter
	}
	if !time.Now().Add(delay).Before(observation.deadline) {
		return 0, false
	}
	observation.retries++
	return delay, true
}

func gatewayRecoveryMessage(f gatewayRecoveryFailure) string {
	message := extractUpstreamErrorMessage(f.body)
	if nested := gjson.GetBytes(f.body, "response.error.message"); nested.Type == gjson.String {
		message = nested.String()
	}
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" || strings.HasPrefix(message, "<") {
		message = "Upstream request failed"
	}
	return truncateString(message, 1024)
}

func gatewayRecoveryTerminal(c *gin.Context, f gatewayRecoveryFailure, rule *model.ErrorPassthroughRule, action, outcome string) *UpstreamFailoverError {
	status := f.status
	if status < 400 || status > 599 {
		status = http.StatusBadGateway
	}
	message := gatewayRecoveryMessage(f)
	headers := f.headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	ruleID := int64(0)
	if rule != nil {
		ruleID = rule.ID
		if rule.SkipMonitoring && c != nil {
			c.Set(OpsSkipPassthroughKey, true)
		}
		if action == model.ErrorActionRewrite {
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				status = *rule.ResponseCode
			}
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				message = sanitizeUpstreamErrorMessage(*rule.CustomMessage)
			}
		}
	}
	if action == model.ErrorActionRetry && outcome != "not_retryable" {
		status = http.StatusServiceUnavailable
		message = "Upstream service temporarily unavailable; recovery attempts exhausted"
		if !validOpenAIPassthroughRetryAfter(headers.Get("Retry-After"), time.Now()) {
			headers.Set("Retry-After", "1")
		}
	}
	setOpsUpstreamError(c, f.httpStatus, gatewayRecoveryMessage(f), "")
	SetGatewayRecoveryOutcome(c, outcome)
	return &UpstreamFailoverError{
		StatusCode: f.status, ResponseBody: append([]byte(nil), f.body...), ResponseHeaders: headers,
		RecoveryHandled: true, RecoveryAction: action, RecoveryRuleID: ruleID,
		RecoveryFailureKind: f.kind, RecoveryUpstreamStatus: f.httpStatus,
		NextAccountAction: NextAccountStop, ClientStatusCode: status, ClientMessage: message,
		RequestScopedTransient: true,
	}
}

func recordGatewayRecoveryAttempt(c *gin.Context, observation *gatewayRequestObservation, account *Account, item GatewayAttemptMetadata, f *gatewayRecoveryFailure, rule *model.ErrorPassthroughRule) {
	item.EndedAt = time.Now().UTC()
	if account != nil {
		item.AccountID = account.ID
	}
	if rule != nil {
		item.RuleID = rule.ID
	}
	if f != nil {
		item.UpstreamStatus, item.FailureKind = f.httpStatus, f.kind
		item.CostAccounting = "unknown"
		for _, prefix := range []string{"usage", "response.usage"} {
			if gjson.GetBytes(f.body, prefix).Exists() {
				item.InputTokens = gjson.GetBytes(f.body, prefix+".input_tokens").Int()
				item.OutputTokens = gjson.GetBytes(f.body, prefix+".output_tokens").Int()
				item.CostAccounting = "upstream_reported_not_billed"
			}
		}
	}
	observation.mu.Lock()
	if len(observation.meta.Attempts) < 32 {
		observation.meta.Attempts = append(observation.meta.Attempts, item)
	}
	meta := observation.meta
	observation.mu.Unlock()
	logger.FromContext(c.Request.Context()).Info("gateway.upstream_attempt",
		zap.String("request_id", meta.DownstreamRequestID), zap.String("conversation_id", meta.ConversationID),
		zap.String("conversation_id_source", meta.ConversationIDSource), zap.Int("attempt", item.Attempt),
		zap.Int64("account_id", item.AccountID), zap.String("model", item.Model),
		zap.Int("upstream_status", item.UpstreamStatus), zap.String("failure_kind", item.FailureKind),
		zap.Int64("rule_id", item.RuleID), zap.String("action", item.Action), zap.String("outcome", item.Outcome),
		zap.Time("started_at", item.StartedAt), zap.Time("ended_at", item.EndedAt), zap.String("cost_accounting", item.CostAccounting))
	// Legacy handlers own their Ops event (and proxy attribution). The shared
	// metadata/log still captures this attempt, but must not duplicate that event.
	if f != nil && account != nil && item.Outcome != "legacy_handling" {
		proxyID, proxyName := opsUpstreamProxyAttribution(account)
		if f.websocket {
			proxyID, proxyName = opsUpstreamWSProxyAttribution(account)
		}
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name,
			ProxyID: proxyID, ProxyName: proxyName,
			Passthrough: c.GetBool("openai_passthrough"),
			Attempt:     item.Attempt, RuleID: item.RuleID, Action: item.Action, RecoveryOutcome: item.Outcome,
			SkipMonitoring:     rule != nil && rule.SkipMonitoring && item.Outcome != "retrying",
			UpstreamStatusCode: f.httpStatus, Kind: "recovery_" + f.kind, Message: gatewayRecoveryMessage(*f)})
	}
}

type gatewayRecoveryBody struct {
	io.Reader
	closer io.Closer
	finish func()
	once   sync.Once
}

func (b *gatewayRecoveryBody) Close() error {
	var err error
	b.once.Do(func() {
		err = b.closer.Close()
		if b.finish != nil {
			b.finish()
		}
	})
	return err
}

func recoverOpenAIUpstream(c *gin.Context, request *http.Request, account *Account, trip func(*http.Request) (*http.Response, error), options gatewayRecoveryOptions) (*http.Response, error) {
	if c == nil || c.Request == nil || request == nil {
		return trip(request)
	}
	observation := ensureGatewayObservation(c.Request)
	requestModel, replayable := gatewayRecoveryRequestInfo(request)
	for retry := 0; ; retry++ {
		if err := c.Request.Context().Err(); err != nil {
			SetGatewayRecoveryOutcome(c, "cancelled")
			return nil, err
		}
		if err := request.Context().Err(); err != nil {
			return nil, err
		}
		attemptCtx, cancel := context.WithCancel(request.Context())
		observation.mu.Lock()
		recoveryDeadline := observation.deadline
		observation.mu.Unlock()
		var budgetTimer *time.Timer
		if !recoveryDeadline.IsZero() {
			budgetTimer = time.AfterFunc(time.Until(recoveryDeadline), cancel)
		}
		stopDownstream := context.AfterFunc(c.Request.Context(), cancel)
		cleanup := func() {
			if budgetTimer != nil {
				budgetTimer.Stop()
			}
			stopDownstream()
			cancel()
		}
		attemptRequest := request.Clone(attemptCtx)
		if retry > 0 && request.Body != nil && request.Body != http.NoBody {
			body, err := request.GetBody()
			if err != nil {
				cleanup()
				return nil, err
			}
			attemptRequest.Body = body
		}
		attempt, started := beginGatewayAttempt(c)
		item := GatewayAttemptMetadata{Attempt: attempt, Model: requestModel, StartedAt: started}
		resp, err := trip(attemptRequest)
		failure := gatewayRecoveryFailure{kind: "transport", status: http.StatusBadGateway, err: err}
		if err == nil && resp == nil {
			err = io.ErrUnexpectedEOF
			failure.err = err
		}
		if err == nil {
			failure.httpStatus, failure.headers = resp.StatusCode, resp.Header
			if resp.StatusCode >= 400 {
				prefix, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
				resp.Body = &gatewayRecoveryBody{Reader: io.MultiReader(bytes.NewReader(prefix), resp.Body), closer: resp.Body}
				failure.kind, failure.status, failure.body = "http", resp.StatusCode, prefix
			} else if isEventStreamResponse(resp.Header) {
				streamFailure, barrier := inspectGatewayRecoveryPrelude(resp, options)
				if streamFailure != nil {
					failure = *streamFailure
				} else {
					observation.mu.Lock()
					observation.semanticOutput = observation.semanticOutput || barrier
					observation.mu.Unlock()
					failure.kind = ""
				}
			} else {
				failure.kind = ""
			}
		}
		if downstreamErr := c.Request.Context().Err(); downstreamErr != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			cleanup()
			item.Outcome = "cancelled"
			recordGatewayRecoveryAttempt(c, observation, account, item, &failure, nil)
			SetGatewayRecoveryOutcome(c, "cancelled")
			return nil, downstreamErr
		}
		if budgetTimer != nil {
			if !budgetTimer.Stop() {
				if resp != nil && resp.Body != nil {
					_ = resp.Body.Close()
				}
				cleanup()
				failure.kind, failure.err = "recovery_deadline", context.DeadlineExceeded
				item.Action, item.Outcome = model.ErrorActionRetry, "exhausted"
				recordGatewayRecoveryAttempt(c, observation, account, item, &failure, nil)
				return nil, gatewayRecoveryTerminal(c, failure, nil, model.ErrorActionRetry, "exhausted")
			}
			// A long accepted generation is not a retry attempt budget timeout.
			budgetTimer = nil
		}
		if failure.kind == "" {
			// The accepted generation keeps its bounded transport drain for final
			// billing usage. Cancellation still prevents all new recovery attempts.
			stopDownstream()
			item.UpstreamStatus, item.Outcome, item.CostAccounting = resp.StatusCode, "forwarded", "usage_log"
			resp.Body = &gatewayRecoveryBody{Reader: resp.Body, closer: resp.Body, finish: func() {
				cleanup()
				recordGatewayRecoveryAttempt(c, observation, account, item, nil, nil)
				CompleteGatewayRequestObservation(c)
			}}
			if retry > 0 {
				SetGatewayRecoveryOutcome(c, "forwarded_after_retry")
			}
			return resp, nil
		}
		// Security-policy faults keep the dedicated policy response/telemetry path.
		// For unowned 429/auth/context faults, retain the existing capacity and
		// credential policies rather than swallowing their original stream event.
		policyFault, _, _ := detectOpenAICyberPolicy(failure.body)
		rule := gatewayRecoveryRule(c, account, failure)
		action := ""
		if rule != nil {
			action = rule.EffectiveAction()
		}
		transient := gatewayRecoveryTransient(failure)
		// 429 keeps the existing capacity strategy unless explicitly covered by a
		// retry rule. Its Retry-After is a minimum, never truncated to a fast retry.
		if rule != nil && action == model.ErrorActionRetry && failure.status == http.StatusTooManyRequests && !failure.unsafe {
			transient = true
		}
		if rule == nil && transient {
			action = model.ErrorActionRetry
		}
		if action == "" && failure.err != nil && retry > 0 {
			// Once recovery owns this request, an unclassified later transport
			// fault must not escape into a fresh outer failover/retry budget.
			item.Action, item.Outcome = model.ErrorActionRetry, "stopped_after_retry"
			recordGatewayRecoveryAttempt(c, observation, account, item, &failure, nil)
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			cleanup()
			return nil, gatewayRecoveryTerminal(c, failure, nil, model.ErrorActionRetry, "stopped_after_retry")
		}
		legacyStreamFault := failure.kind == "stream_event" && (failure.status == http.StatusTooManyRequests || openAIStreamCredentialAuthFailure(failure.body) || isOpenAIUpstreamAccessStateError("", failure.body) || isOpenAIContextWindowError("", failure.body) || isOpenAICompatibleModelNotFoundBody(failure.body))
		if policyFault || (action == "" && ((failure.kind != "stream_event" && failure.kind != "stream_read") || legacyStreamFault)) {
			item.Outcome = "legacy_handling"
			recordGatewayRecoveryAttempt(c, observation, account, item, &failure, nil)
			if resp != nil && resp.Body != nil {
				resp.Body = &gatewayRecoveryBody{Reader: resp.Body, closer: resp.Body, finish: cleanup}
			} else {
				cleanup()
			}
			return resp, err
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		cleanup()
		outcome := "rule_terminal"
		var delay time.Duration
		allowed := false
		if action == model.ErrorActionRetry {
			outcome = "not_retryable"
			if transient {
				outcome = "not_replayable"
				if replayable && gatewayRecoveryReplayAllowed(c, request, observation) {
					outcome = "exhausted"
					delay, allowed = reserveGatewayRecoveryRetry(c, request, account, observation, failure, options)
				}
			}
		}
		item.Action, item.Outcome = action, outcome
		if allowed {
			item.Outcome = "retrying"
		}
		recordGatewayRecoveryAttempt(c, observation, account, item, &failure, rule)
		if !allowed {
			return nil, gatewayRecoveryTerminal(c, failure, rule, action, outcome)
		}
		SetGatewayRecoveryOutcome(c, "retrying")
		timer := time.NewTimer(delay)
		select {
		case <-c.Request.Context().Done():
			timer.Stop()
			SetGatewayRecoveryOutcome(c, "cancelled")
			return nil, c.Request.Context().Err()
		case <-request.Context().Done():
			timer.Stop()
			return nil, request.Context().Err()
		case <-timer.C:
		}
	}
}

// inspectGatewayRecoveryPrelude buffers only a bounded protocol prefix. It is
// replayed byte-for-byte exactly once on acceptance. Failure metadata is never
// forwarded into a subsequent successful attempt. Unknown events conservatively
// cross the replay barrier rather than risking duplicate content or side effects.
func inspectGatewayRecoveryPrelude(resp *http.Response, options gatewayRecoveryOptions) (failure *gatewayRecoveryFailure, barrier bool) {
	body := resp.Body
	reader := bufio.NewReaderSize(body, 4096)
	var prefix bytes.Buffer
	var data strings.Builder
	eventType := ""
	var timedOut atomic.Bool
	timeoutDone := make(chan struct{})
	timer := time.AfterFunc(options.prefixTimeout, func() { timedOut.Store(true); _ = body.Close(); close(timeoutDone) })
	defer func() {
		if !timer.Stop() {
			<-timeoutDone
			failure = &gatewayRecoveryFailure{kind: "stream_read", httpStatus: resp.StatusCode, status: http.StatusGatewayTimeout, headers: resp.Header, err: errGatewayRecoveryPreludeTimeout}
			barrier = false
		}
	}()
	restore := func() {
		// Bytes already read from upstream belong to the same available batch.
		// Preserve that boundary so the stream writer can coalesce flushes without
		// waiting for another upstream read.
		if buffered := reader.Buffered(); buffered > 0 {
			available, _ := reader.Peek(buffered)
			_, _ = prefix.Write(available)
			_, _ = reader.Discard(buffered)
		}
		resp.Body = &gatewayRecoveryBody{Reader: io.MultiReader(bytes.NewReader(prefix.Bytes()), reader), closer: body}
	}
	var partialLine strings.Builder
	for {
		line, readErr := reader.ReadSlice('\n')
		_, _ = prefix.Write(line)
		_, _ = partialLine.Write(line)
		if timedOut.Load() {
			return &gatewayRecoveryFailure{kind: "stream_read", httpStatus: resp.StatusCode, status: http.StatusGatewayTimeout, headers: resp.Header, err: errGatewayRecoveryPreludeTimeout}, false
		}
		if prefix.Len() >= options.prefixBytes {
			restore()
			return nil, true
		}
		if readErr == bufio.ErrBufferFull {
			continue
		}
		text := strings.TrimRight(partialLine.String(), "\r\n")
		partialLine.Reset()
		if strings.HasPrefix(text, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(text, "event:"))
		}
		if strings.HasPrefix(text, "data:") {
			if data.Len() > 0 {
				_ = data.WriteByte('\n')
			}
			_, _ = data.WriteString(strings.TrimPrefix(strings.TrimPrefix(text, "data:"), " "))
		}
		if text == "" || readErr != nil {
			payload := data.String()
			if eventType == "" {
				eventType = gjson.Get(payload, "type").String()
			}
			if payload == "[DONE]" {
				restore()
				return nil, true
			}
			if eventType == "response.failed" || eventType == "error" || (eventType == "" && gjson.Get(payload, "error").IsObject()) {
				failedBody := []byte(payload)
				message := extractOpenAISSEErrorMessage(failedBody)
				status := openAIStreamFailedEventSemanticStatus(failedBody, message)
				for _, path := range openAIStreamErrorStatusPaths {
					if code := int(gjson.GetBytes(failedBody, path).Int()); code >= 400 && code <= 599 {
						status = code
						break
					}
				}
				unsafe := false
				for _, output := range gjson.Get(payload, "response.output").Array() {
					unsafe = unsafe || openAIStreamItemHasVisibleOutput(output) || output.Get("type").String() == "function_call"
				}
				restore()
				return &gatewayRecoveryFailure{kind: "stream_event", httpStatus: resp.StatusCode, status: status, headers: resp.Header, body: failedBody, unsafe: unsafe}, false
			}
			if payload != "" && gatewayRecoveryStreamStartsOutput(payload, eventType) {
				restore()
				return nil, true
			}
			data.Reset()
			eventType = ""
		}
		if readErr != nil {
			restore()
			return &gatewayRecoveryFailure{kind: "stream_read", httpStatus: resp.StatusCode, status: http.StatusBadGateway, headers: resp.Header, err: readErr}, false
		}
	}
}

func gatewayRecoveryStreamStartsOutput(data, eventType string) bool {
	if eventType == "response.output_item.added" {
		switch gjson.Get(data, "item.type").String() {
		case "function_call", "custom_tool_call", "mcp_call":
			return true
		}
	}
	if eventType == "" && gjson.Get(data, "choices").IsArray() {
		for _, choice := range gjson.Get(data, "choices").Array() {
			for _, field := range []string{"content", "reasoning", "reasoning_content", "tool_calls", "function_call", "audio", "refusal"} {
				value := choice.Get("delta." + field)
				if value.Exists() && value.Raw != "null" && value.String() != "" && value.Raw != "[]" {
					return true
				}
			}
			if value := choice.Get("finish_reason"); value.Exists() && value.Raw != "null" {
				return true
			}
		}
		return false
	}
	return openAIStreamDataStartsClientOutput(data, eventType)
}
