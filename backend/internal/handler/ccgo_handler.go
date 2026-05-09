package handler

import (
	"log"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"

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

func (h *CcgoHandler) ConnectAgent(c *gin.Context) {
	if h == nil || h.ccgoService == nil {
		response.ErrorFrom(c, service.ErrCcgoWorkspaceUnavailable)
		return
	}
	token := bearerToken(c.GetHeader("Authorization"))
	nonce := strings.TrimSpace(c.GetHeader("X-CCGO-Nonce"))
	if token == "" || nonce == "" {
		response.Unauthorized(c, "Missing ccgo agent credential")
		return
	}

	wsConn, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{
		CompressionMode: coderws.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer func() {
		_ = wsConn.CloseNow()
	}()
	wsConn.SetReadLimit(16 * 1024 * 1024)
	transport := hub.NewWebSocketTransport(wsConn)

	connection, err := h.ccgoService.RegisterAgentConnection(c.Request.Context(), token, nonce, transport)
	if err != nil {
		_ = wsConn.Close(coderws.StatusPolicyViolation, err.Error())
		return
	}
	log.Printf("[INFO] ccgo agent connected workspace_id=%d user_id=%d", connection.WorkspaceID, connection.UserID)
	<-c.Request.Context().Done()
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
