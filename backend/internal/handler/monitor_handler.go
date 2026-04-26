package handler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	monitorLeaderboardLimit = 5000
	monitorUsersTrendLimit  = 12
	monitorModelLimit       = 1000
	monitorAccountLimit     = 1000
)

type MonitorHandler struct {
	dashboardService *service.DashboardService
	opsService       *service.OpsService
}

func NewMonitorHandler(dashboardService *service.DashboardService, opsService *service.OpsService) *MonitorHandler {
	return &MonitorHandler{
		dashboardService: dashboardService,
		opsService:       opsService,
	}
}

type monitorOverview struct {
	TotalRequests    int64   `json:"total_requests"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalActualCost  float64 `json:"total_actual_cost"`
	TodayRequests    int64   `json:"today_requests"`
	TodayTokens      int64   `json:"today_tokens"`
	ActiveUsers      int64   `json:"active_users"`
	TotalAccounts    int64   `json:"total_accounts"`
	HealthyAccounts  int64   `json:"healthy_accounts"`
	PeakHourLabel    string  `json:"peak_hour_label"`
	PeakHourRequests int64   `json:"peak_hour_requests"`
	StatsUpdatedAt   string  `json:"stats_updated_at"`
	StatsStale       bool    `json:"stats_stale"`
	AverageDuration  float64 `json:"average_duration_ms"`
}

type monitorLeaderboardItem struct {
	Rank         int     `json:"rank"`
	UserID       int64   `json:"user_id"`
	Email        string  `json:"email"`
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	ActualCost   float64 `json:"actual_cost"`
	SharePercent float64 `json:"share_percent"`
}

type monitorModelItem struct {
	Model        string  `json:"model"`
	Requests     int64   `json:"requests"`
	TotalTokens  int64   `json:"total_tokens"`
	ActualCost   float64 `json:"actual_cost"`
	SharePercent float64 `json:"share_percent"`
}

type monitorHourlyPoint struct {
	Label               string  `json:"label"`
	Requests            int64   `json:"requests"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	ActualCost          float64 `json:"actual_cost"`
}

type monitorPeakWindow struct {
	Label        string  `json:"label"`
	Requests     int64   `json:"requests"`
	TotalTokens  int64   `json:"total_tokens"`
	ActualCost   float64 `json:"actual_cost"`
	SharePercent float64 `json:"share_percent"`
}

type monitorAccountItem struct {
	AccountID             int64   `json:"account_id"`
	AccountName           string  `json:"account_name"`
	Platform              string  `json:"platform"`
	GroupID               int64   `json:"group_id"`
	GroupName             string  `json:"group_name"`
	Status                string  `json:"status"`
	StatusLabel           string  `json:"status_label"`
	CurrentInUse          int64   `json:"current_in_use"`
	MaxCapacity           int64   `json:"max_capacity"`
	LoadPercentage        float64 `json:"load_percentage"`
	WaitingInQueue        int64   `json:"waiting_in_queue"`
	IsAvailable           bool    `json:"is_available"`
	IsRateLimited         bool    `json:"is_rate_limited"`
	IsOverloaded          bool    `json:"is_overloaded"`
	HasError              bool    `json:"has_error"`
	ErrorMessage          string  `json:"error_message,omitempty"`
	RateLimitRemainingSec *int64  `json:"rate_limit_remaining_sec,omitempty"`
	OverloadRemainingSec  *int64  `json:"overload_remaining_sec,omitempty"`
}

type monitorPoolOption struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Platform    string  `json:"platform"`
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	ActualCost  float64 `json:"actual_cost"`
}

type monitorSnapshotResponse struct {
	GeneratedAt      string                           `json:"generated_at"`
	Overview         monitorOverview                  `json:"overview"`
	Leaderboard      []monitorLeaderboardItem         `json:"leaderboard"`
	HourlyTrend      []monitorHourlyPoint             `json:"hourly_trend"`
	UsersTrend       []usagestats.UserUsageTrendPoint `json:"users_trend"`
	PeakWindows      []monitorPeakWindow              `json:"peak_windows"`
	ModelStats       []usagestats.ModelStat           `json:"model_stats"`
	ModelShare       []monitorModelItem               `json:"model_share"`
	Accounts         []monitorAccountItem             `json:"accounts"`
	Pools            []monitorPoolOption              `json:"pools"`
	OpsEnabled       bool                             `json:"ops_enabled"`
	WindowLabel      string                           `json:"window_label"`
	AccountRange     string                           `json:"account_range"`
	TrendGranularity string                           `json:"trend_granularity"`
}

