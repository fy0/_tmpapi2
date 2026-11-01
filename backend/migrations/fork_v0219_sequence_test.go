package migrations

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Fork v0.2.13 contains seven migrations absent from the matching upstream
// release. Reserve upstream prefixes 242-248 and assign one new numeric prefix
// per migration starting at 249; the next Fork migration starts at 257.
func TestForkV0219MigrationsFollowExistingSequence(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)

	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		index[entry.Name()] = i
	}
	previous, exists := index["241_add_typesafe_platform.sql"]
	require.True(t, exists, "expected final pre-upgrade Fork migration")

	expected := []struct {
		name     string
		upstream string
	}{
		{"249_drop_platform_check_constraints.sql", "242_drop_platform_check_constraints.sql"},
		{"250_group_i18n.sql", "242_group_i18n.sql"},
		{"251_gemini_cached_contents.sql", "243_gemini_cached_contents.sql"},
		{"252_gemini_cached_contents_channel_usage.sql", "244_gemini_cached_contents_channel_usage.sql"},
		{"253_gemini_cached_contents_mutation_lease.sql", "245_gemini_cached_contents_mutation_lease.sql"},
		{"254_enable_gemini_image_generation_groups.sql", "246_enable_gemini_image_generation_groups.sql"},
		{"255_scoped_admin_api_keys.sql", "247_scoped_admin_api_keys.sql"},
		{"256_group_image_count_billing.sql", "248_group_image_count_billing.sql"},
	}
	for i, item := range expected {
		current, ok := index[item.name]
		require.True(t, ok, "expected Fork migration %s", item.name)
		require.Greater(t, current, previous, "%s must follow its predecessor", item.name)
		previous = current
		require.NotContains(t, index, item.upstream, "unshifted upstream migration must not be embedded")
		prefix := strings.SplitN(item.name, "_", 2)[0]
		number, err := strconv.Atoi(prefix)
		require.NoError(t, err)
		require.Equal(t, 249+i, number, fmt.Sprintf("Fork migration %s must have its own ordinal", item.name))
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		number, err := strconv.Atoi(prefix)
		if err != nil {
			continue
		}
		require.False(t, number >= 242 && number <= 248,
			"upstream-only migration prefix %03d must remain reserved in the Fork: %s", number, entry.Name())
		if number >= 249 && number <= 256 {
			require.Equal(t, expected[number-249].name, entry.Name(),
				"Fork prefix %03d is already allocated; future migrations start at 257", number)
		}
	}
}
