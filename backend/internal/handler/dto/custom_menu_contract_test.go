package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCustomMenuItemsPreservesExtensionFields(t *testing.T) {
	raw := `[
		{"id":"draw","label":"Draw","url":"https://draw.example.com/?key=[img-key]","page_slug":"draw-docs","visibility":"user","sort_order":2,"open_mode":"external_confirm","group":"services"},
		{"id":"admin","label":"Admin","url":"https://admin.example.com","visibility":"admin","sort_order":3,"open_mode":"external","group":"operations"}
	]`

	items := ParseCustomMenuItems(raw)
	require.Len(t, items, 2)
	require.Equal(t, "external_confirm", items[0].OpenMode)
	require.Equal(t, "services", items[0].Group)
	require.Equal(t, "draw-docs", items[0].PageSlug)
	require.Contains(t, items[0].URL, "[img-key]")

	visible := ParseUserVisibleMenuItems(raw)
	require.Len(t, visible, 1)
	require.Equal(t, "draw", visible[0].ID)
	require.Equal(t, "external_confirm", visible[0].OpenMode)
	require.Equal(t, "services", visible[0].Group)
}
