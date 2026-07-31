package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type routeContract struct {
	method string
	path   string
}

func TestForkExtensionRoutesRemainRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlers := &handler.Handlers{
		User:             &handler.UserHandler{},
		APIKey:           &handler.APIKeyHandler{},
		Usage:            &handler.UsageHandler{},
		Redeem:           &handler.RedeemHandler{},
		Invoice:          &handler.InvoiceHandler{},
		Subscription:     &handler.SubscriptionHandler{},
		Announcement:     &handler.AnnouncementHandler{},
		ChannelMonitor:   &handler.ChannelMonitorUserHandler{},
		Totp:             &handler.TotpHandler{},
		AvailableChannel: &handler.AvailableChannelHandler{},
		SupportTicket:    &handler.SupportTicketHandler{},
		Admin: &handler.AdminHandlers{
			SupportTicket: &adminhandler.SupportTicketHandler{},
			Invoice:       &adminhandler.InvoiceHandler{},
			Setting:       &adminhandler.SettingHandler{},
			System:        &adminhandler.SystemHandler{},
		},
	}

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	RegisterUserRoutes(v1, handlers, func(c *gin.Context) { c.Next() }, nil, nil, nil)
	admin := v1.Group("/admin")
	registerSupportTicketRoutes(admin, handlers)
	registerInvoiceRoutes(admin, handlers)
	registerSettingsRoutes(admin, handlers)
	registerSystemRoutes(admin, handlers)

	registered := make(map[routeContract]struct{})
	for _, route := range engine.Routes() {
		registered[routeContract{method: route.Method, path: route.Path}] = struct{}{}
	}

	expected := []routeContract{
		{http.MethodGet, "/api/v1/keys/img-key"},
		{http.MethodGet, "/api/v1/support-tickets"},
		{http.MethodPost, "/api/v1/support-tickets"},
		{http.MethodPost, "/api/v1/support-tickets/:id/messages"},
		{http.MethodGet, "/api/v1/invoices/summary"},
		{http.MethodGet, "/api/v1/invoices/profile"},
		{http.MethodGet, "/api/v1/invoices/recharges"},
		{http.MethodPost, "/api/v1/invoices"},
		{http.MethodPost, "/api/v1/invoices/:id/withdraw"},
		{http.MethodGet, "/api/v1/admin/support-tickets"},
		{http.MethodPost, "/api/v1/admin/support-tickets/:id/messages"},
		{http.MethodGet, "/api/v1/admin/invoices/export"},
		{http.MethodPut, "/api/v1/admin/invoices/settings"},
		{http.MethodPost, "/api/v1/admin/invoices/:id/upload"},
		{http.MethodGet, "/api/v1/admin/settings/custom-home-links"},
		{http.MethodPut, "/api/v1/admin/settings/custom-home-links"},
		{http.MethodGet, "/api/v1/admin/system/user-id-maintenance"},
		{http.MethodPost, "/api/v1/admin/system/user-id-maintenance/next-id"},
		{http.MethodPost, "/api/v1/admin/system/user-id-maintenance/change-user-id"},
	}
	for _, route := range expected {
		require.Contains(t, registered, route, "%s %s must remain registered", route.method, route.path)
	}
}
