package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SupportTicketStatusOpen     = domain.SupportTicketStatusOpen
	SupportTicketStatusPending  = domain.SupportTicketStatusPending
	SupportTicketStatusResolved = domain.SupportTicketStatusResolved
	SupportTicketStatusClosed   = domain.SupportTicketStatusClosed
)

const (
	SupportTicketPriorityLow    = domain.SupportTicketPriorityLow
	SupportTicketPriorityNormal = domain.SupportTicketPriorityNormal
	SupportTicketPriorityHigh   = domain.SupportTicketPriorityHigh
)

const (
	SupportTicketCategoryFeedback = domain.SupportTicketCategoryFeedback
	SupportTicketCategoryBug      = domain.SupportTicketCategoryBug
	SupportTicketCategoryBilling  = domain.SupportTicketCategoryBilling
	SupportTicketCategoryAccount  = domain.SupportTicketCategoryAccount
	SupportTicketCategoryOther    = domain.SupportTicketCategoryOther
)

const (
	SupportTicketAuthorRoleUser  = domain.SupportTicketAuthorRoleUser
	SupportTicketAuthorRoleAdmin = domain.SupportTicketAuthorRoleAdmin
)

const (
	SupportTicketEntryVisibilityAll   = "all"
	SupportTicketEntryVisibilityAdmin = "admin"
)

var (
	ErrSupportTicketNotFound        = domain.ErrSupportTicketNotFound
	ErrSupportTicketInputRequired   = infraerrors.BadRequest("SUPPORT_TICKET_INPUT_REQUIRED", "support ticket input is required")
	ErrSupportTicketTitleInvalid    = infraerrors.BadRequest("SUPPORT_TICKET_TITLE_INVALID", "support ticket title is invalid")
	ErrSupportTicketContentRequired = infraerrors.BadRequest("SUPPORT_TICKET_CONTENT_REQUIRED", "support ticket content is required")
	ErrSupportTicketStatusInvalid   = infraerrors.BadRequest("SUPPORT_TICKET_STATUS_INVALID", "support ticket status is invalid")
	ErrSupportTicketPriorityInvalid = infraerrors.BadRequest("SUPPORT_TICKET_PRIORITY_INVALID", "support ticket priority is invalid")
	ErrSupportTicketCategoryInvalid = infraerrors.BadRequest("SUPPORT_TICKET_CATEGORY_INVALID", "support ticket category is invalid")
	ErrSupportTicketUserRequired    = infraerrors.BadRequest("SUPPORT_TICKET_USER_REQUIRED", "support ticket user is required")
	ErrSupportTicketDisabledForUser = infraerrors.Forbidden("SUPPORT_TICKET_USER_ENTRY_DISABLED", "support ticket entry is admin-only")
)

type SupportTicket = domain.SupportTicket

type SupportTicketMessage = domain.SupportTicketMessage

type SupportTicketListFilters struct {
	UserID   int64
	Status   string
	Category string
	Search   string
}

type CreateSupportTicketInput struct {
	UserID   int64
	Title    string
	Category string
	Priority string
	Content  string
}

type AddSupportTicketMessageInput struct {
	TicketID   int64
	AuthorID   int64
	AuthorRole string
	Content    string
	Status     *string
}

type UpdateSupportTicketInput struct {
	Status   *string
	Priority *string
	ActorID  int64
}

type SupportTicketRepository interface {
	CreateWithMessage(ctx context.Context, ticket *SupportTicket, message *SupportTicketMessage) error
	AddMessage(ctx context.Context, ticketID int64, message *SupportTicketMessage, status *string, actorID *int64) (*SupportTicket, error)
	GetByID(ctx context.Context, id int64) (*SupportTicket, error)
	List(ctx context.Context, params pagination.PaginationParams, filters SupportTicketListFilters) ([]SupportTicket, *pagination.PaginationResult, error)
	Update(ctx context.Context, id int64, input UpdateSupportTicketInput) (*SupportTicket, error)
}

type SupportTicketEntryVisibilityReader interface {
	GetSupportTicketEntryVisibility(ctx context.Context) string
}

type SupportTicketService struct {
	repo       SupportTicketRepository
	userRepo   UserRepository
	visibility SupportTicketEntryVisibilityReader
}

func NewSupportTicketService(repo SupportTicketRepository, userRepo UserRepository, settingService *SettingService) *SupportTicketService {
	return &SupportTicketService{
		repo:       repo,
		userRepo:   userRepo,
		visibility: settingService,
	}
}

