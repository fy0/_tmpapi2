package domain

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SupportTicketStatusOpen     = "open"
	SupportTicketStatusPending  = "pending"
	SupportTicketStatusResolved = "resolved"
	SupportTicketStatusClosed   = "closed"
)

const (
	SupportTicketPriorityLow    = "low"
	SupportTicketPriorityNormal = "normal"
	SupportTicketPriorityHigh   = "high"
)

const (
	SupportTicketCategoryFeedback = "feedback"
	SupportTicketCategoryBug      = "bug"
	SupportTicketCategoryBilling  = "billing"
	SupportTicketCategoryAccount  = "account"
	SupportTicketCategoryOther    = "other"
)

const (
	SupportTicketAuthorRoleUser  = "user"
	SupportTicketAuthorRoleAdmin = "admin"
)

var ErrSupportTicketNotFound = infraerrors.NotFound("SUPPORT_TICKET_NOT_FOUND", "support ticket not found")

type SupportTicket struct {
	ID            int64
	UserID        int64
	UserEmail     string
	UserName      string
	Title         string
	Category      string
	Status        string
	Priority      string
	CreatedBy     *int64
	UpdatedBy     *int64
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Messages      []SupportTicketMessage
}

type SupportTicketMessage struct {
	ID          int64
	TicketID    int64
	AuthorID    *int64
	AuthorRole  string
	AuthorEmail string
	AuthorName  string
	Content     string
	CreatedAt   time.Time
}
