package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type SupportTicketHandler struct {
	svc *service.SupportTicketService
}

func NewSupportTicketHandler(svc *service.SupportTicketService) *SupportTicketHandler {
	return &SupportTicketHandler{svc: svc}
}

type CreateSupportTicketRequest struct {
	Title    string `json:"title" binding:"required"`
	Category string `json:"category"`
	Priority string `json:"priority"`
	Content  string `json:"content" binding:"required"`
}

type AddSupportTicketMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *SupportTicketHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, pageResult, err := h.svc.ListForUser(
		c.Request.Context(),
		subject.UserID,
		pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    c.DefaultQuery("sort_by", "last_message_at"),
			SortOrder: c.DefaultQuery("sort_order", "desc"),
		},
		supportTicketFiltersFromQuery(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, dto.SupportTicketsFromService(items), pageResult.Total, page, pageSize)
}

func (h *SupportTicketHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	ticketID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}

	ticket, err := h.svc.GetForUser(c.Request.Context(), subject.UserID, ticketID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.SupportTicketFromService(ticket))
}

func (h *SupportTicketHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req CreateSupportTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	ticket, err := h.svc.CreateForUser(c.Request.Context(), &service.CreateSupportTicketInput{
		UserID:   subject.UserID,
		Title:    req.Title,
		Category: req.Category,
		Priority: req.Priority,
		Content:  req.Content,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.SupportTicketFromService(ticket))
}

func (h *SupportTicketHandler) AddMessage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	ticketID, ok := parsePositiveIDParam(c, "id")
	if !ok {
		return
	}

	var req AddSupportTicketMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	ticket, err := h.svc.AddUserMessage(c.Request.Context(), subject.UserID, ticketID, req.Content)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.SupportTicketFromService(ticket))
}

func supportTicketFiltersFromQuery(c *gin.Context) service.SupportTicketListFilters {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	return service.SupportTicketListFilters{
		Status:   strings.TrimSpace(c.Query("status")),
		Category: strings.TrimSpace(c.Query("category")),
		Search:   search,
	}
}

func parsePositiveIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ID")
		return 0, false
	}
	return id, true
}
