package admin

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ResourceSupplyHandler struct {
	resourceSupplyService *service.ResourceSupplyService
}

func NewResourceSupplyHandler(resourceSupplyService *service.ResourceSupplyService) *ResourceSupplyHandler {
	return &ResourceSupplyHandler{resourceSupplyService: resourceSupplyService}
}

func (h *ResourceSupplyHandler) Ledger(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.ResourceSupplyAdminLedgerFilter{
		OwnerUserID:  queryInt64Ptr(c, "owner_user_id"),
		CallerUserID: queryInt64Ptr(c, "caller_user_id"),
		GroupID:      queryInt64Ptr(c, "group_id"),
		AccountID:    queryInt64Ptr(c, "account_id"),
		LedgerType:   c.Query("type"),
		RequestID:    c.Query("request_id"),
		Page:         page,
		PageSize:     pageSize,
	}
	entries, total, err := h.resourceSupplyService.ListAdminLedger(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}

type adminResourceSupplyAccountActionRequest struct {
	Reason string `json:"reason"`
}

func (h *ResourceSupplyHandler) ApproveAccount(c *gin.Context) {
	h.adminAccountAction(c, "admin.resource_supply.approve", func(ctx context.Context, adminUserID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.AdminApproveAccount(ctx, adminUserID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) RejectAccount(c *gin.Context) {
	h.adminAccountAction(c, "admin.resource_supply.reject", func(ctx context.Context, adminUserID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.AdminRejectAccount(ctx, adminUserID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) PauseAccount(c *gin.Context) {
	h.adminAccountAction(c, "admin.resource_supply.pause", func(ctx context.Context, adminUserID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.AdminPauseAccount(ctx, adminUserID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) ResumeAccount(c *gin.Context) {
	h.adminAccountAction(c, "admin.resource_supply.resume", func(ctx context.Context, adminUserID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.AdminResumeAccount(ctx, adminUserID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) adminAccountAction(c *gin.Context, scope string, fn func(context.Context, int64, int64, string) (*service.Account, error)) {
	adminUserID, ok := currentAdminResourceSupplyUserID(c)
	if !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req adminResourceSupplyAccountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	idempotencyPayload := gin.H{
		"account_id": accountID,
		"reason":     req.Reason,
	}
	executeAdminIdempotentJSON(c, scope, idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		account, err := fn(ctx, adminUserID, accountID, req.Reason)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"id":                   account.ID,
			"supply_owner_user_id": account.SupplyOwnerUserID,
			"supply_source":        account.SupplySource,
			"supply_status":        account.SupplyStatus,
			"supply_status_reason": account.SupplyStatusReason,
			"schedulable":          account.Schedulable,
			"updated_at":           account.UpdatedAt,
		}, nil
	})
}

type adminResourceSupplyAdjustmentRequest struct {
	OwnerUserID int64   `json:"owner_user_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Note        string  `json:"note" binding:"required"`
}

func (h *ResourceSupplyHandler) AdjustBalance(c *gin.Context) {
	adminUserID, ok := currentAdminResourceSupplyUserID(c)
	if !ok {
		return
	}
	var req adminResourceSupplyAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	executeAdminIdempotentJSON(c, "admin.resource_supply.adjust", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.resourceSupplyService.AdjustBalance(ctx, service.ResourceSupplyAdjustmentInput{
			OwnerUserID: req.OwnerUserID,
			AdminUserID: adminUserID,
			Amount:      req.Amount,
			Note:        req.Note,
		})
	})
}

func currentAdminResourceSupplyUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Admin user not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func queryInt64Ptr(c *gin.Context, key string) *int64 {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil
	}
	return &value
}
