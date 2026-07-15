package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkV0155MigrationsFollowExistingSequence(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)

	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		index[entry.Name()] = i
	}

	expected := []string{
		"185_add_usage_log_long_context_billing.sql",
		"186_add_usage_logs_api_key_latest_ip_index_notx.sql",
		"187_group_web_search_price_per_call.sql",
		"188_add_ops_system_logs_host.sql",
		"189_default_openai_long_context_billing.sql",
		"190_add_ops_system_logs_host_index_notx.sql",
		"191_channel_monitor_grok_provider.sql",
	}
	previous := index["184_allow_cyber_blocked_usage_request_type.sql"]
	require.NotZero(t, previous)
	for _, name := range expected {
		current, ok := index[name]
		require.True(t, ok, "expected migration %s", name)
		require.Greater(t, current, previous, "%s must follow the existing fork sequence", name)
		previous = current
	}

	for _, obsolete := range []string{
		"174_add_usage_log_long_context_billing.sql",
		"174_add_usage_logs_api_key_latest_ip_index_notx.sql",
		"174_group_web_search_price_per_call.sql",
		"175_add_ops_system_logs_host.sql",
		"175_default_openai_long_context_billing.sql",
		"175a_add_ops_system_logs_host_index_notx.sql",
		"176_channel_monitor_grok_provider.sql",
	} {
		_, ok := index[obsolete]
		require.False(t, ok, "obsolete upstream migration alias must not be embedded: %s", obsolete)
	}
}