// GetSnapshot returns the authenticated shared monitor view.
// GET /api/v1/monitor/snapshot
func (h *MonitorHandler) GetSnapshot(c *gin.Context) {
	if h.dashboardService == nil {
		response.InternalError(c, "Monitor service not available")
		return
	}

	ctx := c.Request.Context()
	startTime, endTime, windowLabel, trendGranularity, err := parseMonitorRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	groupID, err := parseMonitorGroupID(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	now := time.Now().UTC()

	stats, err := h.dashboardService.GetUsageStatsWithFilters(ctx, usagestats.UsageLogFilters{
		GroupID:   groupID,
		StartTime: &startTime,
		EndTime:   &endTime,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	ranking, err := h.dashboardService.GetUserSpendingRankingWithGroup(ctx, startTime, endTime, groupID, monitorLeaderboardLimit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	hourlyTrend, err := h.dashboardService.GetUsageTrendWithFilters(ctx, startTime, endTime, trendGranularity, 0, 0, 0, groupID, "", nil, nil, nil)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	usersTrend, err := h.dashboardService.GetUserUsageTrendWithGroup(ctx, startTime, endTime, trendGranularity, groupID, monitorUsersTrendLimit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	modelStats, err := h.dashboardService.GetModelStatsWithFilters(ctx, startTime, endTime, 0, 0, 0, groupID, nil, nil, nil)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	groupStats, err := h.dashboardService.GetGroupStatsWithFilters(ctx, startTime, endTime, 0, 0, 0, 0, nil, nil, nil)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	accounts, opsEnabled := h.buildMonitorAccounts(ctx, groupID)
	peakHourLabel, peakHourRequests := monitorPeakHour(hourlyTrend)
	totalAccounts, healthyAccounts := monitorAccountCounts(accounts)

	response.Success(c, monitorSnapshotResponse{
		GeneratedAt: now.Format(time.RFC3339),
		Overview: monitorOverview{
			TotalRequests:    stats.TotalRequests,
			TotalTokens:      stats.TotalTokens,
			TotalActualCost:  stats.TotalActualCost,
			ActiveUsers:      int64(len(ranking.Ranking)),
			TotalAccounts:    totalAccounts,
			HealthyAccounts:  healthyAccounts,
			PeakHourLabel:    peakHourLabel,
			PeakHourRequests: peakHourRequests,
			AverageDuration:  stats.AverageDurationMs,
		},
		Leaderboard:      h.buildLeaderboard(ranking),
		HourlyTrend:      buildHourlyTrend(hourlyTrend, startTime, endTime, trendGranularity),
		UsersTrend:       usersTrend,
		PeakWindows:      buildPeakWindows(hourlyTrend, 3),
		ModelStats:       modelStats,
		ModelShare:       buildModelShare(modelStats, monitorModelLimit),
		Accounts:         accounts,
		Pools:            buildMonitorPoolOptions(groupStats, accounts, groupID),
		OpsEnabled:       opsEnabled,
		WindowLabel:      windowLabel,
		AccountRange:     "Live",
		TrendGranularity: trendGranularity,
	})
}

func parseMonitorRange(c *gin.Context) (time.Time, time.Time, string, string, error) {
	userTZ := c.Query("timezone")
	now := timezone.NowInUserLocation(userTZ)
	rangeKey := strings.ToLower(strings.TrimSpace(c.DefaultQuery("range", "7d")))
	requestedGranularity := strings.ToLower(strings.TrimSpace(c.Query("granularity")))

	switch rangeKey {
	case "1d":
		start := timezone.StartOfDayInUserLocation(now, userTZ)
		granularity, err := resolveMonitorGranularity("hour", requestedGranularity)
		return start.UTC(), now.UTC(), "Last 1 day", granularity, err
	case "", "7d":
		start := timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -6), userTZ)
		granularity, err := resolveMonitorGranularity("day", requestedGranularity)
		return start.UTC(), now.UTC(), "Last 7 days", granularity, err
	case "30d":
		start := timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -29), userTZ)
		granularity, err := resolveMonitorGranularity("day", requestedGranularity)
		return start.UTC(), now.UTC(), "Last 30 days", granularity, err
	case "all":
		granularity, err := resolveMonitorGranularity("day", requestedGranularity)
		return time.Unix(0, 0).UTC(), now.UTC(), "All time", granularity, err
	case "custom":
		startDateStr := strings.TrimSpace(c.Query("start_date"))
		endDateStr := strings.TrimSpace(c.Query("end_date"))
		if startDateStr == "" || endDateStr == "" {
			return time.Time{}, time.Time{}, "", "", errors.New("start_date and end_date are required for custom range")
		}

		start, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", errors.New("invalid start_date format, use YYYY-MM-DD")
		}
		end, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", errors.New("invalid end_date format, use YYYY-MM-DD")
		}
		if end.Before(start) {
			return time.Time{}, time.Time{}, "", "", errors.New("end_date must be on or after start_date")
		}

		endExclusive := end.AddDate(0, 0, 1)
		granularity := "day"
		if endExclusive.Sub(start) <= 48*time.Hour {
			granularity = "hour"
		}

		resolvedGranularity, err := resolveMonitorGranularity(granularity, requestedGranularity)
		return start.UTC(), endExclusive.UTC(), fmt.Sprintf("%s to %s", startDateStr, endDateStr), resolvedGranularity, err
	default:
		return time.Time{}, time.Time{}, "", "", errors.New("invalid range, use 1d, 7d, 30d, all, or custom")
	}
}

