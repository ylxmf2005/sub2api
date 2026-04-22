package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerMonitorRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	if authenticated == nil || h == nil || h.Admin == nil || h.Admin.Dashboard == nil {
		return
	}

	monitor := authenticated.Group("/monitor")
	{
		monitor.GET("/snapshot-v2", h.Admin.Dashboard.GetSnapshotV2)
		monitor.GET("/users-trend", h.Admin.Dashboard.GetUserUsageTrend)
		monitor.GET("/users-ranking", h.Admin.Dashboard.GetUserSpendingRanking)
		monitor.GET("/user-breakdown", h.Admin.Dashboard.GetUserBreakdown)
		if h.Admin.Group != nil {
			monitor.GET("/groups/all", h.Admin.Group.GetAll)
		}
	}
}
