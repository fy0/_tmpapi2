package admin

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

func (h *InvoiceHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.invoiceService.ListAdminInvoices(c.Request.Context(), service.InvoiceListParams{
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

func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}
	invoice, err := h.invoiceService.GetAdminInvoice(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, invoice)
}

func (h *InvoiceHandler) Upload(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}

	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "invoice file is required")
		return
	}
	file, err := header.Open()
	if err != nil {
		response.BadRequest(c, "invalid invoice file")
		return
	}
	defer func() { _ = file.Close() }()

	invoice, err := h.invoiceService.UploadIssuedFile(c.Request.Context(), service.UploadInvoiceFileInput{
		InvoiceID:   id,
		AdminID:     subject.UserID,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Reader:      file,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, invoice)
}

func (h *InvoiceHandler) Download(c *gin.Context) {
	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}
	file, err := h.invoiceService.GetAdminInvoiceFile(c.Request.Context(), id)
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
