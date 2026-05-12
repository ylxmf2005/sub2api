package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupHandlerCreateSettlementPoolAddsCreatorAsCandidate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	settlementRepo := &createGroupSettlementPoolRepo{}
	groupRepo := &createGroupSettlementGroupRepo{
		group: &service.Group{
			ID:               200,
			Name:             "settlement",
			Status:           service.StatusActive,
			SubscriptionType: service.SubscriptionTypeSettlementPool,
		},
	}
	settlementSvc := service.NewSettlementPoolService(settlementRepo, groupRepo, nil)
	handler := NewGroupHandler(adminSvc, nil, nil, settlementSvc)

	router := gin.New()
	router.POST("/groups", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 77})
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"settlement","platform":"anthropic","subscription_type":"settlement_pool"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(200), settlementRepo.syncedGroupID)
	require.Equal(t, []int64{77}, settlementRepo.syncedUserIDs)
}

type createGroupSettlementPoolRepo struct {
	syncedGroupID int64
	syncedUserIDs []int64
	participants  []service.SettlementPoolParticipant
}

func (r *createGroupSettlementPoolRepo) GetConfig(context.Context, int64) (*service.SettlementPoolConfig, error) {
	panic("unexpected GetConfig")
}

func (r *createGroupSettlementPoolRepo) EnsureActiveCycle(_ context.Context, groupID int64, defaults *service.SettlementPoolConfig) (*service.SettlementPoolConfig, *service.SettlementPoolCycle, error) {
	if defaults == nil {
		defaults = service.DefaultSettlementPoolConfig(groupID)
	}
	return defaults, &service.SettlementPoolCycle{
		ID:        900,
		GroupID:   groupID,
		Status:    service.SettlementPoolCycleStatusActive,
		StartedAt: time.Now().Add(-time.Hour),
		TotalCost: 0,
		BaseRatio: defaults.BaseRatio,
		MarketCap: defaults.MarketCap,
		Tiers:     defaults.Tiers,
	}, nil
}

func (r *createGroupSettlementPoolRepo) UpsertConfig(context.Context, *service.SettlementPoolConfig) error {
	panic("unexpected UpsertConfig")
}

func (r *createGroupSettlementPoolRepo) SetActiveCycleID(context.Context, int64, *int64) error {
	panic("unexpected SetActiveCycleID")
}

func (r *createGroupSettlementPoolRepo) GetActiveCycle(context.Context, int64) (*service.SettlementPoolCycle, error) {
	panic("unexpected GetActiveCycle")
}

func (r *createGroupSettlementPoolRepo) CreateCycle(context.Context, *service.SettlementPoolCycle) error {
	panic("unexpected CreateCycle")
}

func (r *createGroupSettlementPoolRepo) UpdateActiveCycleConfig(context.Context, int64, service.SettlementPoolConfigInput) error {
	panic("unexpected UpdateActiveCycleConfig")
}

func (r *createGroupSettlementPoolRepo) LockCycle(context.Context, int64, time.Time, *service.SettlementPoolEstimate) error {
	panic("unexpected LockCycle")
}

func (r *createGroupSettlementPoolRepo) RotateCycle(context.Context, int64, int64, time.Time, *service.SettlementPoolEstimate, *service.SettlementPoolCycle) error {
	panic("unexpected RotateCycle")
}

func (r *createGroupSettlementPoolRepo) ListCycles(context.Context, int64, int) ([]service.SettlementPoolCycle, error) {
	return nil, nil
}

func (r *createGroupSettlementPoolRepo) ListCyclesForUser(context.Context, int64, int) ([]service.SettlementPoolCycle, error) {
	panic("unexpected ListCyclesForUser")
}

func (r *createGroupSettlementPoolRepo) ListCandidates(context.Context, int64) ([]service.SettlementPoolParticipant, error) {
	return append([]service.SettlementPoolParticipant(nil), r.participants...), nil
}

func (r *createGroupSettlementPoolRepo) SyncCandidates(_ context.Context, groupID int64, userIDs []int64) error {
	r.syncedGroupID = groupID
	r.syncedUserIDs = append([]int64(nil), userIDs...)
	r.participants = r.participants[:0]
	for _, userID := range userIDs {
		r.participants = append(r.participants, service.SettlementPoolParticipant{
			UserID: userID,
			Status: service.StatusActive,
		})
	}
	return nil
}

