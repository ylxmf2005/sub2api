package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestCalculateSettlementPoolEstimate_ProgressiveTiersAndCap(t *testing.T) {
	cycle := &SettlementPoolCycle{
		ID:        10,
		GroupID:   20,
		Status:    SettlementPoolCycleStatusActive,
		StartedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		TotalCost: 1000,
		BaseRatio: 0.2,
		MarketCap: 0.35,
		Tiers:     DefaultSettlementPoolTiers(),
	}
	participants := []SettlementPoolParticipant{
		{UserID: 1, Email: "a@example.com", Username: "a", Status: StatusActive},
		{UserID: 2, Email: "b@example.com", Username: "b", Status: StatusActive},
	}
	estimate := CalculateSettlementPoolEstimate(cycle, participants, map[int64]float64{
		1: 200,
		2: 1400,
	}, cycle.Tiers)

	require.Equal(t, 2, estimate.ParticipantCount)
	require.InDelta(t, 200, estimate.FixedPool, 1e-9)
	require.InDelta(t, 800, estimate.DynamicPool, 1e-9)
	require.InDelta(t, 200, estimate.Participants[0].WeightedUsage, 1e-9)
	require.InDelta(t, 1148, estimate.Participants[1].WeightedUsage, 1e-9)
	require.InDelta(t, 0.5934718101, estimate.UncappedDynamicRate, 1e-9)
	require.InDelta(t, 0.35, estimate.EffectiveDynamicRate, 1e-9)
	require.InDelta(t, 170, estimate.Participants[0].TotalDue, 1e-9)
	require.InDelta(t, 501.8, estimate.Participants[1].TotalDue, 1e-9)
	require.InDelta(t, 328.2, estimate.OwnerCoveredLoss, 1e-9)
}

func TestCalculateSettlementPoolEstimate_NoWeightedUsage(t *testing.T) {
	cycle := &SettlementPoolCycle{
		ID:        1,
		GroupID:   2,
		Status:    SettlementPoolCycleStatusActive,
		StartedAt: time.Now(),
		TotalCost: 500,
		BaseRatio: 0.2,
		MarketCap: 0.35,
		Tiers:     DefaultSettlementPoolTiers(),
	}
	participants := []SettlementPoolParticipant{
		{UserID: 1, Email: "a@example.com", Status: StatusActive},
		{UserID: 2, Email: "b@example.com", Status: StatusActive},
	}
	estimate := CalculateSettlementPoolEstimate(cycle, participants, map[int64]float64{}, cycle.Tiers)

	require.InDelta(t, 50, estimate.Participants[0].TotalDue, 1e-9)
	require.InDelta(t, 50, estimate.Participants[1].TotalDue, 1e-9)
	require.InDelta(t, 400, estimate.OwnerCoveredLoss, 1e-9)
	require.Zero(t, estimate.EffectiveDynamicRate)
}

func TestWeightedSettlementUsage_AddsOpenEndedTier(t *testing.T) {
	upTo100 := 100.0
	tiers, err := normalizeSettlementPoolTiers([]SettlementPoolTier{
		{UpTo: &upTo100, Weight: 1},
	})
	require.NoError(t, err)
	require.Len(t, tiers, 2)
	require.Nil(t, tiers[1].UpTo)
	require.InDelta(t, 150, WeightedSettlementUsage(150, tiers), 1e-9)
}

