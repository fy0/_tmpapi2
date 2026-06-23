package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/supportticket"
	"github.com/Wei-Shaw/sub2api/ent/supportticketmessage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type supportTicketRepository struct {
	client *dbent.Client
}

func NewSupportTicketRepository(client *dbent.Client) service.SupportTicketRepository {
	return &supportTicketRepository{client: client}
}

func (r *supportTicketRepository) CreateWithMessage(ctx context.Context, ticket *service.SupportTicket, message *service.SupportTicketMessage) error {
	if ticket == nil || message == nil {
		return service.ErrSupportTicketInputRequired
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	created, err := client.SupportTicket.Create().
		SetUserID(ticket.UserID).
		SetUserEmail(ticket.UserEmail).
		SetUserName(ticket.UserName).
		SetTitle(ticket.Title).
		SetCategory(ticket.Category).
		SetStatus(ticket.Status).
		SetPriority(ticket.Priority).
		SetNillableCreatedBy(ticket.CreatedBy).
		SetNillableUpdatedBy(ticket.UpdatedBy).
		SetLastMessageAt(ticket.LastMessageAt).
		Save(txCtx)
	if err != nil {
		return err
	}

	message.TicketID = created.ID
	createdMessage, err := createSupportTicketMessage(txCtx, client, message)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	applySupportTicketEntityToService(ticket, created)
	applySupportTicketMessageEntityToService(message, createdMessage)
	return nil
}

func (r *supportTicketRepository) AddMessage(ctx context.Context, ticketID int64, message *service.SupportTicketMessage, status *string, actorID *int64) (*service.SupportTicket, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	ticket, err := client.SupportTicket.Query().
		Where(supportticket.IDEQ(ticketID)).
		Only(txCtx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupportTicketNotFound, nil)
	}

	message.TicketID = ticketID
	if _, err := createSupportTicketMessage(txCtx, client, message); err != nil {
		return nil, err
	}

	now := time.Now()
	update := client.SupportTicket.UpdateOneID(ticketID).
		SetLastMessageAt(now).
		SetUpdatedAt(now)
	if actorID != nil && *actorID > 0 {
		update.SetUpdatedBy(*actorID)
	}
	if status != nil && *status != "" {
		update.SetStatus(*status)
	}

	updated, err := update.Save(txCtx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupportTicketNotFound, nil)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	out := supportTicketEntityToService(updated)
	if out != nil && ticket != nil {
		out.CreatedAt = ticket.CreatedAt
	}
	return out, nil
}

func (r *supportTicketRepository) GetByID(ctx context.Context, id int64) (*service.SupportTicket, error) {
	ticket, err := r.client.SupportTicket.Query().
		Where(supportticket.IDEQ(id)).
		WithMessages(func(q *dbent.SupportTicketMessageQuery) {
			q.Order(dbent.Asc(supportticketmessage.FieldCreatedAt), dbent.Asc(supportticketmessage.FieldID))
		}).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupportTicketNotFound, nil)
	}
	return supportTicketEntityToService(ticket), nil
}

