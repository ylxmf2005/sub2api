package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
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

type SyncSettlementPoolParticipantsRequest struct {
	UserIDs []int64 `json:"user_ids"`
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

func (h *SettlementPoolHandler) SyncParticipants(c *gin.Context) {
	groupID, ok := parseSettlementGroupID(c)
	if !ok {
		return
	}
	var req SyncSettlementPoolParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	summary, err := h.settlementService.SyncParticipants(c.Request.Context(), groupID, req.UserIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
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
