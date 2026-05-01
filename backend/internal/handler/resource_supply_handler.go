package handler

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

func (h *ResourceSupplyHandler) Summary(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	summary, err := h.resourceSupplyService.GetUserSummary(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *ResourceSupplyHandler) Ledger(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	entries, total, err := h.resourceSupplyService.ListUserLedger(c.Request.Context(), userID, service.ResourceSupplyLedgerFilter{
		LedgerType: c.Query("type"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}

type submitOpenAIAPIKeyRequest struct {
	GroupID int64  `json:"group_id" binding:"required"`
	Name    string `json:"name"`
	APIKey  string `json:"api_key" binding:"required"`
	BaseURL string `json:"base_url"`
	ModelID string `json:"model_id"`
}

func (h *ResourceSupplyHandler) SubmitOpenAIAPIKey(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	var req submitOpenAIAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	executeUserIdempotentJSON(c, "user.resource_supply.submit_openai_api_key", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		account, err := h.resourceSupplyService.SubmitOpenAIAPIKey(ctx, service.ResourceSupplyOpenAIAPIKeyInput{
			UserID:  userID,
			GroupID: req.GroupID,
			Name:    req.Name,
			APIKey:  req.APIKey,
			BaseURL: req.BaseURL,
			ModelID: req.ModelID,
		})
		if err != nil {
			return nil, err
		}
		return resourceSupplyAccountResult(account), nil
	})
}

type generateOpenAIAuthURLRequest struct {
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

func (h *ResourceSupplyHandler) GenerateOpenAIAuthURL(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	var req generateOpenAIAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.resourceSupplyService.GenerateOpenAIAuthURL(c.Request.Context(), service.ResourceSupplyOpenAIGenerateAuthURLInput{
		UserID:      userID,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type exchangeOpenAICodeRequest struct {
	GroupID     int64  `json:"group_id" binding:"required"`
	Name        string `json:"name"`
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
	ModelID     string `json:"model_id"`
}

func (h *ResourceSupplyHandler) ExchangeOpenAICode(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	var req exchangeOpenAICodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	executeUserIdempotentJSON(c, "user.resource_supply.exchange_openai_code", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		account, err := h.resourceSupplyService.ExchangeOpenAICode(ctx, service.ResourceSupplyOpenAIExchangeCodeInput{
			UserID:      userID,
			GroupID:     req.GroupID,
			Name:        req.Name,
			SessionID:   req.SessionID,
			Code:        req.Code,
			State:       req.State,
			RedirectURI: req.RedirectURI,
			ProxyID:     req.ProxyID,
			ModelID:     req.ModelID,
		})
		if err != nil {
			return nil, err
		}
		return resourceSupplyAccountResult(account), nil
	})
}

type resourceSupplyAccountActionRequest struct {
	Reason string `json:"reason"`
}

func (h *ResourceSupplyHandler) PauseAccount(c *gin.Context) {
	h.ownerAccountAction(c, func(ctx context.Context, userID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.OwnerPauseAccount(ctx, userID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) RevokeAccount(c *gin.Context) {
	h.ownerAccountAction(c, func(ctx context.Context, userID, accountID int64, reason string) (*service.Account, error) {
		return h.resourceSupplyService.OwnerRevokeAccount(ctx, userID, accountID, reason)
	})
}

func (h *ResourceSupplyHandler) ownerAccountAction(c *gin.Context, fn func(context.Context, int64, int64, string) (*service.Account, error)) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req resourceSupplyAccountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	idempotencyPayload := gin.H{
		"account_id": accountID,
		"reason":     req.Reason,
	}
	executeUserIdempotentJSON(c, "user.resource_supply.account_action", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		account, err := fn(ctx, userID, accountID, req.Reason)
		if err != nil {
			return nil, err
		}
		return resourceSupplyAccountResult(account), nil
	})
}

func (h *ResourceSupplyHandler) Transfer(c *gin.Context) {
	userID, ok := currentResourceSupplyUserID(c)
	if !ok {
		return
	}
	executeUserIdempotentJSON(c, "user.resource_supply.transfer", gin.H{"transfer": "all"}, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.resourceSupplyService.TransferAvailableToBalance(ctx, userID, c.GetHeader("Idempotency-Key"))
	})
}

func currentResourceSupplyUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func resourceSupplyAccountResult(account *service.Account) gin.H {
	if account == nil {
		return gin.H{}
	}
	return gin.H{
		"id":                   account.ID,
		"name":                 account.Name,
		"platform":             account.Platform,
		"type":                 account.Type,
		"group_ids":            account.GroupIDs,
		"group_names":          []string{},
		"supply_source":        account.SupplySource,
		"supply_status":        account.SupplyStatus,
		"supply_status_reason": account.SupplyStatusReason,
		"schedulable":          account.Schedulable,
		"created_at":           account.CreatedAt,
		"updated_at":           account.UpdatedAt,
	}
}