func (r *supportTicketRepository) List(ctx context.Context, params pagination.PaginationParams, filters service.SupportTicketListFilters) ([]service.SupportTicket, *pagination.PaginationResult, error) {
	q := r.client.SupportTicket.Query()

	if filters.UserID > 0 {
		q = q.Where(supportticket.UserIDEQ(filters.UserID))
	}
	if filters.Status != "" {
		q = q.Where(supportticket.StatusEQ(filters.Status))
	}
	if filters.Category != "" {
		q = q.Where(supportticket.CategoryEQ(filters.Category))
	}
	if filters.Search != "" {
		search := strings.TrimSpace(filters.Search)
		q = q.Where(
			supportticket.Or(
				supportticket.TitleContainsFold(search),
				supportticket.UserEmailContainsFold(search),
				supportticket.UserNameContainsFold(search),
			),
		)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.Offset(params.Offset()).Limit(params.Limit())
	for _, order := range supportTicketListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return supportTicketEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func (r *supportTicketRepository) Update(ctx context.Context, id int64, input service.UpdateSupportTicketInput) (*service.SupportTicket, error) {
	update := r.client.SupportTicket.UpdateOneID(id)
	if input.Status != nil {
		update.SetStatus(*input.Status)
	}
	if input.Priority != nil {
		update.SetPriority(*input.Priority)
	}
	if input.ActorID > 0 {
		update.SetUpdatedBy(input.ActorID)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupportTicketNotFound, nil)
	}
	return supportTicketEntityToService(updated), nil
}

func createSupportTicketMessage(ctx context.Context, client *dbent.Client, message *service.SupportTicketMessage) (*dbent.SupportTicketMessage, error) {
	if message == nil {
		return nil, service.ErrSupportTicketInputRequired
	}
	builder := client.SupportTicketMessage.Create().
		SetTicketID(message.TicketID).
		SetAuthorRole(message.AuthorRole).
		SetAuthorEmail(message.AuthorEmail).
		SetAuthorName(message.AuthorName).
		SetContent(message.Content)
	if message.AuthorID != nil {
		builder.SetAuthorID(*message.AuthorID)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create support ticket message: %w", err)
	}
	return created, nil
}

func supportTicketListOrder(params pagination.PaginationParams) (string, string) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	switch sortBy {
	case "title":
		return supportticket.FieldTitle, sortOrder
	case "status":
		return supportticket.FieldStatus, sortOrder
	case "category":
		return supportticket.FieldCategory, sortOrder
	case "priority":
		return supportticket.FieldPriority, sortOrder
	case "user_id":
		return supportticket.FieldUserID, sortOrder
	case "created_at":
		return supportticket.FieldCreatedAt, sortOrder
	case "updated_at":
		return supportticket.FieldUpdatedAt, sortOrder
	case "id":
		return supportticket.FieldID, sortOrder
	case "", "last_message_at":
		return supportticket.FieldLastMessageAt, sortOrder
	default:
		return supportticket.FieldLastMessageAt, pagination.SortOrderDesc
	}
}

func supportTicketListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	field, sortOrder := supportTicketListOrder(params)

	if sortOrder == pagination.SortOrderAsc {
		if field == supportticket.FieldID {
			return []func(*entsql.Selector){dbent.Asc(field)}
		}
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(supportticket.FieldID)}
	}

	if field == supportticket.FieldID {
		return []func(*entsql.Selector){dbent.Desc(field)}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(supportticket.FieldID)}
}

func applySupportTicketEntityToService(dst *service.SupportTicket, src *dbent.SupportTicket) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func applySupportTicketMessageEntityToService(dst *service.SupportTicketMessage, src *dbent.SupportTicketMessage) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.TicketID = src.TicketID
	dst.CreatedAt = src.CreatedAt
}

func supportTicketEntityToService(m *dbent.SupportTicket) *service.SupportTicket {
	if m == nil {
		return nil
	}

	out := &service.SupportTicket{
		ID:            m.ID,
		UserID:        m.UserID,
		UserEmail:     m.UserEmail,
		UserName:      m.UserName,
		Title:         m.Title,
		Category:      m.Category,
		Status:        m.Status,
		Priority:      m.Priority,
		CreatedBy:     m.CreatedBy,
		UpdatedBy:     m.UpdatedBy,
		LastMessageAt: m.LastMessageAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if len(m.Edges.Messages) > 0 {
		out.Messages = supportTicketMessageEntitiesToService(m.Edges.Messages)
	}
	return out
}

func supportTicketEntitiesToService(models []*dbent.SupportTicket) []service.SupportTicket {
	out := make([]service.SupportTicket, 0, len(models))
	for i := range models {
		if s := supportTicketEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

func supportTicketMessageEntityToService(m *dbent.SupportTicketMessage) *service.SupportTicketMessage {
	if m == nil {
		return nil
	}
	return &service.SupportTicketMessage{
		ID:          m.ID,
		TicketID:    m.TicketID,
		AuthorID:    m.AuthorID,
		AuthorRole:  m.AuthorRole,
		AuthorEmail: m.AuthorEmail,
		AuthorName:  m.AuthorName,
		Content:     m.Content,
		CreatedAt:   m.CreatedAt,
	}
}

func supportTicketMessageEntitiesToService(models []*dbent.SupportTicketMessage) []service.SupportTicketMessage {
	out := make([]service.SupportTicketMessage, 0, len(models))
	for i := range models {
		if s := supportTicketMessageEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
