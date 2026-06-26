package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/sysutil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles system-related operations
type SystemHandler struct {
	updateSvc            systemUpdateService
	lockSvc              *service.SystemOperationLockService
	userIDMaintenanceSvc userIDMaintenanceService
}

type systemUpdateService interface {
	CurrentVersion() string
	CheckUpdate(ctx context.Context, force bool) (*service.UpdateInfo, error)
	PerformUpdate(ctx context.Context) error
	Rollback() error
}

type userIDMaintenanceService interface {
	GetStatus(ctx context.Context) (*service.UserIDMaintenanceStatus, error)
	SetNextUserID(ctx context.Context, nextUserID int64) (*service.SetUserNextIDResult, error)
	ChangeUserID(ctx context.Context, req service.ChangeUserIDRequest) (*service.ChangeUserIDResult, error)
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(updateSvc systemUpdateService, lockSvc *service.SystemOperationLockService, maintenanceSvc ...userIDMaintenanceService) *SystemHandler {
	h := &SystemHandler{
		updateSvc: updateSvc,
		lockSvc:   lockSvc,
	}
	if len(maintenanceSvc) > 0 {
		h.userIDMaintenanceSvc = maintenanceSvc[0]
	}
	return h
}

// GetVersion returns the current version
// GET /api/v1/admin/system/version
func (h *SystemHandler) GetVersion(c *gin.Context) {
	response.Success(c, gin.H{
		"version": h.updateSvc.CurrentVersion(),
	})
}

// CheckUpdates checks for available updates
// GET /api/v1/admin/system/check-updates
func (h *SystemHandler) CheckUpdates(c *gin.Context) {
	force := c.Query("force") == "true"
	info, err := h.updateSvc.CheckUpdate(c.Request.Context(), force)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, info)
}

// PerformUpdate downloads and applies the update
// POST /api/v1/admin/system/update
func (h *SystemHandler) PerformUpdate(c *gin.Context) {
	operationID := buildSystemOperationID(c, "update")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.update", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		if err := h.updateSvc.PerformUpdate(ctx); err != nil {
			if errors.Is(err, service.ErrNoUpdateAvailable) {
				info, checkErr := h.updateSvc.CheckUpdate(ctx, false)
				if checkErr != nil {
					releaseReason = "SYSTEM_UPDATE_FAILED"
					return nil, checkErr
				}
				succeeded = true
				return gin.H{
					"message":            "Already up to date",
					"already_up_to_date": true,
					"current_version":    info.CurrentVersion,
					"latest_version":     info.LatestVersion,
					"operation_id":       lock.OperationID(),
				}, nil
			}
			releaseReason = "SYSTEM_UPDATE_FAILED"
			return nil, err
		}
		succeeded = true

		return gin.H{
			"message":      "Update completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

// Rollback restores the previous version
// POST /api/v1/admin/system/rollback
func (h *SystemHandler) Rollback(c *gin.Context) {
	operationID := buildSystemOperationID(c, "rollback")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.rollback", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		if err := h.updateSvc.Rollback(); err != nil {
			releaseReason = "SYSTEM_ROLLBACK_FAILED"
			return nil, err
		}
		succeeded = true

		return gin.H{
			"message":      "Rollback completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}, nil
	})
}

// RestartService restarts the systemd service
// POST /api/v1/admin/system/restart
func (h *SystemHandler) RestartService(c *gin.Context) {
	operationID := buildSystemOperationID(c, "restart")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.restart", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		// Schedule service restart in background after sending response
		// This ensures the client receives the success response before the service restarts
		go func() {
			// Wait a moment to ensure the response is sent
			time.Sleep(500 * time.Millisecond)
			sysutil.RestartServiceAsync()
		}()
		succeeded = true
		return gin.H{
			"message":      "Service restart initiated",
			"operation_id": lock.OperationID(),
		}, nil
	})
}

// GetUserIDMaintenanceStatus returns users.id sequence status.
// GET /api/v1/admin/system/user-id-maintenance
func (h *SystemHandler) GetUserIDMaintenanceStatus(c *gin.Context) {
	if h.userIDMaintenanceSvc == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable"))
		return
	}
	status, err := h.userIDMaintenanceSvc.GetStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

type setUserNextIDRequest struct {
	NextUserID int64 `json:"next_user_id"`
}

// SetUserNextID advances the users.id sequence.
// POST /api/v1/admin/system/user-id-maintenance/next-id
func (h *SystemHandler) SetUserNextID(c *gin.Context) {
	if h.userIDMaintenanceSvc == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable"))
		return
	}
	var req setUserNextIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REQUEST", err.Error()))
		return
	}

	operationID := buildSystemOperationID(c, "set-user-next-id")
	payload := gin.H{"operation_id": operationID, "next_user_id": req.NextUserID}
	executeAdminIdempotentJSON(c, "admin.system.user_id_maintenance.set_next_id", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		result, err := h.userIDMaintenanceSvc.SetNextUserID(ctx, req.NextUserID)
		if err != nil {
			return nil, err
		}
		succeeded = true
		return gin.H{
			"operation_id": lock.OperationID(),
			"result":       result,
		}, nil
	})
}

type changeUserIDRequest struct {
	OldUserID    int64  `json:"old_user_id"`
	NewUserID    int64  `json:"new_user_id"`
	Confirmation string `json:"confirmation"`
}

// ChangeUserID changes a real users.id primary key.
// POST /api/v1/admin/system/user-id-maintenance/change-user-id
func (h *SystemHandler) ChangeUserID(c *gin.Context) {
	if h.userIDMaintenanceSvc == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("USER_ID_MAINTENANCE_UNAVAILABLE", "user id maintenance is unavailable"))
		return
	}
	var req changeUserIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REQUEST", err.Error()))
		return
	}

	var operatorUserID int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		operatorUserID = subject.UserID
	}
	operationID := buildSystemOperationID(c, "change-user-id")
	payload := gin.H{
		"operation_id":     operationID,
		"old_user_id":      req.OldUserID,
		"new_user_id":      req.NewUserID,
		"operator_user_id": operatorUserID,
		"confirmation":     req.Confirmation,
	}
	executeAdminIdempotentJSON(c, "admin.system.user_id_maintenance.change_user_id", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		result, err := h.userIDMaintenanceSvc.ChangeUserID(ctx, service.ChangeUserIDRequest{
			OldUserID:      req.OldUserID,
			NewUserID:      req.NewUserID,
			OperatorUserID: operatorUserID,
			Confirmation:   req.Confirmation,
		})
		if err != nil {
			return nil, err
		}
		succeeded = true
		return gin.H{
			"operation_id": lock.OperationID(),
			"result":       result,
		}, nil
	})
}

func (h *SystemHandler) acquireSystemLock(
	ctx context.Context,
	operationID string,
) (*service.SystemOperationLock, func(string, bool), error) {
	if h.lockSvc == nil {
		return nil, nil, service.ErrIdempotencyStoreUnavail
	}
	lock, err := h.lockSvc.Acquire(ctx, operationID)
	if err != nil {
		return nil, nil, err
	}
	release := func(reason string, succeeded bool) {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.lockSvc.Release(releaseCtx, lock, succeeded, reason)
	}
	return lock, release, nil
}

func buildSystemOperationID(c *gin.Context, operation string) string {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		return "sysop-" + operation + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	actorScope := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	seed := operation + "|" + actorScope + "|" + c.FullPath() + "|" + key
	hash := service.HashIdempotencyKey(seed)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	return "sysop-" + hash
}
