package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

type SupportTicket struct {
	ID            int64                  `json:"id"`
	UserID        int64                  `json:"user_id"`
	UserEmail     string                 `json:"user_email"`
	UserName      string                 `json:"user_name"`
	Title         string                 `json:"title"`
	Category      string                 `json:"category"`
	Status        string                 `json:"status"`
	Priority      string                 `json:"priority"`
	CreatedBy     *int64                 `json:"created_by,omitempty"`
	UpdatedBy     *int64                 `json:"updated_by,omitempty"`
	LastMessageAt int64                  `json:"last_message_at"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
	Messages      []SupportTicketMessage `json:"messages,omitempty"`
}

type SupportTicketMessage struct {
	ID          int64  `json:"id"`
	TicketID    int64  `json:"ticket_id"`
	AuthorID    *int64 `json:"author_id,omitempty"`
	AuthorRole  string `json:"author_role"`
	AuthorEmail string `json:"author_email"`
	AuthorName  string `json:"author_name"`
	Content     string `json:"content"`
	CreatedAt   int64  `json:"created_at"`
}

func SupportTicketFromService(ticket *service.SupportTicket) *SupportTicket {
	if ticket == nil {
		return nil
	}
	out := &SupportTicket{
		ID:            ticket.ID,
		UserID:        ticket.UserID,
		UserEmail:     ticket.UserEmail,
		UserName:      ticket.UserName,
		Title:         ticket.Title,
		Category:      ticket.Category,
		Status:        ticket.Status,
		Priority:      ticket.Priority,
		CreatedBy:     ticket.CreatedBy,
		UpdatedBy:     ticket.UpdatedBy,
		LastMessageAt: ticket.LastMessageAt.Unix(),
		CreatedAt:     ticket.CreatedAt.Unix(),
		UpdatedAt:     ticket.UpdatedAt.Unix(),
	}
	if len(ticket.Messages) > 0 {
		out.Messages = make([]SupportTicketMessage, 0, len(ticket.Messages))
		for i := range ticket.Messages {
			out.Messages = append(out.Messages, *SupportTicketMessageFromService(&ticket.Messages[i]))
		}
	}
	return out
}

func SupportTicketsFromService(items []service.SupportTicket) []SupportTicket {
	out := make([]SupportTicket, 0, len(items))
	for i := range items {
		if item := SupportTicketFromService(&items[i]); item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func SupportTicketMessageFromService(message *service.SupportTicketMessage) *SupportTicketMessage {
	if message == nil {
		return nil
	}
	return &SupportTicketMessage{
		ID:          message.ID,
		TicketID:    message.TicketID,
		AuthorID:    message.AuthorID,
		AuthorRole:  message.AuthorRole,
		AuthorEmail: message.AuthorEmail,
		AuthorName:  message.AuthorName,
		Content:     message.Content,
		CreatedAt:   message.CreatedAt.Unix(),
	}
}
