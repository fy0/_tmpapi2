package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

type createInvoiceRequest struct {
	InvoiceTitle  string  `json:"invoice_title" binding:"required"`
	TaxNo         string  `json:"tax_no"`
	Note          string  `json:"note"`
	RedeemCodeIDs []int64 `json:"redeem_code_ids" binding:"required"`
}

func (h *InvoiceHandler) GetSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	summary, err := h.invoiceService.GetSummary(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *InvoiceHandler) ListRecharges(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	items, err := h.invoiceService.ListAvailableRecharges(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *InvoiceHandler) GetProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	profile, err := h.invoiceService.GetProfile(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *InvoiceHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, total, err := h.invoiceService.ListUserInvoices(c.Request.Context(), subject.UserID, service.InvoiceListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *InvoiceHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req createInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	invoice, err := h.invoiceService.CreateRequest(c.Request.Context(), service.CreateInvoiceRequestInput{
		UserID:        subject.UserID,
		InvoiceTitle:  req.InvoiceTitle,
		TaxNo:         req.TaxNo,
		Note:          req.Note,
		RedeemCodeIDs: req.RedeemCodeIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, invoice)
}

func (h *InvoiceHandler) Withdraw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}
	invoice, err := h.invoiceService.WithdrawRequest(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, invoice)
}

func (h *InvoiceHandler) Download(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}
	file, err := h.invoiceService.GetUserInvoiceFile(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	c.FileAttachment(file.FilePath, file.FileName)
}

func parseInvoiceID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}
