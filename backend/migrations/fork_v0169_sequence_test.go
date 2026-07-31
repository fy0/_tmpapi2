package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForkV0169MigrationsFollowExistingSequence(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)

	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		index[entry.Name()] = i
	}

	previous, ok := index["208_add_users_email_alias_dedup_index_notx.sql"]
	require.True(t, ok, "expected previous fork migration")
	current, ok := index["209_passkey_credentials.sql"]
	require.True(t, ok, "expected renumbered passkey migration")
	require.Greater(t, current, previous, "passkey migration must follow the existing fork sequence")

	_, ok = index["191_passkey_credentials.sql"]
	require.False(t, ok, "obsolete upstream migration alias must not be embedded")
}
