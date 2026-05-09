package handler

import (
	"context"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"

	"github.com/gin-gonic/gin"
)

type CcgoHandler struct {
	ccgoService        *service.CcgoService
	deviceLoginService ccgoDeviceLoginService
}

type ccgoDeviceLoginService interface {
	StartDeviceLogin(ctx context.Context, serverBaseURL, deviceID string) (*service.CcgoDeviceLoginStartResult, error)
	PollDeviceLogin(ctx context.Context, deviceCode string) (*service.CcgoDeviceLoginPollResult, error)
	ApproveDeviceLogin(ctx context.Context, userID int64, userCode string) (*service.CcgoDeviceLoginApproveResult, error)
}

func NewCcgoHandler(ccgoService *service.CcgoService) *CcgoHandler {
	return &CcgoHandler{ccgoService: ccgoService, deviceLoginService: ccgoService}
}

func (h *CcgoHandler) deviceLogin() ccgoDeviceLoginService {
	if h == nil {
		return nil
	}
	if h.deviceLoginService != nil {
		return h.deviceLoginService
	}
	return h.ccgoService
}

type ccgoResolveWorkspaceRequest struct {
	CanonicalRoot    string `json:"canonical_root" binding:"required"`
	LocalRootDisplay string `json:"local_root_display" binding:"required"`
	LocalRootHash    string `json:"local_root_hash" binding:"required"`
	OS               string `json:"os" binding:"required"`
	PathStyle        string `json:"path_style" binding:"required"`
	DeviceID         string `json:"device_id" binding:"required"`
}

type ccgoDeviceLoginStartRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

func (h *CcgoHandler) StartDeviceLogin(c *gin.Context) {
	var req ccgoDeviceLoginStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	loginSvc := h.deviceLogin()
	if loginSvc == nil {
		response.ErrorFrom(c, service.ErrCcgoWorkspaceUnavailable)
		return
	}
	result, err := loginSvc.StartDeviceLogin(c.Request.Context(), ccgoFrontendBaseURL(c), req.DeviceID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type ccgoDeviceLoginPollRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
}

func (h *CcgoHandler) PollDeviceLogin(c *gin.Context) {
	var req ccgoDeviceLoginPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	loginSvc := h.deviceLogin()
	if loginSvc == nil {
		response.ErrorFrom(c, service.ErrCcgoWorkspaceUnavailable)
		return
	}
	result, err := loginSvc.PollDeviceLogin(c.Request.Context(), req.DeviceCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type ccgoDeviceLoginApproveRequest struct {
	UserCode string `json:"user_code" binding:"required"`
}

func (h *CcgoHandler) ApproveDeviceLogin(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req ccgoDeviceLoginApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	loginSvc := h.deviceLogin()
	if loginSvc == nil {
		response.ErrorFrom(c, service.ErrCcgoWorkspaceUnavailable)
		return
	}
	result, err := loginSvc.ApproveDeviceLogin(c.Request.Context(), subject.UserID, req.UserCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
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

func (h *CcgoHandler) WorkstationStatus(c *gin.Context) {
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
	result, err := h.ccgoService.WorkstationStatus(c.Request.Context(), service.CcgoWorkstationStatusInput{
		UserID:      subject.UserID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type ccgoStopWorkstationRequest struct {
	Reason string `json:"reason,omitempty"`
}

func (h *CcgoHandler) StopWorkstation(c *gin.Context) {
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
	var req ccgoStopWorkstationRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	result, err := h.ccgoService.StopWorkstation(c.Request.Context(), service.CcgoStopWorkstationInput{
		UserID:      subject.UserID,
		WorkspaceID: workspaceID,
		Reason:      req.Reason,
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

func ccgoFrontendBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
		if scheme == "" {
			scheme = "https"
		}
		return (&url.URL{Scheme: scheme, Host: forwardedHost}).String()
	}
	if c.Request.URL != nil && c.Request.URL.Scheme != "" && c.Request.Host != "" {
		return (&url.URL{Scheme: c.Request.URL.Scheme, Host: c.Request.Host}).String()
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		host = c.GetHeader("Host")
	}
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return (&url.URL{Scheme: scheme, Host: host}).String()
	}
	if strings.Contains(host, "://") {
		parsed, err := url.Parse(host)
		if err == nil && parsed.Host != "" {
			return (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
		}
	}
	return (&url.URL{Scheme: scheme, Host: host}).String()
}
