package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkV0165MigrationsFollowExistingSequence(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)

	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		index[entry.Name()] = i
	}

	expected := []string{
		"192_composite_model_routes.sql",
		"193_add_subscription_plan_currency.sql",
		"194_channel_image_input_price.sql",
		"195_usage_log_image_input_tokens.sql",
		"196_audit_logs.sql",
		"197_group_duplicate_operation_id.sql",
		"198_prompt_audit.sql",
		"199_prompt_audit_full_prompt.sql",
		"200_ops_ingress_reject_aggregates.sql",
		"201_auth_cache_invalidation_outbox.sql",
		"202_group_reasoning_effort_policy.sql",
		"203_alipay_mobile_precreate_deep_link.sql",
		"204_group_auth_cache_image_generation.sql",
		"205_add_usage_log_session_id.sql",
		"206_allow_live_usage_request_type.sql",
		"207_add_group_allow_live.sql",
		"208_add_users_email_alias_dedup_index_notx.sql",
	}
	previous := index["191_channel_monitor_grok_provider.sql"]
	require.NotZero(t, previous)
	for _, name := range expected {
		current, ok := index[name]
		require.True(t, ok, "expected migration %s", name)
		require.Greater(t, current, previous, "%s must follow the existing fork sequence", name)
		previous = current
	}

	for _, obsolete := range []string{
		"172_composite_model_routes.sql",
		"177_add_subscription_plan_currency.sql",
		"178_channel_image_input_price.sql",
		"179_usage_log_image_input_tokens.sql",
		"180_audit_logs.sql",
		"181_group_duplicate_operation_id.sql",
		"181_prompt_audit.sql",
		"182_prompt_audit_full_prompt.sql",
		"183_ops_ingress_reject_aggregates.sql",
		"184_auth_cache_invalidation_outbox.sql",
		"185_group_reasoning_effort_policy.sql",
		"186_alipay_mobile_precreate_deep_link.sql",
		"186_group_auth_cache_image_generation.sql",
		"187_add_usage_log_session_id.sql",
		"188_allow_live_usage_request_type.sql",
		"189_add_group_allow_live.sql",
		"190_add_users_email_alias_dedup_index_notx.sql",
	} {
		_, ok := index[obsolete]
		require.False(t, ok, "obsolete upstream migration alias must not be embedded: %s", obsolete)
	}
}
