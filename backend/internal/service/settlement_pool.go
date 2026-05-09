package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettlementPoolCycleStatusActive = "active"
	SettlementPoolCycleStatusLocked = "locked"
)

var (
	ErrSettlementPoolNotFound           = infraerrors.NotFound("SETTLEMENT_POOL_NOT_FOUND", "settlement pool not found")
	ErrSettlementPoolInvalidConfig      = infraerrors.BadRequest("SETTLEMENT_POOL_INVALID_CONFIG", "invalid settlement pool config")
	ErrSettlementPoolForbidden          = infraerrors.Forbidden("SETTLEMENT_POOL_FORBIDDEN", "not allowed to access settlement pool")
	ErrSettlementPoolCandidateMissing   = infraerrors.Forbidden("SETTLEMENT_POOL_CANDIDATE_MISSING", "user is not a settlement pool candidate")
	ErrSettlementPoolParticipantMissing = infraerrors.Forbidden("SETTLEMENT_POOL_PARTICIPANT_MISSING", "user is not a current settlement pool participant")
)

type SettlementPoolTier struct {
	UpTo   *float64 `json:"up_to"`
	Weight float64  `json:"weight"`
}

type SettlementPoolConfig struct {
	GroupID       int64                `json:"group_id"`
	BaseRatio     float64              `json:"base_ratio"`
	MarketCap     float64              `json:"market_cap"`
	Tiers         []SettlementPoolTier `json:"tiers"`
	ActiveCycleID *int64               `json:"active_cycle_id,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type SettlementPoolCycle struct {
	ID        int64                   `json:"id"`
	GroupID   int64                   `json:"group_id"`
	Status    string                  `json:"status"`
	StartedAt time.Time               `json:"started_at"`
	EndedAt   *time.Time              `json:"ended_at,omitempty"`
	TotalCost float64                 `json:"total_cost"`
	BaseRatio float64                 `json:"base_ratio"`
	MarketCap float64                 `json:"market_cap"`
	Tiers     []SettlementPoolTier    `json:"tiers"`
	Snapshot  *SettlementPoolEstimate `json:"snapshot,omitempty"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type SettlementPoolParticipant struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SettlementPoolParticipantEstimate struct {
	UserID        int64   `json:"user_id"`
	Email         string  `json:"email"`
	Username      string  `json:"username"`
	Status        string  `json:"status"`
	RawUsage      float64 `json:"raw_usage"`
	WeightedUsage float64 `json:"weighted_usage"`
	CurrentTier   int     `json:"current_tier"`
	FixedShare    float64 `json:"fixed_share"`
	DynamicCharge float64 `json:"dynamic_charge"`
	TotalDue      float64 `json:"total_due"`
}

type SettlementPoolAccountUsage struct {
	AccountID           int64   `json:"account_id"`
	Name                string  `json:"name"`
	Platform            string  `json:"platform"`
	Type                string  `json:"type"`
	Status              string  `json:"status"`
	Schedulable         bool    `json:"schedulable"`
	Requests            int64   `json:"requests"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	TotalUsage          float64 `json:"total_usage"`
}

type SettlementPoolGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Platform         string  `json:"platform"`
	SubscriptionType string  `json:"subscription_type"`
	Status           string  `json:"status"`
	RateMultiplier   float64 `json:"rate_multiplier"`
}

type SettlementPoolEstimate struct {
	GroupID              int64                               `json:"group_id"`
	CycleID              int64                               `json:"cycle_id"`
	Status               string                              `json:"status"`
	StartedAt            time.Time                           `json:"started_at"`
	EndedAt              *time.Time                          `json:"ended_at,omitempty"`
	LockedAt             *time.Time                          `json:"locked_at,omitempty"`
	TotalCost            float64                             `json:"total_cost"`
	BaseRatio            float64                             `json:"base_ratio"`
	MarketCap            float64                             `json:"market_cap"`
	Tiers                []SettlementPoolTier                `json:"tiers"`
	ParticipantCount     int                                 `json:"participant_count"`
	FixedPool            float64                             `json:"fixed_pool"`
	DynamicPool          float64                             `json:"dynamic_pool"`
	TotalRawUsage        float64                             `json:"total_raw_usage"`
	TotalWeightedUsage   float64                             `json:"total_weighted_usage"`
	UncappedDynamicRate  float64                             `json:"uncapped_dynamic_rate"`
	EffectiveDynamicRate float64                             `json:"effective_dynamic_rate"`
	OwnerCoveredLoss     float64                             `json:"owner_covered_loss"`
	Participants         []SettlementPoolParticipantEstimate `json:"participants"`
	AccountUsage         []SettlementPoolAccountUsage        `json:"account_usage"`
}

type SettlementPoolSummary struct {
	Group                *SettlementPoolGroup        `json:"group"`
	Config               *SettlementPoolConfig       `json:"config,omitempty"`
	ActiveCycle          *SettlementPoolCycle        `json:"active_cycle,omitempty"`
	Estimate             *SettlementPoolEstimate     `json:"estimate,omitempty"`
	Cycles               []SettlementPoolCycle       `json:"cycles"`
	Candidates           []SettlementPoolParticipant `json:"candidates,omitempty"`
	IsCandidate          bool                        `json:"is_candidate,omitempty"`
	IsCurrentParticipant bool                        `json:"is_current_participant,omitempty"`
	CanJoinActiveCycle   bool                        `json:"can_join_active_cycle,omitempty"`
}

type SettlementPoolConfigInput struct {
	TotalCost float64              `json:"total_cost"`
	BaseRatio float64              `json:"base_ratio"`
	MarketCap float64              `json:"market_cap"`
	Tiers     []SettlementPoolTier `json:"tiers"`
}

type SettlementPoolRepository interface {
	GetConfig(ctx context.Context, groupID int64) (*SettlementPoolConfig, error)
	EnsureActiveCycle(ctx context.Context, groupID int64, defaults *SettlementPoolConfig) (*SettlementPoolConfig, *SettlementPoolCycle, error)
	UpsertConfig(ctx context.Context, config *SettlementPoolConfig) error
	SetActiveCycleID(ctx context.Context, groupID int64, cycleID *int64) error
	GetActiveCycle(ctx context.Context, groupID int64) (*SettlementPoolCycle, error)
	CreateCycle(ctx context.Context, cycle *SettlementPoolCycle) error
	UpdateActiveCycleConfig(ctx context.Context, groupID int64, input SettlementPoolConfigInput) error
	LockCycle(ctx context.Context, cycleID int64, endedAt time.Time, snapshot *SettlementPoolEstimate) error
	RotateCycle(ctx context.Context, groupID, activeCycleID int64, endedAt time.Time, snapshot *SettlementPoolEstimate, next *SettlementPoolCycle) error
	ListCycles(ctx context.Context, groupID int64, limit int) ([]SettlementPoolCycle, error)
	ListCyclesForUser(ctx context.Context, userID int64, limit int) ([]SettlementPoolCycle, error)
	ListCandidates(ctx context.Context, groupID int64) ([]SettlementPoolParticipant, error)
	SyncCandidates(ctx context.Context, groupID int64, userIDs []int64) error
	IsCandidate(ctx context.Context, userID, groupID int64) (bool, error)
	ListCandidateGroupIDs(ctx context.Context, userID int64) ([]int64, error)
	ListCycleParticipants(ctx context.Context, cycleID int64) ([]SettlementPoolParticipant, error)
	JoinCurrentCycle(ctx context.Context, groupID, userID int64) error
	ForceJoinCurrentCycle(ctx context.Context, groupID int64, userIDs []int64) error
	RemoveCurrentParticipant(ctx context.Context, groupID, userID int64) error
	IsCurrentParticipant(ctx context.Context, userID, groupID int64) (bool, error)
	ListCurrentParticipantGroupIDs(ctx context.Context, userID int64) ([]int64, error)
	SumUsageByUsers(ctx context.Context, groupID int64, userIDs []int64, startedAt time.Time, endedAt *time.Time) (map[int64]float64, error)
	ListEnabledAccountUsage(ctx context.Context, groupID int64, startedAt time.Time, endedAt *time.Time) ([]SettlementPoolAccountUsage, error)
}

type SettlementPoolService struct {
	repo                 SettlementPoolRepository
	groupRepo            GroupRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewSettlementPoolService(repo SettlementPoolRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *SettlementPoolService {
	return &SettlementPoolService{repo: repo, groupRepo: groupRepo, authCacheInvalidator: authCacheInvalidator}
}

func DefaultSettlementPoolTiers() []SettlementPoolTier {
	upTo300 := 300.0
	upTo1200 := 1200.0
	return []SettlementPoolTier{
		{UpTo: &upTo300, Weight: 1},
		{UpTo: &upTo1200, Weight: 0.8},
		{UpTo: nil, Weight: 0.64},
	}
}

func DefaultSettlementPoolConfig(groupID int64) *SettlementPoolConfig {
	return &SettlementPoolConfig{
		GroupID:   groupID,
		BaseRatio: 0.2,
		MarketCap: 0.35,
		Tiers:     DefaultSettlementPoolTiers(),
	}
}

func (s *SettlementPoolService) IsCurrentParticipant(ctx context.Context, userID, groupID int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrSettlementPoolNotFound
	}
	return s.repo.IsCurrentParticipant(ctx, userID, groupID)
}

func (s *SettlementPoolService) ListCurrentParticipantGroupIDs(ctx context.Context, userID int64) ([]int64, error) {
	if s == nil || s.repo == nil {
		return nil, ErrSettlementPoolNotFound
	}
	return s.repo.ListCurrentParticipantGroupIDs(ctx, userID)
}

func (s *SettlementPoolService) GetAdminSummary(ctx context.Context, groupID int64) (*SettlementPoolSummary, error) {
	group, err := s.requireSettlementPoolGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	config, active, err := s.ensureActiveCycle(ctx, groupID)
	if err != nil {
		return nil, err
	}
	estimate, err := s.CalculateEstimate(ctx, active)
	if err != nil {
		return nil, err
	}
	cycles, err := s.repo.ListCycles(ctx, groupID, 24)
	if err != nil {
		return nil, fmt.Errorf("list settlement cycles: %w", err)
	}
	candidates, err := s.repo.ListCandidates(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list settlement candidates: %w", err)
	}
	return &SettlementPoolSummary{
		Group:       settlementPoolGroupFromGroup(group),
		Config:      config,
		ActiveCycle: active,
		Estimate:    estimate,
		Cycles:      nonNilSettlementCycles(cycles),
		Candidates:  nonNilSettlementParticipants(candidates),
	}, nil
}

func (s *SettlementPoolService) UpdateConfig(ctx context.Context, groupID int64, input SettlementPoolConfigInput) (*SettlementPoolSummary, error) {
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	normalized, err := normalizeSettlementPoolConfigInput(input)
	if err != nil {
		return nil, err
	}
	config := &SettlementPoolConfig{
		GroupID:   groupID,
		BaseRatio: normalized.BaseRatio,
		MarketCap: normalized.MarketCap,
		Tiers:     normalized.Tiers,
	}
	if err := s.repo.UpsertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("upsert settlement config: %w", err)
	}
	_, active, err := s.ensureActiveCycle(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		if err := s.repo.UpdateActiveCycleConfig(ctx, groupID, normalized); err != nil {
			return nil, fmt.Errorf("update active settlement cycle: %w", err)
		}
	}
	return s.GetAdminSummary(ctx, groupID)
}

func (s *SettlementPoolService) SyncCandidates(ctx context.Context, groupID int64, userIDs []int64) (*SettlementPoolSummary, error) {
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	userIDs = uniquePositiveInt64s(userIDs)
	if err := s.repo.SyncCandidates(ctx, groupID, userIDs); err != nil {
		return nil, fmt.Errorf("sync settlement candidates: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return s.GetAdminSummary(ctx, groupID)
}

func (s *SettlementPoolService) JoinCurrentCycle(ctx context.Context, userID, groupID int64) (*SettlementPoolSummary, error) {
	if userID <= 0 {
		return nil, ErrSettlementPoolForbidden
	}
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	isCandidate, err := s.repo.IsCandidate(ctx, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("check settlement candidate: %w", err)
	}
	if !isCandidate {
		return nil, ErrSettlementPoolCandidateMissing
	}
	if _, _, err := s.ensureActiveCycle(ctx, groupID); err != nil {
		return nil, err
	}
	if err := s.repo.JoinCurrentCycle(ctx, groupID, userID); err != nil {
		return nil, fmt.Errorf("join settlement current cycle: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return s.GetUserSummary(ctx, userID, groupID)
}

func (s *SettlementPoolService) ForceJoinCurrentCycle(ctx context.Context, groupID int64, userIDs []int64) (*SettlementPoolSummary, error) {
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	userIDs = uniquePositiveInt64s(userIDs)
	if len(userIDs) > 0 {
		if _, _, err := s.ensureActiveCycle(ctx, groupID); err != nil {
			return nil, err
		}
		if err := s.repo.ForceJoinCurrentCycle(ctx, groupID, userIDs); err != nil {
			return nil, fmt.Errorf("force join settlement current cycle: %w", err)
		}
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
		}
	}
	return s.GetAdminSummary(ctx, groupID)
}

func (s *SettlementPoolService) RemoveCurrentParticipant(ctx context.Context, groupID, userID int64) (*SettlementPoolSummary, error) {
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrSettlementPoolParticipantMissing
	}
	if err := s.repo.RemoveCurrentParticipant(ctx, groupID, userID); err != nil {
		return nil, fmt.Errorf("remove settlement current participant: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return s.GetAdminSummary(ctx, groupID)
}

func (s *SettlementPoolService) StartNextCycle(ctx context.Context, groupID int64) (*SettlementPoolSummary, error) {
	if _, err := s.requireSettlementPoolGroup(ctx, groupID); err != nil {
		return nil, err
	}
	config, active, err := s.ensureActiveCycle(ctx, groupID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if active == nil || active.Status != SettlementPoolCycleStatusActive {
		return nil, ErrSettlementPoolNotFound
	}
	lockingCycle := *active
	lockingCycle.EndedAt = &now
	estimate, err := s.CalculateEstimate(ctx, &lockingCycle)
	if err != nil {
		return nil, err
	}
	estimate.Status = SettlementPoolCycleStatusLocked
	estimate.EndedAt = &now
	estimate.LockedAt = &now
	next := &SettlementPoolCycle{
		GroupID:   groupID,
		Status:    SettlementPoolCycleStatusActive,
		StartedAt: now,
		TotalCost: 0,
		BaseRatio: config.BaseRatio,
		MarketCap: config.MarketCap,
		Tiers:     cloneSettlementPoolTiers(config.Tiers),
	}
	if err := s.repo.RotateCycle(ctx, groupID, active.ID, now, estimate, next); err != nil {
		return nil, fmt.Errorf("rotate settlement cycle: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return s.GetAdminSummary(ctx, groupID)
}

func (s *SettlementPoolService) GetUserSummaries(ctx context.Context, userID int64) ([]SettlementPoolSummary, error) {
	if userID <= 0 {
		return nil, ErrSettlementPoolForbidden
	}
	groupIDs, err := s.repo.ListCandidateGroupIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user settlement pools: %w", err)
	}
	cycles, err := s.repo.ListCyclesForUser(ctx, userID, 24)
	if err != nil {
		return nil, fmt.Errorf("list user settlement cycles: %w", err)
	}
	cyclesByGroup := settlementCyclesByGroup(cycles)

	out := make([]SettlementPoolSummary, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := s.groupRepo.GetByIDLite(ctx, groupID)
		if err != nil {
			continue
		}
		if !group.IsSettlementPoolType() {
			continue
		}

		isCurrentParticipant, err := s.repo.IsCurrentParticipant(ctx, userID, groupID)
		if err != nil {
			return nil, fmt.Errorf("check settlement participant: %w", err)
		}

		cycles := cyclesByGroup[groupID]
		summary := SettlementPoolSummary{
			Group:                settlementPoolGroupFromGroup(group),
			Cycles:               nonNilSettlementCycles(cycles),
			IsCandidate:          true,
			IsCurrentParticipant: isCurrentParticipant,
			CanJoinActiveCycle:   !isCurrentParticipant,
		}
		config, active, err := s.ensureActiveCycle(ctx, groupID)
		if err != nil {
			return nil, err
		}
		summary.Config = config
		summary.ActiveCycle = active
		if active != nil {
			estimate, err := s.CalculateEstimate(ctx, active)
			if err != nil {
				return nil, err
			}
			summary.Estimate = estimate
		}
		out = append(out, summary)
	}
	return out, nil
}

func (s *SettlementPoolService) GetUserSummary(ctx context.Context, userID, groupID int64) (*SettlementPoolSummary, error) {
	if userID <= 0 {
		return nil, ErrSettlementPoolForbidden
	}
	group, err := s.groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !group.IsSettlementPoolType() {
		return nil, ErrSettlementPoolNotFound
	}

	cycles, err := s.repo.ListCyclesForUser(ctx, userID, 24)
	if err != nil {
		return nil, fmt.Errorf("list user settlement cycles: %w", err)
	}
	filteredCycles := make([]SettlementPoolCycle, 0, len(cycles))
	for _, cycle := range cycles {
		if cycle.GroupID == groupID {
			filteredCycles = append(filteredCycles, cycle)
		}
	}

	isCandidate, err := s.repo.IsCandidate(ctx, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("check settlement candidate: %w", err)
	}
	if !isCandidate {
		return nil, ErrSettlementPoolCandidateMissing
	}

	isCurrentParticipant, err := s.repo.IsCurrentParticipant(ctx, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("check settlement participant: %w", err)
	}

	summary := &SettlementPoolSummary{
		Group:                settlementPoolGroupFromGroup(group),
		Cycles:               filteredCycles,
		IsCandidate:          true,
		IsCurrentParticipant: isCurrentParticipant,
		CanJoinActiveCycle:   !isCurrentParticipant,
	}
	config, active, err := s.ensureActiveCycle(ctx, groupID)
	if err != nil {
		return nil, err
	}
	summary.Config = config
	summary.ActiveCycle = active
	if active != nil {
		estimate, err := s.CalculateEstimate(ctx, active)
		if err != nil {
			return nil, err
		}
		summary.Estimate = estimate
	}
	return summary, nil
}

func (s *SettlementPoolService) CalculateEstimate(ctx context.Context, cycle *SettlementPoolCycle) (*SettlementPoolEstimate, error) {
	if cycle == nil {
		return nil, ErrSettlementPoolNotFound
	}
	tiers, err := normalizeSettlementPoolTiers(cycle.Tiers)
	if err != nil {
		return nil, err
	}
	participants, err := s.repo.ListCycleParticipants(ctx, cycle.ID)
	if err != nil {
		return nil, fmt.Errorf("list settlement participants: %w", err)
	}
	userIDs := make([]int64, 0, len(participants))
	for _, participant := range participants {
		userIDs = append(userIDs, participant.UserID)
	}
	rawByUser, err := s.repo.SumUsageByUsers(ctx, cycle.GroupID, userIDs, cycle.StartedAt, cycle.EndedAt)
	if err != nil {
		return nil, fmt.Errorf("sum settlement usage: %w", err)
	}
	accountUsage, err := s.repo.ListEnabledAccountUsage(ctx, cycle.GroupID, cycle.StartedAt, cycle.EndedAt)
	if err != nil {
		return nil, fmt.Errorf("list settlement account usage: %w", err)
	}
	estimate := CalculateSettlementPoolEstimate(cycle, participants, rawByUser, tiers)
	estimate.AccountUsage = nonNilSettlementAccountUsage(accountUsage)
	return estimate, nil
}

func CalculateSettlementPoolEstimate(cycle *SettlementPoolCycle, participants []SettlementPoolParticipant, rawByUser map[int64]float64, tiers []SettlementPoolTier) *SettlementPoolEstimate {
	if cycle == nil {
		return nil
	}
	baseRatio := clamp(cycle.BaseRatio, 0, 1)
	totalCost := math.Max(cycle.TotalCost, 0)
	fixedPool := totalCost * baseRatio
	dynamicPool := totalCost - fixedPool
	if dynamicPool < 0 {
		dynamicPool = 0
	}

	rows := make([]SettlementPoolParticipantEstimate, 0, len(participants))
	totalRawUsage := 0.0
	totalWeightedUsage := 0.0
	for _, participant := range participants {
		raw := math.Max(rawByUser[participant.UserID], 0)
		weighted := WeightedSettlementUsage(raw, tiers)
		totalRawUsage += raw
		totalWeightedUsage += weighted
		rows = append(rows, SettlementPoolParticipantEstimate{
			UserID:        participant.UserID,
			Email:         participant.Email,
			Username:      participant.Username,
			Status:        participant.Status,
			RawUsage:      roundMoney(raw),
			WeightedUsage: roundMoney(weighted),
			CurrentTier:   CurrentSettlementTier(raw, tiers),
		})
	}

	fixedShare := 0.0
	ownerLoss := 0.0
	if len(participants) > 0 {
		fixedShare = fixedPool / float64(len(participants))
	} else {
		ownerLoss += fixedPool
	}

	uncappedRate := 0.0
	effectiveRate := 0.0
	if totalWeightedUsage > 0 {
		uncappedRate = dynamicPool / totalWeightedUsage
		effectiveRate = uncappedRate
		if cycle.MarketCap >= 0 && effectiveRate > cycle.MarketCap {
			effectiveRate = cycle.MarketCap
		}
	} else {
		ownerLoss += dynamicPool
	}

	recoveredDynamic := 0.0
	for i := range rows {
		rows[i].FixedShare = roundMoney(fixedShare)
		rows[i].DynamicCharge = roundMoney(effectiveRate * rows[i].WeightedUsage)
		rows[i].TotalDue = roundMoney(rows[i].FixedShare + rows[i].DynamicCharge)
		recoveredDynamic += rows[i].DynamicCharge
	}
	if totalWeightedUsage > 0 {
		ownerLoss += math.Max(dynamicPool-recoveredDynamic, 0)
	}

	return &SettlementPoolEstimate{
		GroupID:              cycle.GroupID,
		CycleID:              cycle.ID,
		Status:               cycle.Status,
		StartedAt:            cycle.StartedAt,
		EndedAt:              cycle.EndedAt,
		TotalCost:            roundMoney(totalCost),
		BaseRatio:            baseRatio,
		MarketCap:            cycle.MarketCap,
		Tiers:                cloneSettlementPoolTiers(tiers),
		ParticipantCount:     len(participants),
		FixedPool:            roundMoney(fixedPool),
		DynamicPool:          roundMoney(dynamicPool),
		TotalRawUsage:        roundMoney(totalRawUsage),
		TotalWeightedUsage:   roundMoney(totalWeightedUsage),
		UncappedDynamicRate:  roundMoney(uncappedRate),
		EffectiveDynamicRate: roundMoney(effectiveRate),
		OwnerCoveredLoss:     roundMoney(ownerLoss),
		Participants:         rows,
	}
}

func WeightedSettlementUsage(rawUsage float64, tiers []SettlementPoolTier) float64 {
	if rawUsage <= 0 {
		return 0
	}
	tiers, err := normalizeSettlementPoolTiers(tiers)
	if err != nil {
		return 0
	}
	remainingStart := 0.0
	weighted := 0.0
	for _, tier := range tiers {
		upper := rawUsage
		if tier.UpTo != nil && *tier.UpTo < upper {
			upper = *tier.UpTo
		}
		if upper > remainingStart {
			weighted += (upper - remainingStart) * tier.Weight
		}
		if tier.UpTo == nil || rawUsage <= *tier.UpTo {
			break
		}
		remainingStart = *tier.UpTo
	}
	return weighted
}

func CurrentSettlementTier(rawUsage float64, tiers []SettlementPoolTier) int {
	if rawUsage <= 0 || len(tiers) == 0 {
		return 0
	}
	normalized, err := normalizeSettlementPoolTiers(tiers)
	if err != nil {
		return 0
	}
	for i, tier := range normalized {
		if tier.UpTo == nil || rawUsage <= *tier.UpTo {
			return i
		}
	}
	return len(normalized) - 1
}

func settlementPoolGroupFromGroup(group *Group) *SettlementPoolGroup {
	if group == nil {
		return nil
	}
	return &SettlementPoolGroup{
		ID:               group.ID,
		Name:             group.Name,
		Description:      group.Description,
		Platform:         group.Platform,
		SubscriptionType: group.SubscriptionType,
		Status:           group.Status,
		RateMultiplier:   group.RateMultiplier,
	}
}

func (s *SettlementPoolService) requireSettlementPoolGroup(ctx context.Context, groupID int64) (*Group, error) {
	if s == nil || s.repo == nil || s.groupRepo == nil {
		return nil, ErrSettlementPoolNotFound
	}
	group, err := s.groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !group.IsSettlementPoolType() {
		return nil, ErrSettlementPoolNotFound
	}
	return group, nil
}

func (s *SettlementPoolService) ensureActiveCycle(ctx context.Context, groupID int64) (*SettlementPoolConfig, *SettlementPoolCycle, error) {
	config, active, err := s.repo.EnsureActiveCycle(ctx, groupID, DefaultSettlementPoolConfig(groupID))
	if err != nil {
		return nil, nil, fmt.Errorf("ensure active settlement cycle: %w", err)
	}
	return config, active, nil
}

func settlementCyclesByGroup(cycles []SettlementPoolCycle) map[int64][]SettlementPoolCycle {
	out := make(map[int64][]SettlementPoolCycle)
	for _, cycle := range cycles {
		out[cycle.GroupID] = append(out[cycle.GroupID], cycle)
	}
	return out
}

func nonNilSettlementCycles(cycles []SettlementPoolCycle) []SettlementPoolCycle {
	if cycles == nil {
		return []SettlementPoolCycle{}
	}
	return cycles
}

func nonNilSettlementParticipants(participants []SettlementPoolParticipant) []SettlementPoolParticipant {
	if participants == nil {
		return []SettlementPoolParticipant{}
	}
	return participants
}

func nonNilSettlementAccountUsage(rows []SettlementPoolAccountUsage) []SettlementPoolAccountUsage {
	if rows == nil {
		return []SettlementPoolAccountUsage{}
	}
	for i := range rows {
		rows[i].TotalUsage = roundMoney(rows[i].TotalUsage)
	}
	return rows
}

func normalizeSettlementPoolConfigInput(input SettlementPoolConfigInput) (SettlementPoolConfigInput, error) {
	if input.TotalCost < 0 || input.BaseRatio < 0 || input.BaseRatio > 1 || input.MarketCap < 0 {
		return SettlementPoolConfigInput{}, ErrSettlementPoolInvalidConfig
	}
	tiers, err := normalizeSettlementPoolTiers(input.Tiers)
	if err != nil {
		return SettlementPoolConfigInput{}, err
	}
	input.Tiers = tiers
	return input, nil
}

func normalizeSettlementPoolTiers(tiers []SettlementPoolTier) ([]SettlementPoolTier, error) {
	if len(tiers) == 0 {
		tiers = DefaultSettlementPoolTiers()
	}
	out := cloneSettlementPoolTiers(tiers)
	for i := range out {
		if out[i].Weight < 0 {
			return nil, ErrSettlementPoolInvalidConfig
		}
		if out[i].UpTo != nil && *out[i].UpTo <= 0 {
			return nil, ErrSettlementPoolInvalidConfig
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpTo == nil {
			return false
		}
		if out[j].UpTo == nil {
			return true
		}
		return *out[i].UpTo < *out[j].UpTo
	})
	for i := 1; i < len(out); i++ {
		if out[i-1].UpTo == nil {
			return nil, ErrSettlementPoolInvalidConfig
		}
		if out[i].UpTo != nil && *out[i].UpTo <= *out[i-1].UpTo {
			return nil, ErrSettlementPoolInvalidConfig
		}
	}
	if out[len(out)-1].UpTo != nil {
		out = append(out, SettlementPoolTier{UpTo: nil, Weight: out[len(out)-1].Weight})
	}
	return out, nil
}

func cloneSettlementPoolTiers(tiers []SettlementPoolTier) []SettlementPoolTier {
	out := make([]SettlementPoolTier, 0, len(tiers))
	for _, tier := range tiers {
		clone := SettlementPoolTier{Weight: tier.Weight}
		if tier.UpTo != nil {
			v := *tier.UpTo
			clone.UpTo = &v
		}
		out = append(out, clone)
	}
	return out
}

func uniquePositiveInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func roundMoney(value float64) float64 {
	return math.Round(value*1e10) / 1e10
}

func MarshalSettlementEstimate(estimate *SettlementPoolEstimate) ([]byte, error) {
	if estimate == nil {
		return nil, nil
	}
	return json.Marshal(estimate)
}
