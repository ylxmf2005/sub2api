package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SettlementPoolHandler struct {
	settlementService *service.SettlementPoolService
}

func NewSettlementPoolHandler(settlementService *service.SettlementPoolService) *SettlementPoolHandler {
	return &SettlementPoolHandler{settlementService: settlementService}
}

type UpdateSettlementPoolConfigRequest struct {
	TotalCost float64                      `json:"total_cost"`
	BaseRatio float64                      `json:"base_ratio"`
	MarketCap float64                      `json:"market_cap"`
	Tiers     []service.SettlementPoolTier `json:"tiers"`
}

type SyncSettlementPoolCandidatesRequest struct {
	UserIDs []int64 `json:"user_ids"`
}

type ForceJoinSettlementPoolRequest struct {
	UserIDs []int64 `json:"user_ids"`
	UserID  int64   `json:"user_id"`
}

type CreateSettlementPoolManualUsageAdjustmentRequest struct {
	UserID      int64   `json:"user_id"`
	AccountID   int64   `json:"account_id"`
	UsageAmount float64 `json:"usage_amount"`
	Reason      string  `json:"reason"`
}

func (h *SettlementPoolHandler) GetSummary(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	summary, err := h.settlementService.GetAdminSummary(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *SettlementPoolHandler) UpdateConfig(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	var req UpdateSettlementPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	summary, err := h.settlementService.UpdateConfig(c.Request.Context(), groupID, service.SettlementPoolConfigInput{
		TotalCost: req.TotalCost,
		BaseRatio: req.BaseRatio,
		MarketCap: req.MarketCap,
		Tiers:     req.Tiers,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *SettlementPoolHandler) SyncCandidates(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	var req SyncSettlementPoolCandidatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	summary, err := h.settlementService.SyncCandidates(c.Request.Context(), groupID, req.UserIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *SettlementPoolHandler) ForceJoinCurrentCycle(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	var req ForceJoinSettlementPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	userIDs := req.UserIDs
	if len(userIDs) == 0 && req.UserID > 0 {
		userIDs = []int64{req.UserID}
	}
	summary, err := h.settlementService.ForceJoinCurrentCycle(c.Request.Context(), groupID, userIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *SettlementPoolHandler) RemoveCurrentParticipant(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	summary, err := h.settlementService.RemoveCurrentParticipant(c.Request.Context(), groupID, userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *SettlementPoolHandler) CreateManualUsageAdjustment(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var req CreateSettlementPoolManualUsageAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)

	idempotencyPayload := struct {
		OperatorID int64                                            `json:"operator_id"`
		GroupID    int64                                            `json:"group_id"`
		Body       CreateSettlementPoolManualUsageAdjustmentRequest `json:"body"`
	}{
		OperatorID: subject.UserID,
		GroupID:    groupID,
		Body:       req,
	}
	executeAdminIdempotentJSON(c, "admin.settlement_pools.manual_usage_adjustments.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.settlementService.CreateManualUsageAdjustment(ctx, groupID, service.SettlementPoolManualUsageAdjustmentInput{
			UserID:      req.UserID,
			AccountID:   req.AccountID,
			UsageAmount: req.UsageAmount,
			Reason:      req.Reason,
			CreatedBy:   subject.UserID,
		})
	})
}

func (h *SettlementPoolHandler) StartNextCycle(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	summary, err := h.settlementService.StartNextCycle(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func parseSettlementGroupID(c *gin.Context) (int64, bool) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return 0, false
	}
	return groupID, true
}
