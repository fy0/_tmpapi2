package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayRecoveryUnclassifiedSecondTransportFaultIsTerminal(t *testing.T) {
	c, request, account, options := recoveryTestContext(t, `{"model":"m","input":"x"}`)
	calls := 0
	_, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return recoveryResponse(499, "", false), nil
		}
		return nil, errors.New("custom transport unavailable")
	}, options)
	var final *UpstreamFailoverError
	require.ErrorAs(t, err, &final)
	require.True(t, final.RecoveryHandled)
	require.False(t, final.ShouldRetryNextAccount())
	require.Equal(t, http.StatusServiceUnavailable, final.ClientStatusCode)
	require.Equal(t, 2, calls)
	meta := GatewayRequestMetadataFromContext(c.Request.Context())
	require.Len(t, meta.Attempts, 2)
	require.Equal(t, "stopped_after_retry", meta.Attempts[1].Outcome)
}

func TestGatewayRecoveryWSUnclassifiedSecondFaultCannotRestartOuterLoop(t *testing.T) {
	c, _, account, _ := recoveryTestContext(t, `{"input":"x"}`)
	payload := []byte(`{"input":"x","store":true}`)
	first := beginGatewayWSRecoveryAttempt(c, account, payload, false)
	err := first.recover(nil, nil, []byte(`{"type":"response.failed","response":{"error":{"type":"server_error"}}}`), false)
	var retry *gatewayWSRecoveryRetry
	require.ErrorAs(t, err, &retry)
	second := beginGatewayWSRecoveryAttempt(c, account, payload, false)
	second.status = http.StatusSwitchingProtocols
	err = second.recover(errors.New("custom closed websocket"), nil, nil, false)
	var final *UpstreamFailoverError
	require.ErrorAs(t, err, &final)
	require.True(t, final.RecoveryHandled)
	require.False(t, final.ShouldRetryNextAccount())
	require.Equal(t, http.StatusServiceUnavailable, final.ClientStatusCode)
	require.Equal(t, 101, final.RecoveryUpstreamStatus)
	require.Len(t, GatewayRequestMetadataFromContext(c.Request.Context()).Attempts, 2)
}

func TestGatewayRecoveryLegacyHandlerOwnsOpsEvent(t *testing.T) {
	c, request, account, options := recoveryTestContext(t, `{"input":"x"}`)
	resp, err := recoverOpenAIUpstream(c, request, account, func(*http.Request) (*http.Response, error) {
		return recoveryResponse(400, `{"error":{"type":"invalid_request_error","message":"bad"}}`, false), nil
	}, options)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	events, _ := c.Get(OpsUpstreamErrorsKey)
	require.Empty(t, events, "legacy handler emits its original proxy-attributed event once")
	meta := GatewayRequestMetadataFromContext(c.Request.Context())
	require.Len(t, meta.Attempts, 1)
	require.Equal(t, "legacy_handling", meta.Attempts[0].Outcome)
}
