package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// A reserved retry is consumed only by the existing same-account WS loop.
// The loop must not run payload-repair or account-selection retries around it.
type gatewayWSRecoveryRetry struct{ delay time.Duration }

func (e *gatewayWSRecoveryRetry) Error() string { return "upstream websocket recovery retry reserved" }

type gatewayWSRecoveryAttempt struct {
	c              *gin.Context
	account        *Account
	observation    *gatewayRequestObservation
	request        *http.Request
	item           GatewayAttemptMetadata
	replayable     bool
	strictAffinity bool
	semantic       bool
	recorded       bool
	accepted       bool
	status         int
	options        gatewayRecoveryOptions
}

func beginGatewayWSRecoveryAttempt(c *gin.Context, account *Account, payload []byte, strictAffinity bool) *gatewayWSRecoveryAttempt {
	if c == nil || c.Request == nil {
		return nil
	}
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, "https://upstream.invalid/v1/responses", bytes.NewReader(payload))
	if err != nil {
		return nil
	}
	modelName, replayable := gatewayRecoveryRequestInfo(request)
	attempt, started := beginGatewayAttempt(c)
	return &gatewayWSRecoveryAttempt{c: c, account: account, observation: ensureGatewayObservation(c.Request), request: request,
		item: GatewayAttemptMetadata{Attempt: attempt, Model: modelName, StartedAt: started}, replayable: replayable,
		strictAffinity: strictAffinity, options: defaultGatewayRecoveryOptions()}
}

func gatewayWSPayloadBytes(payload map[string]any) []byte {
	data, _ := json.Marshal(payload)
	return data
}

func (a *gatewayWSRecoveryAttempt) observeOutput(payload []byte, eventType string) {
	if a == nil || eventType == "error" || eventType == "response.failed" {
		return
	}
	a.status = http.StatusSwitchingProtocols
	if gatewayRecoveryStreamStartsOutput(string(payload), eventType) {
		a.semantic = true
		a.observation.mu.Lock()
		a.observation.semanticOutput = true
		a.observation.mu.Unlock()
	}
	if eventType == "response.completed" || eventType == "response.done" {
		a.accepted = true
	}
}

func (a *gatewayWSRecoveryAttempt) stopReplay() {
	if a == nil {
		return
	}
	a.semantic = true
	a.observation.mu.Lock()
	a.observation.semanticOutput = true
	a.observation.mu.Unlock()
}

// Returns nil only when the original fault remains owned by the legacy handler.
// Original protocol/config errors keep their existing explicit repair semantics.
func (a *gatewayWSRecoveryAttempt) recover(err error, headers http.Header, event []byte, outputStarted bool) error {
	if a == nil {
		return nil
	}
	if cancelled := a.c.Request.Context().Err(); cancelled != nil {
		SetGatewayRecoveryOutcome(a.c, "cancelled")
		return cancelled
	}
	failure := gatewayRecoveryFailure{kind: "transport", status: http.StatusBadGateway, err: err, headers: headers}
	if len(event) > 0 {
		failure.kind, failure.httpStatus, failure.err = "stream_event", http.StatusSwitchingProtocols, nil
		failure.body = event
		failure.status = openAIStreamFailedEventSemanticStatus(event, extractOpenAISSEErrorMessage(event))
		for _, path := range openAIStreamErrorStatusPaths {
			if code := int(gjson.GetBytes(event, path).Int()); code >= 400 && code <= 599 {
				failure.status = code
				break
			}
		}
		for _, output := range gjson.GetBytes(event, "response.output").Array() {
			failure.unsafe = failure.unsafe || openAIStreamItemHasVisibleOutput(output) || output.Get("type").String() == "function_call"
		}
	} else {
		var dialErr *openAIWSDialError
		if errors.As(err, &dialErr) && dialErr.StatusCode >= 400 {
			failure.kind, failure.httpStatus, failure.status = "http", dialErr.StatusCode, dialErr.StatusCode
			failure.body, failure.headers, failure.err = dialErr.ResponseBody, dialErr.ResponseHeaders, nil
		} else if a.status == http.StatusSwitchingProtocols {
			failure.kind, failure.httpStatus = "stream_read", a.status
		}
	}
	rule := gatewayRecoveryRule(a.c, a.account, failure)
	action := ""
	if rule != nil {
		action = rule.EffectiveAction()
	}
	transient := gatewayRecoveryTransient(failure)
	if action == model.ErrorActionRetry && failure.status == http.StatusTooManyRequests && !failure.unsafe {
		transient = true
	}
	if action == "" && transient {
		action = model.ErrorActionRetry
	}
	if action == "" {
		a.observation.mu.Lock()
		alreadyRecovering := a.observation.retries > 0
		a.observation.mu.Unlock()
		if alreadyRecovering && failure.err != nil {
			// An unclassified close/write error on a recovery connection cannot
			// restart the old payload repair or account-switch retry loops.
			a.item.Action, a.item.Outcome = model.ErrorActionRetry, "stopped_after_retry"
			recordGatewayRecoveryAttempt(a.c, a.observation, a.account, a.item, &failure, nil)
			a.recorded = true
			return gatewayRecoveryTerminal(a.c, failure, nil, model.ErrorActionRetry, "stopped_after_retry")
		}
		return nil
	}
	outcome := "rule_terminal"
	var delay time.Duration
	allowed := false
	if action == model.ErrorActionRetry {
		outcome = "not_retryable"
		if transient {
			outcome = "not_replayable"
			if a.replayable && !a.strictAffinity && !a.semantic && !outputStarted && gatewayRecoveryReplayAllowed(a.c, a.request, a.observation) {
				outcome = "exhausted"
				delay, allowed = reserveGatewayRecoveryRetry(a.c, a.request, a.account, a.observation, failure, a.options)
			}
		}
	}
	a.item.Action, a.item.Outcome = action, outcome
	if allowed {
		a.item.Outcome = "retrying"
	}
	recordGatewayRecoveryAttempt(a.c, a.observation, a.account, a.item, &failure, rule)
	a.recorded = true
	if allowed {
		SetGatewayRecoveryOutcome(a.c, "retrying")
		return &gatewayWSRecoveryRetry{delay: delay}
	}
	return gatewayRecoveryTerminal(a.c, failure, rule, action, outcome)
}