func (s *SupportTicketService) CanUserAccess(ctx context.Context) bool {
	return normalizeSupportTicketEntryVisibility(s.currentEntryVisibility(ctx)) == SupportTicketEntryVisibilityAll
}

func (s *SupportTicketService) CreateForUser(ctx context.Context, input *CreateSupportTicketInput) (*SupportTicket, error) {
	if !s.CanUserAccess(ctx) {
		return nil, ErrSupportTicketDisabledForUser
	}
	return s.create(ctx, input)
}

func (s *SupportTicketService) CreateAsAdmin(ctx context.Context, input *CreateSupportTicketInput) (*SupportTicket, error) {
	return s.create(ctx, input)
}

func (s *SupportTicketService) create(ctx context.Context, input *CreateSupportTicketInput) (*SupportTicket, error) {
	if input == nil {
		return nil, ErrSupportTicketInputRequired
	}
	if input.UserID <= 0 {
		return nil, ErrSupportTicketUserRequired
	}

	title := strings.TrimSpace(input.Title)
	if title == "" || len(title) > 200 {
		return nil, ErrSupportTicketTitleInvalid
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrSupportTicketContentRequired
	}

	category := normalizeSupportTicketCategory(input.Category)
	if !isValidSupportTicketCategory(category) {
		return nil, ErrSupportTicketCategoryInvalid
	}

	priority := normalizeSupportTicketPriority(input.Priority)
	if !isValidSupportTicketPriority(priority) {
		return nil, ErrSupportTicketPriorityInvalid
	}

	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	actorID := input.UserID
	ticket := &SupportTicket{
		UserID:        user.ID,
		UserEmail:     strings.TrimSpace(user.Email),
		UserName:      strings.TrimSpace(user.Username),
		Title:         title,
		Category:      category,
		Status:        SupportTicketStatusOpen,
		Priority:      priority,
		CreatedBy:     &actorID,
		UpdatedBy:     &actorID,
		LastMessageAt: now,
	}
	message := &SupportTicketMessage{
		AuthorID:    &actorID,
		AuthorRole:  SupportTicketAuthorRoleUser,
		AuthorEmail: ticket.UserEmail,
		AuthorName:  ticket.UserName,
		Content:     content,
	}

	if err := s.repo.CreateWithMessage(ctx, ticket, message); err != nil {
		return nil, err
	}
	ticket.Messages = []SupportTicketMessage{*message}
	return ticket, nil
}

func (s *SupportTicketService) ListForUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters SupportTicketListFilters) ([]SupportTicket, *pagination.PaginationResult, error) {
	if !s.CanUserAccess(ctx) {
		return nil, nil, ErrSupportTicketDisabledForUser
	}
	if userID <= 0 {
		return nil, nil, ErrSupportTicketUserRequired
	}
	filters.UserID = userID
	return s.list(ctx, params, filters)
}

func (s *SupportTicketService) ListForAdmin(ctx context.Context, params pagination.PaginationParams, filters SupportTicketListFilters) ([]SupportTicket, *pagination.PaginationResult, error) {
	return s.list(ctx, params, filters)
}

func (s *SupportTicketService) list(ctx context.Context, params pagination.PaginationParams, filters SupportTicketListFilters) ([]SupportTicket, *pagination.PaginationResult, error) {
	filters.Status = strings.TrimSpace(filters.Status)
	if filters.Status != "" && !isValidSupportTicketStatus(filters.Status) {
		return nil, nil, ErrSupportTicketStatusInvalid
	}
	filters.Category = strings.TrimSpace(filters.Category)
	if filters.Category != "" && !isValidSupportTicketCategory(filters.Category) {
		return nil, nil, ErrSupportTicketCategoryInvalid
	}
	filters.Search = strings.TrimSpace(filters.Search)
	if len(filters.Search) > 200 {
		filters.Search = filters.Search[:200]
	}
	return s.repo.List(ctx, params, filters)
}

func (s *SupportTicketService) GetForUser(ctx context.Context, userID, ticketID int64) (*SupportTicket, error) {
	if !s.CanUserAccess(ctx) {
		return nil, ErrSupportTicketDisabledForUser
	}
	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.UserID != userID {
		return nil, ErrSupportTicketNotFound
	}
	return ticket, nil
}

func (s *SupportTicketService) GetForAdmin(ctx context.Context, ticketID int64) (*SupportTicket, error) {
	return s.repo.GetByID(ctx, ticketID)
}

