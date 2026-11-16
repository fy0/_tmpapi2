package dto

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGatewayObservationDTOAdminOnlyAttemptDetails(t *testing.T) {
	usage := &service.UsageLog{UserID: 42, APIKeyID: 7, GatewayRequestMetadata: service.GatewayRequestMetadata{ConversationID: "conversation-1", ConversationIDSource: "header.conversation_id", AttemptCount: 2, Attempts: []service.GatewayAttemptMetadata{{Attempt: 1, AccountID: 87, Model: "internal-upstream-model", RuleID: 19, UpstreamStatus: 499, Outcome: "retrying"}}}}
	user := UsageLogFromService(usage)
	require.Equal(t, "conversation-1", user.ConversationID)
	require.Empty(t, user.Attempts)
	userJSON, err := json.Marshal(user)
	require.NoError(t, err)
	require.NotContains(t, string(userJSON), "internal-upstream-model")
	require.NotContains(t, string(userJSON), "rule_id")
	admin := UsageLogFromServiceAdmin(usage)
	require.Len(t, admin.Attempts, 1)
	require.Equal(t, 499, admin.Attempts[0].UpstreamStatus)
	require.Equal(t, "conversation-1", admin.ConversationID)
	require.Equal(t, "header.conversation_id", admin.ConversationIDSource)
	require.Len(t, usage.Attempts, 1, "redacting a DTO must not mutate stored metadata")
}