func TestSettlementPoolUserSummary_HistoryOnlyOmitsActiveEstimate(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	lockedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		groupIDs: []int64{groupID},
		currentParticipants: map[int64]bool{
			groupID: false,
		},
		userCycles: []SettlementPoolCycle{
			{
				ID:        1,
				GroupID:   groupID,
				Status:    SettlementPoolCycleStatusLocked,
				StartedAt: lockedAt.Add(-24 * time.Hour),
				EndedAt:   &lockedAt,
				Snapshot: &SettlementPoolEstimate{
					GroupID: groupID,
					Participants: []SettlementPoolParticipantEstimate{
						{UserID: userID, TotalDue: 12.5},
					},
				},
			},
		},
		active: &SettlementPoolCycle{
			ID:        2,
			GroupID:   groupID,
			Status:    SettlementPoolCycleStatusActive,
			StartedAt: lockedAt,
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	summaries, err := svc.GetUserSummaries(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Nil(t, summaries[0].ActiveCycle)
	require.Nil(t, summaries[0].Estimate)
	require.Len(t, summaries[0].Cycles, 1)
	require.Zero(t, repo.listParticipantsCalls)
	require.Zero(t, repo.sumUsageCalls)
}

func TestSettlementPoolUserSummary_CurrentParticipantIncludesActiveEstimate(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		groupIDs: []int64{groupID},
		currentParticipants: map[int64]bool{
			groupID: true,
		},
		config: &SettlementPoolConfig{
			GroupID:   groupID,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		active: &SettlementPoolCycle{
			ID:        2,
			GroupID:   groupID,
			Status:    SettlementPoolCycleStatusActive,
			StartedAt: startedAt,
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		participants: []SettlementPoolParticipant{
			{UserID: userID, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{
			userID: 40,
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	summary, err := svc.GetUserSummary(context.Background(), userID, groupID)
	require.NoError(t, err)
	require.NotNil(t, summary.ActiveCycle)
	require.NotNil(t, summary.Estimate)
	require.Len(t, summary.Estimate.Participants, 1)
	require.InDelta(t, 40, summary.Estimate.Participants[0].RawUsage, 1e-9)
	require.Equal(t, 1, repo.listParticipantsCalls)
	require.Equal(t, 1, repo.sumUsageCalls)
}

func TestSettlementPoolUserSummary_DeniesUserWithoutCurrentOrHistory(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	repo := &settlementPoolRepoStub{
		currentParticipants: map[int64]bool{
			groupID: false,
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	_, err := svc.GetUserSummary(context.Background(), userID, groupID)
	require.ErrorIs(t, err, ErrSettlementPoolParticipantMissing)
}

func TestSettlementPoolStartNextCycle_RotatesWithLockedSnapshot(t *testing.T) {
	groupID := int64(11)
	activeStarted := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		config: &SettlementPoolConfig{
			GroupID:   groupID,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		active: &SettlementPoolCycle{
			ID:        2,
			GroupID:   groupID,
			Status:    SettlementPoolCycleStatusActive,
			StartedAt: activeStarted,
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		participants: []SettlementPoolParticipant{
			{UserID: 7, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{7: 40},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	summary, err := svc.StartNextCycle(context.Background(), groupID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 1, repo.rotateCalls)
	require.Equal(t, int64(2), repo.rotatedActiveID)
	require.NotNil(t, repo.rotatedSnapshot)
	require.Equal(t, SettlementPoolCycleStatusLocked, repo.rotatedSnapshot.Status)
	require.NotNil(t, repo.rotatedSnapshot.EndedAt)
	require.NotNil(t, repo.rotatedSnapshot.LockedAt)
	require.Len(t, repo.rotatedSnapshot.Participants, 1)
	require.InDelta(t, 100, repo.rotatedSnapshot.TotalCost, 1e-9)
	require.NotNil(t, repo.rotatedNext)
	require.Equal(t, SettlementPoolCycleStatusActive, repo.rotatedNext.Status)
	require.Zero(t, repo.rotatedNext.TotalCost)
	require.NotNil(t, summary.ActiveCycle)
	require.Equal(t, repo.rotatedNext.ID, summary.ActiveCycle.ID)
}

type settlementPoolRepoStub struct {
	config                *SettlementPoolConfig
	active                *SettlementPoolCycle
	groupIDs              []int64
	currentParticipants   map[int64]bool
	cycles                []SettlementPoolCycle
	userCycles            []SettlementPoolCycle
	participants          []SettlementPoolParticipant
	rawUsage              map[int64]float64
	listParticipantsCalls int
	sumUsageCalls         int
	rotateCalls           int
	rotatedActiveID       int64
	rotatedSnapshot       *SettlementPoolEstimate
	rotatedNext           *SettlementPoolCycle
}

func (s *settlementPoolRepoStub) GetConfig(context.Context, int64) (*SettlementPoolConfig, error) {
	if s.config == nil {
		return nil, ErrSettlementPoolNotFound
	}
	return s.config, nil
}

func (s *settlementPoolRepoStub) UpsertConfig(context.Context, *SettlementPoolConfig) error {
	panic("unexpected UpsertConfig call")
}

func (s *settlementPoolRepoStub) EnsureActiveCycle(_ context.Context, groupID int64, defaults *SettlementPoolConfig) (*SettlementPoolConfig, *SettlementPoolCycle, error) {
	if s.config == nil {
		if defaults == nil {
			defaults = DefaultSettlementPoolConfig(groupID)
		}
		s.config = defaults
	}
	if s.active == nil {
		return s.config, nil, ErrSettlementPoolNotFound
	}
	return s.config, s.active, nil
}

func (s *settlementPoolRepoStub) SetActiveCycleID(context.Context, int64, *int64) error {
	panic("unexpected SetActiveCycleID call")
}

func (s *settlementPoolRepoStub) GetActiveCycle(context.Context, int64) (*SettlementPoolCycle, error) {
	if s.active == nil {
		return nil, ErrSettlementPoolNotFound
	}
	return s.active, nil
}

func (s *settlementPoolRepoStub) CreateCycle(context.Context, *SettlementPoolCycle) error {
	panic("unexpected CreateCycle call")
}

func (s *settlementPoolRepoStub) UpdateActiveCycleConfig(context.Context, int64, SettlementPoolConfigInput) error {
	panic("unexpected UpdateActiveCycleConfig call")
}

func (s *settlementPoolRepoStub) LockCycle(context.Context, int64, time.Time, *SettlementPoolEstimate) error {
	panic("unexpected LockCycle call")
}

func (s *settlementPoolRepoStub) RotateCycle(_ context.Context, groupID, activeCycleID int64, endedAt time.Time, snapshot *SettlementPoolEstimate, next *SettlementPoolCycle) error {
	s.rotateCalls++
	s.rotatedActiveID = activeCycleID
	s.rotatedSnapshot = snapshot
	if next.ID == 0 {
		next.ID = activeCycleID + 1
	}
	if next.GroupID == 0 {
		next.GroupID = groupID
	}
	if next.StartedAt.IsZero() {
		next.StartedAt = endedAt
	}
	s.rotatedNext = next
	s.active = next
	s.cycles = []SettlementPoolCycle{*next}
	return nil
}

func (s *settlementPoolRepoStub) ListCycles(context.Context, int64, int) ([]SettlementPoolCycle, error) {
	return s.cycles, nil
}

func (s *settlementPoolRepoStub) ListCyclesForUser(context.Context, int64, int) ([]SettlementPoolCycle, error) {
	return s.userCycles, nil
}

func (s *settlementPoolRepoStub) ListParticipants(context.Context, int64) ([]SettlementPoolParticipant, error) {
	s.listParticipantsCalls++
	return s.participants, nil
}

func (s *settlementPoolRepoStub) SyncParticipants(context.Context, int64, []int64) error {
	panic("unexpected SyncParticipants call")
}

func (s *settlementPoolRepoStub) IsParticipant(_ context.Context, _ int64, groupID int64) (bool, error) {
	return s.currentParticipants[groupID], nil
}

func (s *settlementPoolRepoStub) ListUserPoolGroupIDs(context.Context, int64) ([]int64, error) {
	return s.groupIDs, nil
}

func (s *settlementPoolRepoStub) SumUsageByUsers(context.Context, int64, []int64, time.Time, *time.Time) (map[int64]float64, error) {
	s.sumUsageCalls++
	return s.rawUsage, nil
}

type settlementGroupRepoStub struct {
	groups map[int64]*Group
}

func (s *settlementGroupRepoStub) Create(context.Context, *Group) error {
	panic("unexpected Create call")
}

func (s *settlementGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	return s.GetByIDLite(context.Background(), id)
}

func (s *settlementGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	group := s.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

func (s *settlementGroupRepoStub) Update(context.Context, *Group) error {
	panic("unexpected Update call")
}

func (s *settlementGroupRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *settlementGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}

func (s *settlementGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *settlementGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *settlementGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	panic("unexpected ListActive call")
}

func (s *settlementGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected ListActiveByPlatform call")
}

func (s *settlementGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName call")
}

func (s *settlementGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount call")
}

func (s *settlementGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID call")
}

func (s *settlementGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs call")
}

func (s *settlementGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup call")
}

func (s *settlementGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}
