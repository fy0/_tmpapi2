package repository

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGatewayObservationUsageAndErrorInsertPreserveMetadata(t *testing.T) {
	now := time.Now().UTC()
	meta := service.GatewayRequestMetadata{ConversationID: "client-controlled ' ID", ConversationIDSource: "header.conversation_id", DownstreamRequestID: "request-1", RequestStartedAt: &now, AttemptCount: 2, RecoveryOutcome: "recovered", Attempts: []service.GatewayAttemptMetadata{{Attempt: 1, AccountID: 87, UpstreamStatus: 499, Outcome: "retrying", CostAccounting: "unknown"}}}
	log := &service.UsageLog{GatewayRequestMetadata: meta, UserID: 42, APIKeyID: 7, AccountID: 87, RequestID: "request-1", Model: "gpt-5", CreatedAt: now}
	prepared := prepareUsageLogInsert(log)
	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
	require.Equal(t, []string{"text", "text", "jsonb"}, usageLogInsertArgTypes[len(usageLogInsertArgTypes)-3:])
	id, err := driver.DefaultParameterConverter.ConvertValue(prepared.args[len(prepared.args)-3])
	require.NoError(t, err)
	require.Equal(t, meta.ConversationID, id)
	source, err := driver.DefaultParameterConverter.ConvertValue(prepared.args[len(prepared.args)-2])
	require.NoError(t, err)
	require.Equal(t, meta.ConversationIDSource, source)
	encoded, err := driver.DefaultParameterConverter.ConvertValue(prepared.args[len(prepared.args)-1])
	require.NoError(t, err)
	var decoded service.GatewayRequestMetadata
	encodedJSON, ok := encoded.(string)
	require.True(t, ok)
	require.NoError(t, json.Unmarshal([]byte(encodedJSON), &decoded))
	require.Equal(t, meta.ConversationID, decoded.ConversationID)
	require.Equal(t, 499, decoded.Attempts[0].UpstreamStatus)
	errorArgs := opsInsertErrorLogArgs(&service.OpsInsertErrorLogInput{GatewayRequestMetadata: meta})
	require.Len(t, errorArgs, 41)
	require.Equal(t, sql.NullString{String: meta.ConversationID, Valid: true}, errorArgs[38])
	require.Equal(t, sql.NullString{String: meta.ConversationIDSource, Valid: true}, errorArgs[39])
	errorMetadata, ok := errorArgs[40].(sql.NullString)
	require.True(t, ok)
	require.Contains(t, errorMetadata.String, `"upstream_status":499`)
	require.NotContains(t, insertOpsErrorLogSQL, meta.ConversationID, "client-controlled IDs must be parameters, never SQL")
	for _, column := range []string{"conversation_id", "conversation_id_source", "request_metadata"} {
		require.Contains(t, usageLogSelectColumns, column)
		require.Contains(t, insertOpsErrorLogSQL, column)
	}
	query, args := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	require.Contains(t, query, "request_metadata")
	require.Len(t, args, len(prepared.args))
	require.NotContains(t, query, meta.ConversationID)
}

func TestGatewayObservationUnknownHistoricalMetadataRemainsNull(t *testing.T) {
	prepared := prepareUsageLogInsert(&service.UsageLog{UserID: 42, APIKeyID: 7, RequestID: "old-request", Model: "gpt-5", CreatedAt: time.Now()})
	for _, arg := range prepared.args[len(prepared.args)-3:] {
		value, err := driver.DefaultParameterConverter.ConvertValue(arg)
		require.NoError(t, err)
		require.Nil(t, value)
	}
	args := opsInsertErrorLogArgs(&service.OpsInsertErrorLogInput{})
	for _, arg := range args[len(args)-3:] {
		value, ok := arg.(sql.NullString)
		require.True(t, ok)
		require.False(t, value.Valid)
	}
	decoded := service.DecodeGatewayRequestMetadata(`{"conversation_id":"stale-json-id","attempt_count":2}`, "", "")
	require.Empty(t, decoded.ConversationID, "indexed NULL must not manufacture a historical ID")
	require.Equal(t, 2, decoded.AttemptCount)
	require.NotContains(t, insertOpsErrorLogSQL, "request_headers")
}
