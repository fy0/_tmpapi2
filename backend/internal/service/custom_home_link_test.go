package service

import (
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCustomHomeLinks(t *testing.T) {
	links, err := NormalizeCustomHomeLinks([]CustomHomeLink{
		{
			Title:           " Docs ",
			Description:     " API docs ",
			URL:             " https://example.com/docs ",
			OpenInNewWindow: true,
			SortOrder:       99,
		},
		{
			Title:           " Console ",
			URL:             " /console ",
			OpenInNewWindow: false,
		},
	})

	require.NoError(t, err)
	require.Equal(t, []CustomHomeLink{
		{
			Title:           "Docs",
			Description:     "API docs",
			URL:             "https://example.com/docs",
			OpenInNewWindow: true,
			SortOrder:       10,
		},
		{
			Title:           "Console",
			URL:             "/console",
			OpenInNewWindow: false,
			SortOrder:       20,
		},
	}, links)
}

func TestNormalizeCustomHomeLinksRejectsInvalidURLs(t *testing.T) {
	for _, rawURL := range []string{"//example.com", "javascript:alert(1)", "mailto:admin@example.com", "https://"} {
		t.Run(rawURL, func(t *testing.T) {
			links, err := NormalizeCustomHomeLinks([]CustomHomeLink{{
				Title: "Bad link",
				URL:   rawURL,
			}})

			require.Nil(t, links)
			require.Error(t, err)
			require.True(t, infraerrors.IsBadRequest(err))
			require.Equal(t, "CUSTOM_HOME_LINK_INVALID", infraerrors.Reason(err))
		})
	}
}

func TestNormalizeCustomHomeLinksRejectsInvalidFields(t *testing.T) {
	t.Run("missing title", func(t *testing.T) {
		links, err := NormalizeCustomHomeLinks([]CustomHomeLink{{
			Title: " ",
			URL:   "/docs",
		}})

		require.Nil(t, links)
		require.Error(t, err)
		require.True(t, infraerrors.IsBadRequest(err))
	})

	t.Run("too many", func(t *testing.T) {
		links := make([]CustomHomeLink, MaxCustomHomeLinks+1)
		for i := range links {
			links[i] = CustomHomeLink{Title: "Link", URL: "/docs"}
		}

		normalized, err := NormalizeCustomHomeLinks(links)

		require.Nil(t, normalized)
		require.Error(t, err)
		require.True(t, infraerrors.IsBadRequest(err))
		require.Equal(t, "CUSTOM_HOME_LINKS_TOO_MANY", infraerrors.Reason(err))
	})

	t.Run("too long description", func(t *testing.T) {
		links, err := NormalizeCustomHomeLinks([]CustomHomeLink{{
			Title:       "Long description",
			Description: strings.Repeat("x", MaxCustomHomeLinkDescLen+1),
			URL:         "/docs",
		}})

		require.Nil(t, links)
		require.Error(t, err)
		require.True(t, infraerrors.IsBadRequest(err))
	})
}