func (a *gatewayWSRecoveryAttempt) finish() {
	if a == nil || a.recorded {
		return
	}
	a.item.UpstreamStatus, a.item.Outcome, a.item.CostAccounting = a.status, "legacy_handling", "unknown"
	if a.accepted {
		a.item.Outcome, a.item.CostAccounting = "forwarded", "usage_log"
	}
	recordGatewayRecoveryAttempt(a.c, a.observation, a.account, a.item, nil, nil)
	if a.accepted {
		CompleteGatewayRequestObservation(a.c)
	}
	a.recorded = true
}

func waitGatewayWSRecovery(ctx context.Context, c *gin.Context, delay time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	downstream := ctx
	if c != nil && c.Request != nil {
		downstream = c.Request.Context()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-downstream.Done():
		return downstream.Err()
	case <-timer.C:
		return nil
	}
}

// Deadline applies to obtaining replayable output, not the remainder of a long
// successful generation. Never cancel a recovered stream after output begins.
func gatewayWSRecoveryReadTimeout(c *gin.Context, normal time.Duration, started time.Time, semantic bool) time.Duration {
	if semantic {
		return normal
	}
	remaining := defaultGatewayRecoveryOptions().prefixTimeout - time.Since(started)
	if c != nil && c.Request != nil {
		if observation := gatewayObservation(c.Request.Context()); observation != nil {
			observation.mu.Lock()
			deadline := observation.deadline
			observation.mu.Unlock()
			if !deadline.IsZero() && time.Until(deadline) < remaining {
				remaining = time.Until(deadline)
			}
		}
	}
	if remaining <= 0 {
		return time.Nanosecond
	}
	if normal <= 0 || remaining < normal {
		return remaining
	}
	return normal
}

// Bounded WS metadata prefix used by ingress; emitted only once, immediately
// before semantic output. Discarding this object discards failed response IDs.
type gatewayWSRecoveryPrelude struct {
	events    [][]byte
	bytes     int
	started   time.Time
	committed bool
}

func (p *gatewayWSRecoveryPrelude) write(payload []byte, eventType string, write func([]byte) error) error {
	if p.started.IsZero() {
		p.started = time.Now()
	}
	starts := gatewayRecoveryStreamStartsOutput(string(payload), eventType) || isOpenAIWSTerminalEvent(eventType) || eventType == "error"
	if !p.committed && !starts && p.bytes+len(payload) <= 64<<10 && time.Since(p.started) < 30*time.Second {
		p.events = append(p.events, append([]byte(nil), payload...))
		p.bytes += len(payload)
		return nil
	}
	p.committed = true
	for _, buffered := range p.events {
		if err := write(buffered); err != nil {
			return err
		}
	}
	p.events = nil
	p.bytes = 0
	return write(payload)
}

// Normalize only an abnormal peer EOF; orderly completion is recognized by the
// caller before this function. A peer close before any terminal event is not success.
func gatewayWSRecoveryReadError(err error) error {
	if err == nil {
		return io.ErrUnexpectedEOF
	}
	return err
}
