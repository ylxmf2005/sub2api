package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type CcgoHandler struct {
	ccgoService *service.CcgoService
}

func NewCcgoHandler(ccgoService *service.CcgoService) *CcgoHandler {
	return &CcgoHandler{ccgoService: ccgoService}
}

type ccgoResolveWorkspaceRequest struct {
	CanonicalRoot    string `json:"canonical_root" binding:"required"`
	LocalRootDisplay string `json:"local_root_display" binding:"required"`
	LocalRootHash    string `json:"local_root_hash" binding:"required"`
	OS               string `json:"os" binding:"required"`
	PathStyle        string `json:"path_style" binding:"required"`
	DeviceID         string `json:"device_id" binding:"required"`
}

func (h *CcgoHandler) ResolveWorkspace(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req ccgoResolveWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	resolution, err := h.ccgoService.ResolveWorkspace(c.Request.Context(), service.CcgoResolveWorkspaceInput{
		UserID:           subject.UserID,
		CanonicalRoot:    req.CanonicalRoot,
		LocalRootDisplay: req.LocalRootDisplay,
		LocalRootHash:    req.LocalRootHash,
		OS:               req.OS,
		PathStyle:        req.PathStyle,
		DeviceID:         req.DeviceID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, resolution)
}