func resolveMonitorGranularity(defaultGranularity, requestedGranularity string) (string, error) {
	switch requestedGranularity {
	case "", "auto":
		return defaultGranularity, nil
	case "hour", "day":
		return requestedGranularity, nil
	default:
		return "", errors.New("invalid granularity, use auto, hour, or day")
	}
}

func parseMonitorGroupID(c *gin.Context) (int64, error) {
	value := strings.TrimSpace(c.Query("group_id"))
	if value == "" {
		return 0, nil
	}

	var groupID int64
	if _, err := fmt.Sscan(value, &groupID); err != nil || groupID < 0 {
		return 0, errors.New("invalid group_id")
	}

	return groupID, nil
}

func (h *MonitorHandler) buildLeaderboard(ranking *usagestats.UserSpendingRankingResponse) []monitorLeaderboardItem {
	if ranking == nil || len(ranking.Ranking) == 0 {
		return []monitorLeaderboardItem{}
	}

	totalActualCost := ranking.TotalActualCost
	if totalActualCost <= 0 {
		totalActualCost = 0
		for _, row := range ranking.Ranking {
			totalActualCost += row.ActualCost
		}
	}

	items := make([]monitorLeaderboardItem, 0, len(ranking.Ranking))
	for index, row := range ranking.Ranking {
		share := 0.0
		if totalActualCost > 0 {
			share = row.ActualCost / totalActualCost * 100
		}

		items = append(items, monitorLeaderboardItem{
			Rank:         index + 1,
			UserID:       row.UserID,
			Email:        row.Email,
			Requests:     row.Requests,
			Tokens:       row.Tokens,
			ActualCost:   row.ActualCost,
			SharePercent: share,
		})
	}

	return items
}

func buildHourlyTrend(points []usagestats.TrendDataPoint, startTime, endTime time.Time, granularity string) []monitorHourlyPoint {
	if len(points) == 0 {
		return []monitorHourlyPoint{}
	}

	filled := fillTrendPoints(points, startTime, endTime, granularity)
	items := make([]monitorHourlyPoint, 0, len(filled))
	for _, point := range filled {
		items = append(items, monitorHourlyPoint{
			Label:               point.Date,
			Requests:            point.Requests,
			InputTokens:         point.InputTokens,
			OutputTokens:        point.OutputTokens,
			CacheCreationTokens: point.CacheCreationTokens,
			CacheReadTokens:     point.CacheReadTokens,
			TotalTokens:         point.TotalTokens,
			Cost:                point.Cost,
			ActualCost:          point.ActualCost,
		})
	}

	return items
}

func fillTrendPoints(points []usagestats.TrendDataPoint, startTime, endTime time.Time, granularity string) []usagestats.TrendDataPoint {
	if len(points) == 0 {
		return []usagestats.TrendDataPoint{}
	}

	format := monitorTrendLabelFormat(granularity)
	if format == "" {
		return append([]usagestats.TrendDataPoint(nil), points...)
	}

	byLabel := make(map[string]usagestats.TrendDataPoint, len(points))
	for _, point := range points {
		byLabel[point.Date] = point
	}

	endBucket := endTime.Add(-time.Nanosecond)
	if endBucket.Before(startTime) {
		endBucket = startTime
	}

	// For "all", avoid manufacturing buckets all the way from 1970.
	rangeStart := startTime
	if startTime.Equal(time.Unix(0, 0).UTC()) {
		firstPointTime, err := time.ParseInLocation(format, points[0].Date, time.UTC)
		if err == nil {
			rangeStart = firstPointTime
		}
	}

	filled := make([]usagestats.TrendDataPoint, 0, len(points))
	for current := rangeStart; !current.After(endBucket); current = advanceMonitorBucket(current, granularity) {
		label := current.Format(format)
		if point, ok := byLabel[label]; ok {
			filled = append(filled, point)
			continue
		}

		filled = append(filled, usagestats.TrendDataPoint{
			Date: label,
		})
	}

	return filled
}

