package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/customhomelink"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type customHomeLinkRepository struct {
	client *dbent.Client
}

func NewCustomHomeLinkRepository(client *dbent.Client) service.CustomHomeLinkRepository {
	return &customHomeLinkRepository{client: client}
}

func (r *customHomeLinkRepository) List(ctx context.Context) ([]service.CustomHomeLink, error) {
	items, err := r.client.CustomHomeLink.Query().
		Order(dbent.Asc(customhomelink.FieldSortOrder), dbent.Asc(customhomelink.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return customHomeLinkEntitiesToService(items), nil
}

func (r *customHomeLinkRepository) ReplaceAll(ctx context.Context, links []service.CustomHomeLink) ([]service.CustomHomeLink, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	if _, err := client.CustomHomeLink.Delete().Exec(txCtx); err != nil {
		return nil, err
	}

	if len(links) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return []service.CustomHomeLink{}, nil
	}

	builders := make([]*dbent.CustomHomeLinkCreate, 0, len(links))
	for _, link := range links {
		builders = append(builders, client.CustomHomeLink.Create().
			SetTitle(link.Title).
			SetDescription(link.Description).
			SetURL(link.URL).
			SetOpenInNewWindow(link.OpenInNewWindow).
			SetSortOrder(link.SortOrder))
	}
	created, err := client.CustomHomeLink.CreateBulk(builders...).Save(txCtx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return customHomeLinkEntitiesToService(created), nil
}

func customHomeLinkEntityToService(m *dbent.CustomHomeLink) *service.CustomHomeLink {
	if m == nil {
		return nil
	}
	return &service.CustomHomeLink{
		ID:              m.ID,
		Title:           m.Title,
		Description:     m.Description,
		URL:             m.URL,
		OpenInNewWindow: m.OpenInNewWindow,
		SortOrder:       m.SortOrder,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func customHomeLinkEntitiesToService(models []*dbent.CustomHomeLink) []service.CustomHomeLink {
	out := make([]service.CustomHomeLink, 0, len(models))
	for _, model := range models {
		if item := customHomeLinkEntityToService(model); item != nil {
			out = append(out, *item)
		}
	}
	return out
}
