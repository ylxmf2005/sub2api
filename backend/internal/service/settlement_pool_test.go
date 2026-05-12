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

func TestCalculateSettlementPoolEstimateWithManualUsage_AddsManualUsageToRawUsage(t *testing.T) {
	cycle := &SettlementPoolCycle{
		ID:        10,
		GroupID:   20,
		Status:    SettlementPoolCycleStatusActive,
		StartedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		TotalCost: 100,
		BaseRatio: 0.2,
		MarketCap: 10,
		Tiers:     DefaultSettlementPoolTiers(),
	}
	participants := []SettlementPoolParticipant{
		{UserID: 1, Email: "a@example.com", Status: StatusActive},
	}
	estimate := CalculateSettlementPoolEstimateWithManualUsage(cycle, participants, map[int64]float64{
		1: 12.5,
	}, map[int64]float64{
		1: 7.5,
	}, cycle.Tiers)

	require.Len(t, estimate.Participants, 1)
	require.InDelta(t, 20, estimate.Participants[0].RawUsage, 1e-9)
	require.InDelta(t, 7.5, estimate.Participants[0].ManualUsage, 1e-9)
	require.InDelta(t, 20, estimate.TotalRawUsage, 1e-9)
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

func TestSettlementPoolCreateManualUsageAdjustment_RequiresCurrentParticipant(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	repo := &settlementPoolRepoStub{
		active: &SettlementPoolCycle{
			ID:        2,
			GroupID:   groupID,
			Status:    SettlementPoolCycleStatusActive,
			StartedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		currentParticipants: map[int64]bool{
			groupID: false,
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	_, err := svc.CreateManualUsageAdjustment(context.Background(), groupID, SettlementPoolManualUsageAdjustmentInput{
		UserID:      userID,
		AccountID:   101,
		UsageAmount: 12.5,
		Reason:      "outside proxy",
		CreatedBy:   1,
	})

	require.ErrorIs(t, err, ErrSettlementPoolParticipantMissing)
	require.Zero(t, repo.createManualUsageCalls)
}

func TestSettlementPoolCreateManualUsageAdjustment_CreatesAdjustmentAndRefreshesSummary(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
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
			StartedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		currentParticipants: map[int64]bool{
			groupID: true,
		},
		participants: []SettlementPoolParticipant{
			{UserID: userID, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{
			userID: 40,
		},
		manualUsage: map[int64]float64{
			userID: 5,
		},
		accountUsage: []SettlementPoolAccountUsage{
			{AccountID: 101, Name: "open account", TotalUsage: 40, WeeklyTotalUsage: 7.5},
		},
		manualAccountUsage: map[int64]float64{
			101: 5,
		},
		manualAdjustments: []SettlementPoolManualUsageAdjustment{
			{ID: 1, GroupID: groupID, CycleID: 2, UserID: userID, AccountID: 101, AccountName: "open account", UsageAmount: 5, Reason: "outside proxy", CreatedBy: 1},
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	summary, err := svc.CreateManualUsageAdjustment(context.Background(), groupID, SettlementPoolManualUsageAdjustmentInput{
		UserID:      userID,
		AccountID:   101,
		UsageAmount: 5,
		Reason:      " outside proxy ",
		CreatedBy:   1,
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.createManualUsageCalls)
	require.Equal(t, "outside proxy", repo.createdManualAdjustment.Reason)
	require.Equal(t, int64(101), repo.createdManualAdjustment.AccountID)
	require.NotNil(t, summary.Estimate)
	require.InDelta(t, 45, summary.Estimate.Participants[0].RawUsage, 1e-9)
	require.InDelta(t, 5, summary.Estimate.Participants[0].ManualUsage, 1e-9)
	require.InDelta(t, 45, summary.Estimate.AccountUsage[0].TotalUsage, 1e-9)
	require.InDelta(t, 5, summary.Estimate.AccountUsage[0].ManualUsage, 1e-9)
	require.Len(t, summary.Estimate.ManualAdjustments, 1)
}

func TestSettlementPoolCreateManualUsageAdjustment_RejectsNegativeTotals(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	accountID := int64(101)
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
			StartedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		currentParticipants: map[int64]bool{
			groupID: true,
		},
		participants: []SettlementPoolParticipant{
			{UserID: userID, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{
			userID: 4,
		},
		accountUsage: []SettlementPoolAccountUsage{
			{AccountID: accountID, Name: "open account", TotalUsage: 10},
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	_, err := svc.CreateManualUsageAdjustment(context.Background(), groupID, SettlementPoolManualUsageAdjustmentInput{
		UserID:      userID,
		AccountID:   accountID,
		UsageAmount: -5,
		Reason:      "correction",
		CreatedBy:   1,
	})

	require.ErrorIs(t, err, ErrSettlementPoolInvalidAdjustment)
	require.Zero(t, repo.createManualUsageCalls)
}

func TestSettlementPoolCreateManualUsageAdjustment_RejectsHistoricalAccountOutsideEnabledPool(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	historicalAccountID := int64(101)
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
			StartedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		currentParticipants: map[int64]bool{
			groupID: true,
		},
		participants: []SettlementPoolParticipant{
			{UserID: userID, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{
			userID: 40,
		},
		enabledAccountUsage: []SettlementPoolAccountUsage{},
		accountUsage: []SettlementPoolAccountUsage{
			{AccountID: historicalAccountID, Name: "deleted historical account", Status: StatusError, Schedulable: false, TotalUsage: 40},
		},
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	_, err := svc.CreateManualUsageAdjustment(context.Background(), groupID, SettlementPoolManualUsageAdjustmentInput{
		UserID:      userID,
		AccountID:   historicalAccountID,
		UsageAmount: 5,
		Reason:      "outside proxy",
		CreatedBy:   1,
	})

	require.ErrorIs(t, err, ErrSettlementPoolInvalidAdjustment)
	require.Zero(t, repo.createManualUsageCalls)
}

func TestSettlementPoolUserSummary_CandidateOnlyIncludesActiveEstimate(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	lockedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		candidateGroupIDs: []int64{groupID},
		candidates: map[int64]bool{
			groupID: true,
		},
		currentParticipants: map[int64]bool{
			groupID: false,
		},
		config: &SettlementPoolConfig{
			GroupID:   groupID,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
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
		participants: []SettlementPoolParticipant{
			{UserID: 8, Email: "other@example.com", Status: StatusActive},
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
	require.True(t, summaries[0].IsCandidate)
	require.False(t, summaries[0].IsCurrentParticipant)
	require.True(t, summaries[0].CanJoinActiveCycle)
	require.NotNil(t, summaries[0].ActiveCycle)
	require.NotNil(t, summaries[0].Estimate)
	require.Len(t, summaries[0].Estimate.Participants, 1)
	require.Equal(t, int64(8), summaries[0].Estimate.Participants[0].UserID)
	require.Len(t, summaries[0].Cycles, 1)
	require.Equal(t, 1, repo.listParticipantsCalls)
	require.Equal(t, 1, repo.sumUsageCalls)
}

func TestSettlementPoolUserSummary_CurrentParticipantIncludesActiveEstimate(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		candidateGroupIDs: []int64{groupID},
		candidates: map[int64]bool{
			groupID: true,
		},
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
	require.True(t, summary.IsCandidate)
	require.True(t, summary.IsCurrentParticipant)
	require.False(t, summary.CanJoinActiveCycle)
	require.Len(t, summary.Estimate.Participants, 1)
	require.InDelta(t, 40, summary.Estimate.Participants[0].RawUsage, 1e-9)
	require.Equal(t, 1, repo.listParticipantsCalls)
	require.Equal(t, 1, repo.sumUsageCalls)
}

func TestSettlementPoolUserSummaries_CurrentParticipantReturnsEmptyCyclesArray(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		candidateGroupIDs: []int64{groupID},
		candidates: map[int64]bool{
			groupID: true,
		},
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
	}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, nil)

	summaries, err := svc.GetUserSummaries(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].Cycles)
	require.Empty(t, summaries[0].Cycles)
}

func TestSettlementPoolUserSummary_DeniesUserWithoutCurrentOrHistory(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	repo := &settlementPoolRepoStub{
		candidates: map[int64]bool{
			groupID: false,
		},
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
	require.ErrorIs(t, err, ErrSettlementPoolCandidateMissing)
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
		accountUsage: []SettlementPoolAccountUsage{
			{AccountID: 101, Name: "open account", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Requests: 2, TotalUsage: 40},
		},
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
	require.Equal(t, activeStarted, repo.rotatedSnapshot.StartedAt)
	require.NotNil(t, repo.rotatedSnapshot.EndedAt)
	require.NotNil(t, repo.rotatedSnapshot.LockedAt)
	require.NotNil(t, repo.sumUsageEndedAt)
	require.Equal(t, *repo.rotatedSnapshot.EndedAt, *repo.sumUsageEndedAt)
	require.Len(t, repo.rotatedSnapshot.Participants, 1)
	require.Len(t, repo.rotatedSnapshot.AccountUsage, 1)
	require.Equal(t, int64(101), repo.rotatedSnapshot.AccountUsage[0].AccountID)
	require.NotNil(t, repo.listAccountUsageEndedAt)
	require.Equal(t, *repo.rotatedSnapshot.EndedAt, *repo.listAccountUsageEndedAt)
	require.InDelta(t, 100, repo.rotatedSnapshot.TotalCost, 1e-9)
	require.NotNil(t, repo.rotatedNext)
	require.Equal(t, SettlementPoolCycleStatusActive, repo.rotatedNext.Status)
	require.False(t, repo.rotatedNext.StartedAt.IsZero())
	require.Equal(t, *repo.rotatedSnapshot.EndedAt, repo.rotatedNext.StartedAt)
	require.Zero(t, repo.rotatedNext.TotalCost)
	require.NotNil(t, summary.ActiveCycle)
	require.Equal(t, repo.rotatedNext.ID, summary.ActiveCycle.ID)
	require.Empty(t, summary.Estimate.Participants)
}

func TestSettlementPoolJoinCurrentCycle_AddsCandidateToActiveCycle(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	repo := &settlementPoolRepoStub{
		candidates: map[int64]bool{
			groupID: true,
		},
		currentParticipants: map[int64]bool{
			groupID: false,
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
	}
	cache := &settlementPoolAuthCacheStub{}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, cache)

	summary, err := svc.JoinCurrentCycle(context.Background(), userID, groupID)
	require.NoError(t, err)
	require.Equal(t, 1, repo.joinCurrentCalls)
	require.Equal(t, groupID, repo.joinedGroupID)
	require.Equal(t, userID, repo.joinedUserID)
	require.True(t, summary.IsCurrentParticipant)
	require.False(t, summary.CanJoinActiveCycle)
	require.Equal(t, []int64{groupID}, cache.invalidatedGroupIDs)
}

func TestSettlementPoolRemoveCurrentParticipant_DoesNotCheckUsage(t *testing.T) {
	userID := int64(7)
	groupID := int64(11)
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
			StartedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			TotalCost: 100,
			BaseRatio: 0.2,
			MarketCap: 0.35,
			Tiers:     DefaultSettlementPoolTiers(),
		},
		currentParticipants: map[int64]bool{
			groupID: true,
		},
		participants: []SettlementPoolParticipant{
			{UserID: userID, Email: "user@example.com", Status: StatusActive},
		},
		rawUsage: map[int64]float64{userID: 40},
	}
	cache := &settlementPoolAuthCacheStub{}
	svc := NewSettlementPoolService(repo, &settlementGroupRepoStub{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "pool", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
		},
	}, cache)

	summary, err := svc.RemoveCurrentParticipant(context.Background(), groupID, userID)
	require.NoError(t, err)
	require.Equal(t, 1, repo.removeCurrentCalls)
	require.Equal(t, groupID, repo.removedGroupID)
	require.Equal(t, userID, repo.removedUserID)
	require.Empty(t, summary.Estimate.Participants)
	require.Equal(t, []int64{groupID}, cache.invalidatedGroupIDs)
}

type settlementPoolRepoStub struct {
	config                       *SettlementPoolConfig
	active                       *SettlementPoolCycle
	candidateGroupIDs            []int64
	currentGroupIDs              []int64
	candidates                   map[int64]bool
	currentParticipants          map[int64]bool
	cycles                       []SettlementPoolCycle
	userCycles                   []SettlementPoolCycle
	participants                 []SettlementPoolParticipant
	candidateRows                []SettlementPoolParticipant
	rawUsage                     map[int64]float64
	manualUsage                  map[int64]float64
	manualAccountUsage           map[int64]float64
	manualAdjustments            []SettlementPoolManualUsageAdjustment
	accountUsage                 []SettlementPoolAccountUsage
	enabledAccountUsage          []SettlementPoolAccountUsage
	sumUsageEndedAt              *time.Time
	listAccountUsageEndedAt      *time.Time
	listParticipantsCalls        int
	sumUsageCalls                int
	listAccountUsageCalls        int
	listEnabledAccountUsageCalls int
	rotateCalls                  int
	joinCurrentCalls             int
	removeCurrentCalls           int
	forceJoinCalls               int
	createManualUsageCalls       int
	rotatedActiveID              int64
	rotatedSnapshot              *SettlementPoolEstimate
	rotatedNext                  *SettlementPoolCycle
	joinedGroupID                int64
	joinedUserID                 int64
	removedGroupID               int64
	removedUserID                int64
	createdManualAdjustment      *SettlementPoolManualUsageAdjustment
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
	s.participants = nil
	return nil
}

func (s *settlementPoolRepoStub) ListCycles(context.Context, int64, int) ([]SettlementPoolCycle, error) {
	return s.cycles, nil
}

func (s *settlementPoolRepoStub) ListCyclesForUser(context.Context, int64, int) ([]SettlementPoolCycle, error) {
	return s.userCycles, nil
}

func (s *settlementPoolRepoStub) ListCandidates(context.Context, int64) ([]SettlementPoolParticipant, error) {
	return s.candidateRows, nil
}

func (s *settlementPoolRepoStub) SyncCandidates(context.Context, int64, []int64) error {
	panic("unexpected SyncCandidates call")
}

func (s *settlementPoolRepoStub) IsCandidate(_ context.Context, _ int64, groupID int64) (bool, error) {
	if s.candidates != nil {
		return s.candidates[groupID], nil
	}
	for _, candidateGroupID := range s.candidateGroupIDs {
		if candidateGroupID == groupID {
			return true, nil
		}
	}
	return false, nil
}

func (s *settlementPoolRepoStub) ListCandidateGroupIDs(context.Context, int64) ([]int64, error) {
	return s.candidateGroupIDs, nil
}

func (s *settlementPoolRepoStub) ListCycleParticipants(context.Context, int64) ([]SettlementPoolParticipant, error) {
	s.listParticipantsCalls++
	return s.participants, nil
}

func (s *settlementPoolRepoStub) JoinCurrentCycle(_ context.Context, groupID, userID int64) error {
	s.joinCurrentCalls++
	s.joinedGroupID = groupID
	s.joinedUserID = userID
	if s.currentParticipants == nil {
		s.currentParticipants = make(map[int64]bool)
	}
	s.currentParticipants[groupID] = true
	s.participants = append(s.participants, SettlementPoolParticipant{UserID: userID, Email: "user@example.com", Status: StatusActive})
	return nil
}

func (s *settlementPoolRepoStub) ForceJoinCurrentCycle(context.Context, int64, []int64) error {
	s.forceJoinCalls++
	return nil
}

func (s *settlementPoolRepoStub) RemoveCurrentParticipant(_ context.Context, groupID, userID int64) error {
	s.removeCurrentCalls++
	s.removedGroupID = groupID
	s.removedUserID = userID
	if s.currentParticipants != nil {
		s.currentParticipants[groupID] = false
	}
	filtered := s.participants[:0]
	for _, participant := range s.participants {
		if participant.UserID != userID {
			filtered = append(filtered, participant)
		}
	}
	s.participants = filtered
	return nil
}

func (s *settlementPoolRepoStub) IsCurrentParticipant(_ context.Context, _ int64, groupID int64) (bool, error) {
	return s.currentParticipants[groupID], nil
}

func (s *settlementPoolRepoStub) ListCurrentParticipantGroupIDs(context.Context, int64) ([]int64, error) {
	return s.currentGroupIDs, nil
}

func (s *settlementPoolRepoStub) SumUsageByUsers(_ context.Context, _ int64, _ []int64, _ time.Time, endedAt *time.Time) (map[int64]float64, error) {
	s.sumUsageCalls++
	if endedAt != nil {
		capturedEndedAt := *endedAt
		s.sumUsageEndedAt = &capturedEndedAt
	}
	return s.rawUsage, nil
}

func (s *settlementPoolRepoStub) SumManualUsageByUsers(context.Context, int64, []int64) (map[int64]float64, error) {
	return s.manualUsage, nil
}

func (s *settlementPoolRepoStub) SumManualUsageByAccounts(context.Context, int64, []int64) (map[int64]float64, error) {
	return s.manualAccountUsage, nil
}

func (s *settlementPoolRepoStub) CreateManualUsageAdjustment(_ context.Context, adjustment *SettlementPoolManualUsageAdjustment) error {
	s.createManualUsageCalls++
	copied := *adjustment
	s.createdManualAdjustment = &copied
	return nil
}

func (s *settlementPoolRepoStub) ListManualUsageAdjustments(context.Context, int64) ([]SettlementPoolManualUsageAdjustment, error) {
	return s.manualAdjustments, nil
}

func (s *settlementPoolRepoStub) ListEnabledAccountUsage(_ context.Context, _ int64, _ time.Time, endedAt *time.Time) ([]SettlementPoolAccountUsage, error) {
	s.listEnabledAccountUsageCalls++
	if endedAt != nil {
		capturedEndedAt := *endedAt
		s.listAccountUsageEndedAt = &capturedEndedAt
	}
	if s.enabledAccountUsage != nil {
		return s.enabledAccountUsage, nil
	}
	return s.accountUsage, nil
}

func (s *settlementPoolRepoStub) ListSettlementAccountUsage(_ context.Context, _, _ int64, _ time.Time, endedAt *time.Time) ([]SettlementPoolAccountUsage, error) {
	s.listAccountUsageCalls++
	if endedAt != nil {
		capturedEndedAt := *endedAt
		s.listAccountUsageEndedAt = &capturedEndedAt
	}
	return s.accountUsage, nil
}

type settlementPoolAuthCacheStub struct {
	invalidatedGroupIDs []int64
}

func (s *settlementPoolAuthCacheStub) InvalidateAuthCacheByKey(context.Context, string) {}

func (s *settlementPoolAuthCacheStub) InvalidateAuthCacheByUserID(context.Context, int64) {}

func (s *settlementPoolAuthCacheStub) InvalidateAuthCacheByGroupID(_ context.Context, groupID int64) {
	s.invalidatedGroupIDs = append(s.invalidatedGroupIDs, groupID)
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