func monitorTrendLabelFormat(granularity string) string {
	switch granularity {
	case "hour":
		return "2006-01-02 15:00"
	case "day":
		return "2006-01-02"
	default:
		return ""
	}
}

func advanceMonitorBucket(current time.Time, granularity string) time.Time {
	switch granularity {
	case "hour":
		return current.Add(time.Hour)
	default:
		return current.AddDate(0, 0, 1)
	}
}

func buildPeakWindows(points []usagestats.TrendDataPoint, limit int) []monitorPeakWindow {
	if len(points) == 0 || limit <= 0 {
		return []monitorPeakWindow{}
	}

	totalRequests := int64(0)
	for _, point := range points {
		totalRequests += point.Requests
	}

	sorted := append([]usagestats.TrendDataPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Requests == sorted[j].Requests {
			return sorted[i].Date < sorted[j].Date
		}
		return sorted[i].Requests > sorted[j].Requests
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	items := make([]monitorPeakWindow, 0, len(sorted))
	for _, point := range sorted {
		share := 0.0
		if totalRequests > 0 {
			share = float64(point.Requests) / float64(totalRequests) * 100
		}

		items = append(items, monitorPeakWindow{
			Label:        point.Date,
			Requests:     point.Requests,
			TotalTokens:  point.TotalTokens,
			ActualCost:   point.ActualCost,
			SharePercent: share,
		})
	}

	return items
}

func buildModelShare(modelStats []usagestats.ModelStat, limit int) []monitorModelItem {
	if len(modelStats) == 0 {
		return []monitorModelItem{}
	}

	sorted := append([]usagestats.ModelStat(nil), modelStats...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ActualCost == sorted[j].ActualCost {
			if sorted[i].TotalTokens == sorted[j].TotalTokens {
				return sorted[i].Model < sorted[j].Model
			}
			return sorted[i].TotalTokens > sorted[j].TotalTokens
		}
		return sorted[i].ActualCost > sorted[j].ActualCost
	})

	totalActualCost := 0.0
	for _, row := range sorted {
		totalActualCost += row.ActualCost
	}

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}

	items := make([]monitorModelItem, 0, len(sorted))
	for _, row := range sorted {
		share := 0.0
		if totalActualCost > 0 {
			share = row.ActualCost / totalActualCost * 100
		}

		items = append(items, monitorModelItem{
			Model:        row.Model,
			Requests:     row.Requests,
			TotalTokens:  row.TotalTokens,
			ActualCost:   row.ActualCost,
			SharePercent: share,
		})
	}

	return items
}

