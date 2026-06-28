package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	MaxCustomHomeLinks          = 20
	MaxCustomHomeLinkTitleLen   = 100
	MaxCustomHomeLinkDescLen    = 500
	MaxCustomHomeLinkURLLen     = 2048
	customHomeLinkSortOrderStep = 10
)

var (
	ErrCustomHomeLinkInvalid  = infraerrors.BadRequest("CUSTOM_HOME_LINK_INVALID", "custom home link is invalid")
	ErrCustomHomeLinksTooMany = infraerrors.BadRequest("CUSTOM_HOME_LINKS_TOO_MANY", "too many custom home links")
)

type CustomHomeLink struct {
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	OpenInNewWindow bool      `json:"open_in_new_window"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"-"`
	UpdatedAt       time.Time `json:"-"`
}

type CustomHomeLinkRepository interface {
	List(ctx context.Context) ([]CustomHomeLink, error)
	ReplaceAll(ctx context.Context, links []CustomHomeLink) ([]CustomHomeLink, error)
}

func (s *SettingService) SetCustomHomeLinkRepository(repo CustomHomeLinkRepository) {
	s.customHomeLinkRepo = repo
}

func (s *SettingService) ListCustomHomeLinks(ctx context.Context) ([]CustomHomeLink, error) {
	if s == nil || s.customHomeLinkRepo == nil {
		return []CustomHomeLink{}, nil
	}
	links, err := s.customHomeLinkRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list custom home links: %w", err)
	}
	return links, nil
}

func (s *SettingService) ReplaceCustomHomeLinks(ctx context.Context, links []CustomHomeLink) ([]CustomHomeLink, error) {
	if s == nil || s.customHomeLinkRepo == nil {
		return []CustomHomeLink{}, nil
	}
	normalized, err := NormalizeCustomHomeLinks(links)
	if err != nil {
		return nil, err
	}
	updated, err := s.customHomeLinkRepo.ReplaceAll(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("replace custom home links: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return updated, nil
}

func NormalizeCustomHomeLinks(links []CustomHomeLink) ([]CustomHomeLink, error) {
	if len(links) > MaxCustomHomeLinks {
		return nil, ErrCustomHomeLinksTooMany
	}
	out := make([]CustomHomeLink, 0, len(links))
	for i, item := range links {
		normalized, err := normalizeCustomHomeLink(item, i)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeCustomHomeLink(item CustomHomeLink, index int) (CustomHomeLink, error) {
	title := strings.TrimSpace(item.Title)
	description := strings.TrimSpace(item.Description)
	linkURL := strings.TrimSpace(item.URL)

	if title == "" {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "title", "message": "title is required"})
	}
	if len([]rune(title)) > MaxCustomHomeLinkTitleLen {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "title", "message": "title is too long"})
	}
	if len([]rune(description)) > MaxCustomHomeLinkDescLen {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "description", "message": "description is too long"})
	}
	if linkURL == "" {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "url", "message": "url is required"})
	}
	if len(linkURL) > MaxCustomHomeLinkURLLen {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "url", "message": "url is too long"})
	}
	if !isValidCustomHomeLinkURL(linkURL) {
		return CustomHomeLink{}, ErrCustomHomeLinkInvalid.WithMetadata(map[string]string{"field": "url", "message": "url must be an http(s) URL or an internal path"})
	}

	item.Title = title
	item.Description = description
	item.URL = linkURL
	item.SortOrder = (index + 1) * customHomeLinkSortOrderStep
	return item, nil
}

func isValidCustomHomeLinkURL(raw string) bool {
	if strings.HasPrefix(raw, "/") {
		return !strings.HasPrefix(raw, "//")
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return strings.TrimSpace(u.Host) != ""
}
