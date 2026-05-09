package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterCcgoRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	v1.GET("/ccgo/agent/connect", h.Ccgo.ConnectAgent)
	v1.POST("/ccgo/device-login/start", h.Ccgo.StartDeviceLogin)
	v1.POST("/ccgo/device-login/poll", h.Ccgo.PollDeviceLogin)

	authenticated := v1.Group("/ccgo")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		authenticated.POST("/device-login/approve", h.Ccgo.ApproveDeviceLogin)
		authenticated.POST("/workspaces/resolve", h.Ccgo.ResolveWorkspace)
		authenticated.POST("/workstations/start", h.Ccgo.StartWorkstation)
		authenticated.GET("/workstations/:workspaceID/status", h.Ccgo.WorkstationStatus)
		authenticated.POST("/workstations/:workspaceID/stop", h.Ccgo.StopWorkstation)
		authenticated.GET("/workstations/:workspaceID/attach", h.Ccgo.AttachWorkstation)
	}
}
