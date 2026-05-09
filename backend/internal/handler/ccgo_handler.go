package handler

import (
	"context"
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

type ccgoStartWorkstationRequest struct {
	WorkspaceID int64 `json:"workspace_id" binding:"required"`
}

func (h *CcgoHandler) StartWorkstation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req ccgoStartWorkstationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.ccgoService.StartWorkstation(c.Request.Context(), service.CcgoStartWorkstationInput{
		UserID:      subject.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CcgoHandler) AttachWorkstation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	workspaceID, err := service.ParseCcgoWorkspaceID(c.Param("workspaceID"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	wsConn, err := coderws.Accept(c.Writer, c.Request, &coderws.AcceptOptions{
		CompressionMode: coderws.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer wsConn.CloseNow()
	wsConn.SetReadLimit(16 * 1024 * 1024)
	transport := hub.NewWebSocketTransport(wsConn)
	reader := hub.NewEnvelopeReader(c.Request.Context(), transport.Recv)
	writer := hub.NewEnvelopeWriter(c.Request.Context(), transport.Send)
	if err := h.ccgoService.AttachTerminal(c.Request.Context(), subject.UserID, workspaceID, reader, writer); err != nil {
		_ = wsConn.Close(closeStatusForAttachError(err), err.Error())
	}
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

func closeStatusForAttachError(err error) coderws.StatusCode {
	if err == nil || err == context.Canceled {
		return coderws.StatusNormalClosure
	}
	return coderws.StatusInternalError
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