func (r *createGroupSettlementPoolRepo) IsCandidate(context.Context, int64, int64) (bool, error) {
	panic("unexpected IsCandidate")
}

func (r *createGroupSettlementPoolRepo) ListCandidateGroupIDs(context.Context, int64) ([]int64, error) {
	panic("unexpected ListCandidateGroupIDs")
}

func (r *createGroupSettlementPoolRepo) ListCycleParticipants(context.Context, int64) ([]service.SettlementPoolParticipant, error) {
	return nil, nil
}

func (r *createGroupSettlementPoolRepo) JoinCurrentCycle(context.Context, int64, int64) error {
	panic("unexpected JoinCurrentCycle")
}

func (r *createGroupSettlementPoolRepo) ForceJoinCurrentCycle(context.Context, int64, []int64) error {
	panic("unexpected ForceJoinCurrentCycle")
}

func (r *createGroupSettlementPoolRepo) RemoveCurrentParticipant(context.Context, int64, int64) error {
	panic("unexpected RemoveCurrentParticipant")
}

func (r *createGroupSettlementPoolRepo) IsCurrentParticipant(context.Context, int64, int64) (bool, error) {
	panic("unexpected IsCurrentParticipant")
}

func (r *createGroupSettlementPoolRepo) ListCurrentParticipantGroupIDs(context.Context, int64) ([]int64, error) {
	panic("unexpected ListCurrentParticipantGroupIDs")
}

func (r *createGroupSettlementPoolRepo) SumUsageByUsers(context.Context, int64, []int64, time.Time, *time.Time) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

func (r *createGroupSettlementPoolRepo) SumManualUsageByUsers(context.Context, int64, []int64) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

func (r *createGroupSettlementPoolRepo) SumManualUsageByAccounts(context.Context, int64, []int64) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

func (r *createGroupSettlementPoolRepo) CreateManualUsageAdjustment(context.Context, *service.SettlementPoolManualUsageAdjustment) error {
	panic("unexpected CreateManualUsageAdjustment")
}

func (r *createGroupSettlementPoolRepo) ListManualUsageAdjustments(context.Context, int64) ([]service.SettlementPoolManualUsageAdjustment, error) {
	return []service.SettlementPoolManualUsageAdjustment{}, nil
}

func (r *createGroupSettlementPoolRepo) ListEnabledAccountUsage(context.Context, int64, time.Time, *time.Time) ([]service.SettlementPoolAccountUsage, error) {
	return []service.SettlementPoolAccountUsage{}, nil
}

type createGroupSettlementGroupRepo struct {
	group *service.Group
}

func (r *createGroupSettlementGroupRepo) Create(context.Context, *service.Group) error {
	panic("unexpected Create")
}

func (r *createGroupSettlementGroupRepo) GetByID(context.Context, int64) (*service.Group, error) {
	panic("unexpected GetByID")
}

func (r *createGroupSettlementGroupRepo) GetByIDLite(context.Context, int64) (*service.Group, error) {
	group := *r.group
	return &group, nil
}

func (r *createGroupSettlementGroupRepo) Update(context.Context, *service.Group) error {
	panic("unexpected Update")
}

func (r *createGroupSettlementGroupRepo) Delete(context.Context, int64) error {
	panic("unexpected Delete")
}

func (r *createGroupSettlementGroupRepo) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade")
}

func (r *createGroupSettlementGroupRepo) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected List")
}

func (r *createGroupSettlementGroupRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters")
}

func (r *createGroupSettlementGroupRepo) ListActive(context.Context) ([]service.Group, error) {
	panic("unexpected ListActive")
}

func (r *createGroupSettlementGroupRepo) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	panic("unexpected ListActiveByPlatform")
}

func (r *createGroupSettlementGroupRepo) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName")
}

func (r *createGroupSettlementGroupRepo) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount")
}

func (r *createGroupSettlementGroupRepo) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID")
}

func (r *createGroupSettlementGroupRepo) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs")
}

func (r *createGroupSettlementGroupRepo) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup")
}

func (r *createGroupSettlementGroupRepo) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders")
}