func (s *SupportTicketService) AddUserMessage(ctx context.Context, userID, ticketID int64, content string) (*SupportTicket, error) {
	if !s.CanUserAccess(ctx) {
		return nil, ErrSupportTicketDisabledForUser
	}

	ticket, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.UserID != userID {
		return nil, ErrSupportTicketNotFound
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrSupportTicketContentRequired
	}

	authorID := userID
	message := &SupportTicketMessage{
		AuthorID:    &authorID,
		AuthorRole:  SupportTicketAuthorRoleUser,
		AuthorEmail: ticket.UserEmail,
		AuthorName:  ticket.UserName,
		Content:     content,
	}
	status := SupportTicketStatusOpen
	return s.repo.AddMessage(ctx, ticketID, message, &status, &authorID)
}

func (s *SupportTicketService) AddAdminMessage(ctx context.Context, adminID, ticketID int64, content string, status *string) (*SupportTicket, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrSupportTicketContentRequired
	}

	var normalizedStatus *string
	if status != nil {
		v := strings.TrimSpace(*status)
		if v != "" {
			if !isValidSupportTicketStatus(v) {
				return nil, ErrSupportTicketStatusInvalid
			}
			normalizedStatus = &v
		}
	}

	admin, err := s.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, err
	}

	authorID := adminID
	message := &SupportTicketMessage{
		AuthorID:    &authorID,
		AuthorRole:  SupportTicketAuthorRoleAdmin,
		AuthorEmail: strings.TrimSpace(admin.Email),
		AuthorName:  strings.TrimSpace(admin.Username),
		Content:     content,
	}
	return s.repo.AddMessage(ctx, ticketID, message, normalizedStatus, &authorID)
}

func (s *SupportTicketService) Update(ctx context.Context, ticketID int64, input UpdateSupportTicketInput) (*SupportTicket, error) {
	if input.ActorID <= 0 {
		return nil, ErrSupportTicketUserRequired
	}
	if input.Status != nil {
		v := strings.TrimSpace(*input.Status)
		if !isValidSupportTicketStatus(v) {
			return nil, ErrSupportTicketStatusInvalid
		}
		input.Status = &v
	}
	if input.Priority != nil {
		v := normalizeSupportTicketPriority(*input.Priority)
		if !isValidSupportTicketPriority(v) {
			return nil, ErrSupportTicketPriorityInvalid
		}
		input.Priority = &v
	}
	return s.repo.Update(ctx, ticketID, input)
}

func (s *SupportTicketService) currentEntryVisibility(ctx context.Context) string {
	if s.visibility == nil {
		return SupportTicketEntryVisibilityAll
	}
	return s.visibility.GetSupportTicketEntryVisibility(ctx)
}

func normalizeSupportTicketEntryVisibility(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case SupportTicketEntryVisibilityAdmin:
		return SupportTicketEntryVisibilityAdmin
	default:
		return SupportTicketEntryVisibilityAll
	}
}

func normalizeSupportTicketCategory(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case SupportTicketCategoryBug:
		return SupportTicketCategoryBug
	case SupportTicketCategoryBilling:
		return SupportTicketCategoryBilling
	case SupportTicketCategoryAccount:
		return SupportTicketCategoryAccount
	case SupportTicketCategoryOther:
		return SupportTicketCategoryOther
	default:
		return SupportTicketCategoryFeedback
	}
}

func normalizeSupportTicketPriority(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case SupportTicketPriorityLow:
		return SupportTicketPriorityLow
	case SupportTicketPriorityHigh:
		return SupportTicketPriorityHigh
	default:
		return SupportTicketPriorityNormal
	}
}

func isValidSupportTicketCategory(category string) bool {
	switch category {
	case SupportTicketCategoryFeedback, SupportTicketCategoryBug, SupportTicketCategoryBilling, SupportTicketCategoryAccount, SupportTicketCategoryOther:
		return true
	default:
		return false
	}
}

func isValidSupportTicketPriority(priority string) bool {
	switch priority {
	case SupportTicketPriorityLow, SupportTicketPriorityNormal, SupportTicketPriorityHigh:
		return true
	default:
		return false
	}
}

func isValidSupportTicketStatus(status string) bool {
	switch status {
	case SupportTicketStatusOpen, SupportTicketStatusPending, SupportTicketStatusResolved, SupportTicketStatusClosed:
		return true
	default:
		return false
	}
}