func (h *MonitorHandler) buildMonitorAccounts(ctx context.Context, groupID int64) ([]monitorAccountItem, bool) {
	if h.opsService == nil {
		return []monitorAccountItem{}, false
	}

	if err := h.opsService.RequireMonitoringEnabled(ctx); err != nil {
		if errors.Is(err, service.ErrOpsDisabled) {
			return []monitorAccountItem{}, false
		}
		return []monitorAccountItem{}, false
	}

	var availabilityMap map[int64]*service.AccountAvailability
	var concurrencyMap map[int64]*service.AccountConcurrencyInfo

	_, _, availabilityMap, _, availabilityErr := h.opsService.GetAccountAvailabilityStats(ctx, "", nil)
	_, _, concurrencyMap, _, concurrencyErr := h.opsService.GetConcurrencyStats(ctx, "", nil)

	if availabilityErr != nil && concurrencyErr != nil {
		return []monitorAccountItem{}, false
	}

	keys := make(map[int64]struct{})
	for id := range availabilityMap {
		keys[id] = struct{}{}
	}
	for id := range concurrencyMap {
		keys[id] = struct{}{}
	}

	rows := make([]monitorAccountItem, 0, len(keys))
	for id := range keys {
		availability := availabilityMap[id]
		concurrency := concurrencyMap[id]

		row := monitorAccountItem{
			AccountID: id,
		}

		if availability != nil {
			row.AccountName = availability.AccountName
			row.Platform = availability.Platform
			row.GroupID = availability.GroupID
			row.GroupName = availability.GroupName
			row.Status = availability.Status
			row.IsAvailable = availability.IsAvailable
			row.IsRateLimited = availability.IsRateLimited
			row.IsOverloaded = availability.IsOverloaded
			row.HasError = availability.HasError
			row.ErrorMessage = availability.ErrorMessage
			row.RateLimitRemainingSec = availability.RateLimitRemainingSec
			row.OverloadRemainingSec = availability.OverloadRemainingSec
			row.StatusLabel = monitorAccountStatusLabel(availability)
		}

		if concurrency != nil {
			if row.AccountName == "" {
				row.AccountName = concurrency.AccountName
			}
			if row.Platform == "" {
				row.Platform = concurrency.Platform
			}
			if row.GroupID == 0 {
				row.GroupID = concurrency.GroupID
			}
			if row.GroupName == "" {
				row.GroupName = concurrency.GroupName
			}
			row.CurrentInUse = concurrency.CurrentInUse
			row.MaxCapacity = concurrency.MaxCapacity
			row.LoadPercentage = concurrency.LoadPercentage
			row.WaitingInQueue = concurrency.WaitingInQueue
		}

		if row.StatusLabel == "" {
			row.StatusLabel = "Unknown"
		}
		if groupID > 0 && row.GroupID != groupID {
			continue
		}

		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LoadPercentage == rows[j].LoadPercentage {
			return strings.ToLower(rows[i].AccountName) < strings.ToLower(rows[j].AccountName)
		}
		return rows[i].LoadPercentage > rows[j].LoadPercentage
	})

	if len(rows) > monitorAccountLimit {
		rows = rows[:monitorAccountLimit]
	}

	return rows, true
}

func monitorAccountStatusLabel(item *service.AccountAvailability) string {
	if item == nil {
		return ""
	}
	switch {
	case item.HasError:
		return "Error"
	case item.IsRateLimited:
		return "Rate Limited"
	case item.IsOverloaded:
		return "Overloaded"
	case item.IsAvailable:
		return "Healthy"
	default:
		return "Paused"
	}
}

func monitorPeakHour(points []usagestats.TrendDataPoint) (string, int64) {
	if len(points) == 0 {
		return "N/A", 0
	}

	top := points[0]
	for _, point := range points[1:] {
		if point.Requests > top.Requests {
			top = point
		}
	}

	label := top.Date
	if strings.TrimSpace(label) == "" {
		label = "N/A"
	}
	return label, top.Requests
}

func monitorAccountCounts(accounts []monitorAccountItem) (int64, int64) {
	total := int64(len(accounts))
	healthy := int64(0)
	for _, account := range accounts {
		if account.IsAvailable {
			healthy++
		}
	}
	return total, healthy
}

func buildMonitorPoolOptions(groupStats []usagestats.GroupStat, accounts []monitorAccountItem, selectedGroupID int64) []monitorPoolOption {
	byID := make(map[int64]monitorPoolOption, len(groupStats)+len(accounts))

	for _, stat := range groupStats {
		if stat.GroupID <= 0 {
			continue
		}
		byID[stat.GroupID] = monitorPoolOption{
			ID:          stat.GroupID,
			Name:        stat.GroupName,
			Requests:    stat.Requests,
			TotalTokens: stat.TotalTokens,
			ActualCost:  stat.ActualCost,
		}
	}

	for _, account := range accounts {
		if account.GroupID <= 0 {
			continue
		}
		option := byID[account.GroupID]
		option.ID = account.GroupID
		if option.Name == "" {
			option.Name = account.GroupName
		}
		if option.Platform == "" {
			option.Platform = account.Platform
		}
		byID[account.GroupID] = option
	}

	if selectedGroupID > 0 {
		if _, ok := byID[selectedGroupID]; !ok {
			byID[selectedGroupID] = monitorPoolOption{
				ID:   selectedGroupID,
				Name: fmt.Sprintf("Pool #%d", selectedGroupID),
			}
		}
	}

	options := make([]monitorPoolOption, 0, len(byID))
	for _, option := range byID {
		options = append(options, option)
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].ActualCost == options[j].ActualCost {
			if options[i].Requests == options[j].Requests {
				return strings.ToLower(options[i].Name) < strings.ToLower(options[j].Name)
			}
			return options[i].Requests > options[j].Requests
		}
		return options[i].ActualCost > options[j].ActualCost
	})

	return options
}
