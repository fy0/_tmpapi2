package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

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
	if c.Query("inline") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", file.FileName))
		c.File(file.FilePath)
		return
	}
	c.FileAttachment(file.FilePath, file.FileName)
}

func (h *InvoiceHandler) DeleteFile(c *gin.Context) {
	id, ok := parseInvoiceID(c)
	if !ok {
		return
	}
	invoice, err := h.invoiceService.DeleteIssuedFile(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, invoice)
}

func (h *InvoiceHandler) GetSettings(c *gin.Context) {
	settings, err := h.invoiceService.GetInvoiceSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

type updateInvoiceSettingsRequest struct {
	MinInvoiceAmount float64 `json:"min_invoice_amount"`
	MaxInvoiceAmount float64 `json:"max_invoice_amount"`
}

func (h *InvoiceHandler) UpdateSettings(c *gin.Context) {
	var req updateInvoiceSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.invoiceService.UpdateInvoiceSettings(c.Request.Context(), service.InvoiceSettings{
		MinInvoiceAmount: req.MinInvoiceAmount,
		MaxInvoiceAmount: req.MaxInvoiceAmount,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *InvoiceHandler) ExportPending(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 100 {
		keyword = keyword[:100]
	}
	minAgeHours, _ := strconv.Atoi(strings.TrimSpace(c.Query("min_age_hours")))
	items, err := h.invoiceService.ListPendingInvoicesForExport(c.Request.Context(), keyword, minAgeHours)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{
		"id",
		"user_id",
		"user_email",
		"invoice_title",
		"tax_no",
		"amount",
		"note",
		"recharge_count",
		"recharge_codes",
		"recharge_amounts",
		"created_at",
	}); err != nil {
		response.InternalError(c, "Failed to export invoices: "+err.Error())
		return
	}

	for _, invoice := range items {
		codes := make([]string, 0, len(invoice.Recharges))
		amounts := make([]string, 0, len(invoice.Recharges))
		for _, recharge := range invoice.Recharges {
			codes = append(codes, recharge.Code)
			amounts = append(amounts, fmt.Sprintf("%.2f", recharge.Value))
		}
		if err := writer.Write([]string{
			fmt.Sprintf("%d", invoice.ID),
			fmt.Sprintf("%d", invoice.UserID),
			invoice.UserEmail,
			invoice.InvoiceTitle,
			invoice.TaxNo,
			fmt.Sprintf("%.2f", invoice.Amount),
			invoice.Note,
			fmt.Sprintf("%d", len(invoice.Recharges)),
			strings.Join(codes, ";"),
			strings.Join(amounts, ";"),
			invoice.CreatedAt.Format("2006-01-02 15:04:05"),
		}); err != nil {
			response.InternalError(c, "Failed to export invoices: "+err.Error())
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.InternalError(c, "Failed to export invoices: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=pending_invoices.csv")
	c.Data(200, "text/csv", buf.Bytes())
}

func parseInvoiceID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return 0, false
	}
	return id, true
}
