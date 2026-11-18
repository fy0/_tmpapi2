package migrations

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkV0223MigrationsFollowExistingSequence(t *testing.T) {
	files, err := fs.Glob(FS, "*.sql")
	require.NoError(t, err)
	order := make(map[string]int, len(files))
	for i, name := range files {
		order[name] = i
	}

	// Keep the batch after existing Fork SQL and in the final upstream order.
	previous, ok := order["259_gateway_conversation_indexes_notx.sql"]
	require.True(t, ok)
	for _, item := range []struct{ upstream, fork, sha256 string }{
		{"249_add_usage_logs_api_key_cost_index_notx.sql", "260_add_usage_logs_api_key_cost_index_notx.sql", "789d4d5a5c186531cb6edd65ea2ecc07f5e488272f7876c6f46240e2fed0c30b"},
		{"249_mcp_oauth.sql", "261_mcp_oauth.sql", "881ff2e8ecf430202c593c699b1441db1d2196ac2c312d8a3849acb6c97609e5"},
		{"250_openai_cache_write_ttl.sql", "262_openai_cache_write_ttl.sql", "c470116c49042425f5663c421f0aea2a6aa6eae75b7e489dc95c12fc303562b2"},
		{"251_group_force_openai_ultrafast.sql", "263_group_force_openai_ultrafast.sql", "f91352f2d3b1bffbbea5f6b7fc17e37c5ad64f04c3426cb7168778d10135a7bf"},
		{"251_group_usage_rollup_nonblocking_build.sql", "264_group_usage_rollup_nonblocking_build.sql", "da504646104cec14aa7bcf5a0af6f06f5cd40883459379ae915db27c1d4cec8b"},
		{"252_channel_pricing_ultrafast_multiplier.sql", "265_channel_pricing_ultrafast_multiplier.sql", "01eb249a933e5c3bef3a8e2106098abf56d20a5f2890fcc6490b1f83ec05687f"},
	} {
		require.NotContains(t, order, item.upstream)
		current, ok := order[item.fork]
		require.True(t, ok, "missing migration %s", item.fork)
		require.Equal(t, previous+1, current, "unexpected order for %s", item.fork)
		data, err := FS.ReadFile(item.fork)
		require.NoError(t, err)
		require.Equal(t, item.sha256, fmt.Sprintf("%x", sha256.Sum256(data)), "%s SQL changed", item.fork)
		previous = current
	}
}

func TestForkV0223PreservesHistoricalMigrations(t *testing.T) {
	files, err := fs.Glob(FS, "*.sql")
	require.NoError(t, err)
	hash := sha256.New()
	count := 0
	for _, name := range files {
		if name > "259_gateway_conversation_indexes_notx.sql" {
			continue
		}
		data, err := FS.ReadFile(name)
		require.NoError(t, err)
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		count++
	}
	// Computed from 1c9e81eb2: filename + NUL + unmodified SQL bytes + NUL,
	// for each of the 309 historical files in lexical migration-runner order.
	require.Equal(t, 309, count, "historical migrations must not be added or removed")
	require.Equal(t, "f69909f54690230f3ee951641121db24fab7c0b6b1a3b828725c1af59668ae07",
		fmt.Sprintf("%x", hash.Sum(nil)), "historical migration filenames or SQL bytes changed")
}
