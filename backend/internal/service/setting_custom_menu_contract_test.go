package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterUserVisibleMenuItemsPreservesExtensionFields(t *testing.T) {
	raw := `[
		{"id":"draw","visibility":"user","open_mode":"external_confirm","group":"services","url":"https://draw.example.com/?key=[img-key]","future_field":{"enabled":true}},
		{"id":"admin","visibility":"admin","open_mode":"external","group":"operations","url":"https://admin.example.com"}
	]`

	filtered := filterUserVisibleMenuItems(raw)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(filtered, &items))
	require.Len(t, items, 1)
	require.Equal(t, "draw", items[0]["id"])
	require.Equal(t, "external_confirm", items[0]["open_mode"])
	require.Equal(t, "services", items[0]["group"])
	require.Contains(t, items[0]["url"], "[img-key]")
	require.Equal(t, map[string]any{"enabled": true}, items[0]["future_field"])
}
