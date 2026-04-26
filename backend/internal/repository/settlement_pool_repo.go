package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type settlementPoolRepository struct {
	db *sql.DB
}

func NewSettlementPoolRepository(_ *dbent.Client, sqlDB *sql.DB) service.SettlementPoolRepository {
	return &settlementPoolRepository{db: sqlDB}
}

func (r *settlementPoolRepository) GetConfig(ctx context.Context, groupID int64) (*service.SettlementPoolConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT group_id, base_ratio, market_cap, tiers, active_cycle_id, created_at, updated_at
		FROM settlement_pool_configs
		WHERE group_id = $1
	`, groupID)
	config, err := scanSettlementConfig(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSettlementPoolNotFound
		}
		return nil, err
	}
	return config, nil
}

func (r *settlementPoolRepository) EnsureActiveCycle(ctx context.Context, groupID int64, defaults *service.SettlementPoolConfig) (*service.SettlementPoolConfig, *service.SettlementPoolCycle, error) {
	if defaults == nil {
		defaults = service.DefaultSettlementPoolConfig(groupID)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	tiersJSON, err := json.Marshal(defaults.Tiers)
	if err != nil {
		return nil, nil, fmt.Errorf("encode settlement tiers: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO settlement_pool_configs (group_id, base_ratio, market_cap, tiers, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW())
		ON CONFLICT (group_id) DO NOTHING
	`, groupID, defaults.BaseRatio, defaults.MarketCap, string(tiersJSON)); err != nil {
		return nil, nil, err
	}

	config, err := scanSettlementConfig(tx.QueryRowContext(ctx, `
		SELECT group_id, base_ratio, market_cap, tiers, active_cycle_id, created_at, updated_at
		FROM settlement_pool_configs
		WHERE group_id = $1
		FOR UPDATE
	`, groupID))
	if err != nil {
		return nil, nil, err
	}

	active, err := scanSettlementCycle(tx.QueryRowContext(ctx, settlementCycleSelectSQL+`
		FROM settlement_pool_cycles
		WHERE group_id = $1 AND status = 'active'
		ORDER BY started_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, groupID))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
		active = &service.SettlementPoolCycle{
			GroupID:   groupID,
			Status:    service.SettlementPoolCycleStatusActive,
			StartedAt: time.Now(),
			TotalCost: 0,
			BaseRatio: config.BaseRatio,
			MarketCap: config.MarketCap,
			Tiers:     cloneRepositorySettlementTiers(config.Tiers),
		}
		if err := insertSettlementCycle(ctx, tx, active); err != nil {
			return nil, nil, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE settlement_pool_configs
			SET active_cycle_id = $2, updated_at = NOW()
			WHERE group_id = $1
		`, groupID, active.ID); err != nil {
			return nil, nil, err
		}
		config.ActiveCycleID = &active.ID
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return config, active, nil
}

func (r *settlementPoolRepository) UpsertConfig(ctx context.Context, config *service.SettlementPoolConfig) error {
	if config == nil {
		return nil
	}
	tiersJSON, err := json.Marshal(config.Tiers)
	if err != nil {
		return fmt.Errorf("encode settlement tiers: %w", err)
	}
	var activeCycleID any
	if config.ActiveCycleID != nil {
		activeCycleID = *config.ActiveCycleID
	}
	return execNoRows(r.db.ExecContext(ctx, `
		INSERT INTO settlement_pool_configs (group_id, base_ratio, market_cap, tiers, active_cycle_id, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			base_ratio = EXCLUDED.base_ratio,
			market_cap = EXCLUDED.market_cap,
			tiers = EXCLUDED.tiers,
			active_cycle_id = COALESCE(EXCLUDED.active_cycle_id, settlement_pool_configs.active_cycle_id),
			updated_at = NOW()
	`, config.GroupID, config.BaseRatio, config.MarketCap, string(tiersJSON), activeCycleID))
}

func (r *settlementPoolRepository) SetActiveCycleID(ctx context.Context, groupID int64, cycleID *int64) error {
	var value any
	if cycleID != nil {
		value = *cycleID
	}
	return execNoRows(r.db.ExecContext(ctx, `
		UPDATE settlement_pool_configs
		SET active_cycle_id = $2, updated_at = NOW()
		WHERE group_id = $1
	`, groupID, value))
}

func (r *settlementPoolRepository) GetActiveCycle(ctx context.Context, groupID int64) (*service.SettlementPoolCycle, error) {
	row := r.db.QueryRowContext(ctx, settlementCycleSelectSQL+`
		FROM settlement_pool_cycles
		WHERE group_id = $1 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1
	`, groupID)
	cycle, err := scanSettlementCycle(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSettlementPoolNotFound
		}
		return nil, err
	}
	return cycle, nil
}

func (r *settlementPoolRepository) CreateCycle(ctx context.Context, cycle *service.SettlementPoolCycle) error {
	if cycle == nil {
		return nil
	}
	return insertSettlementCycle(ctx, r.db, cycle)
}

func (r *settlementPoolRepository) UpdateActiveCycleConfig(ctx context.Context, groupID int64, input service.SettlementPoolConfigInput) error {
	tiersJSON, err := json.Marshal(input.Tiers)
	if err != nil {
		return fmt.Errorf("encode settlement tiers: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE settlement_pool_cycles
		SET total_cost = $2,
			base_ratio = $3,
			market_cap = $4,
			tiers = $5::jsonb,
			updated_at = NOW()
		WHERE group_id = $1 AND status = 'active'
	`, groupID, input.TotalCost, input.BaseRatio, input.MarketCap, string(tiersJSON))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSettlementPoolNotFound
	}
	return nil
}

func (r *settlementPoolRepository) LockCycle(ctx context.Context, cycleID int64, endedAt time.Time, snapshot *service.SettlementPoolEstimate) error {
	snapshotJSON, err := settlementSnapshotJSONValue(snapshot)
	if err != nil {
		return fmt.Errorf("encode settlement snapshot: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE settlement_pool_cycles
		SET status = 'locked',
			ended_at = $2,
			snapshot = $3::jsonb,
			updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, cycleID, endedAt, snapshotJSON)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSettlementPoolNotFound
	}
	return nil
}

func (r *settlementPoolRepository) RotateCycle(ctx context.Context, groupID, activeCycleID int64, endedAt time.Time, snapshot *service.SettlementPoolEstimate, next *service.SettlementPoolCycle) error {
	if next == nil {
		return service.ErrSettlementPoolInvalidConfig
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	snapshotJSON, err := settlementSnapshotJSONValue(snapshot)
	if err != nil {
		return fmt.Errorf("encode settlement snapshot: %w", err)
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE settlement_pool_cycles
		SET status = 'locked',
			ended_at = $3,
			snapshot = $4::jsonb,
			updated_at = NOW()
		WHERE id = $1 AND group_id = $2 AND status = 'active'
	`, activeCycleID, groupID, endedAt, snapshotJSON)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSettlementPoolNotFound
	}

	if next.StartedAt.IsZero() {
		next.StartedAt = endedAt
	}
	if next.GroupID == 0 {
		next.GroupID = groupID
	}
	next.Status = service.SettlementPoolCycleStatusActive
	if err := insertSettlementCycle(ctx, tx, next); err != nil {
		return err
	}
	res, err = tx.ExecContext(ctx, `
		UPDATE settlement_pool_configs
		SET active_cycle_id = $2, updated_at = NOW()
		WHERE group_id = $1
	`, groupID, next.ID)
	if err != nil {
		return err
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSettlementPoolNotFound
	}
	return tx.Commit()
}

func (r *settlementPoolRepository) ListCycles(ctx context.Context, groupID int64, limit int) ([]service.SettlementPoolCycle, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, settlementCycleSelectSQL+`
		FROM settlement_pool_cycles
		WHERE group_id = $1
		ORDER BY started_at DESC, id DESC
		LIMIT $2
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSettlementCycles(rows)
}

func (r *settlementPoolRepository) ListCyclesForUser(ctx context.Context, userID int64, limit int) ([]service.SettlementPoolCycle, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	needle, err := json.Marshal([]map[string]int64{{"user_id": userID}})
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, settlementCycleSelectSQL+`
		FROM settlement_pool_cycles
		WHERE status = 'locked'
			AND snapshot IS NOT NULL
			AND snapshot->'participants' @> $1::jsonb
		ORDER BY started_at DESC, id DESC
		LIMIT $2
	`, string(needle), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSettlementCycles(rows)
}

func (r *settlementPoolRepository) ListParticipants(ctx context.Context, groupID int64) ([]service.SettlementPoolParticipant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.user_id, u.email, u.username, u.status, p.created_at
		FROM settlement_pool_participants p
		JOIN users u ON u.id = p.user_id AND u.deleted_at IS NULL
		WHERE p.group_id = $1
		ORDER BY p.created_at ASC, p.user_id ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	participants := make([]service.SettlementPoolParticipant, 0)
	for rows.Next() {
		var participant service.SettlementPoolParticipant
		if err := rows.Scan(&participant.UserID, &participant.Email, &participant.Username, &participant.Status, &participant.CreatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}
	return participants, rows.Err()
}

func (r *settlementPoolRepository) SyncParticipants(ctx context.Context, groupID int64, userIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if len(userIDs) > 0 {
		var existing int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM users
			WHERE id = ANY($1) AND deleted_at IS NULL
		`, pq.Array(userIDs)).Scan(&existing); err != nil {
			return err
		}
		if existing != len(userIDs) {
			return service.ErrUserNotFound
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM settlement_pool_participants WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_allowed_groups WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	if len(userIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settlement_pool_participants (group_id, user_id)
			SELECT $1, unnest($2::bigint[])
			ON CONFLICT (group_id, user_id) DO NOTHING
		`, groupID, pq.Array(userIDs)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_allowed_groups (group_id, user_id)
			SELECT $1, unnest($2::bigint[])
			ON CONFLICT (user_id, group_id) DO NOTHING
		`, groupID, pq.Array(userIDs)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *settlementPoolRepository) IsParticipant(ctx context.Context, userID, groupID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM settlement_pool_participants p
			JOIN users u ON u.id = p.user_id AND u.deleted_at IS NULL AND u.status = $3
			WHERE p.user_id = $1 AND p.group_id = $2
		)
	`, userID, groupID, service.StatusActive).Scan(&exists)
	return exists, err
}

func (r *settlementPoolRepository) ListUserPoolGroupIDs(ctx context.Context, userID int64) ([]int64, error) {
	needle, err := json.Marshal([]map[string]int64{{"user_id": userID}})
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT group_id
		FROM (
			SELECT p.group_id
			FROM settlement_pool_participants p
			WHERE p.user_id = $1
			UNION
			SELECT c.group_id
			FROM settlement_pool_cycles c
			WHERE c.status = 'locked'
				AND c.snapshot IS NOT NULL
				AND c.snapshot->'participants' @> $2::jsonb
		) s
		ORDER BY group_id ASC
	`, userID, string(needle))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	groupIDs := make([]int64, 0)
	for rows.Next() {
		var groupID int64
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}
	return groupIDs, rows.Err()
}

func (r *settlementPoolRepository) SumUsageByUsers(ctx context.Context, groupID int64, userIDs []int64, startedAt time.Time, endedAt *time.Time) (map[int64]float64, error) {
	out := make(map[int64]float64, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, COALESCE(SUM(total_cost), 0)
		FROM usage_logs
		WHERE group_id = $1
			AND user_id = ANY($2)
			AND created_at >= $3
			AND ($4::timestamptz IS NULL OR created_at < $4)
			AND billing_type = $5
		GROUP BY user_id
	`, groupID, pq.Array(userIDs), startedAt, endedAt, service.BillingTypeSettlementPool)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var userID int64
		var usage float64
		if err := rows.Scan(&userID, &usage); err != nil {
			return nil, err
		}
		out[userID] = usage
	}
	return out, rows.Err()
}

const settlementCycleSelectSQL = `
		SELECT id, group_id, status, started_at, ended_at, total_cost, base_ratio, market_cap, tiers, snapshot, created_at, updated_at
`

type settlementScanner interface {
	Scan(dest ...any) error
}

type settlementCycleInserter interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanSettlementConfig(scanner settlementScanner) (*service.SettlementPoolConfig, error) {
	var config service.SettlementPoolConfig
	var tiersJSON []byte
	var activeCycleID sql.NullInt64
	if err := scanner.Scan(&config.GroupID, &config.BaseRatio, &config.MarketCap, &tiersJSON, &activeCycleID, &config.CreatedAt, &config.UpdatedAt); err != nil {
		return nil, err
	}
	if activeCycleID.Valid {
		config.ActiveCycleID = &activeCycleID.Int64
	}
	if err := json.Unmarshal(tiersJSON, &config.Tiers); err != nil {
		return nil, fmt.Errorf("decode settlement tiers: %w", err)
	}
	return &config, nil
}

func scanSettlementCycle(scanner settlementScanner) (*service.SettlementPoolCycle, error) {
	var cycle service.SettlementPoolCycle
	var tiersJSON []byte
	var snapshotJSON []byte
	if err := scanner.Scan(
		&cycle.ID,
		&cycle.GroupID,
		&cycle.Status,
		&cycle.StartedAt,
		&cycle.EndedAt,
		&cycle.TotalCost,
		&cycle.BaseRatio,
		&cycle.MarketCap,
		&tiersJSON,
		&snapshotJSON,
		&cycle.CreatedAt,
		&cycle.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tiersJSON, &cycle.Tiers); err != nil {
		return nil, fmt.Errorf("decode settlement tiers: %w", err)
	}
	if len(snapshotJSON) > 0 {
		var snapshot service.SettlementPoolEstimate
		if err := json.Unmarshal(snapshotJSON, &snapshot); err != nil {
			return nil, fmt.Errorf("decode settlement snapshot: %w", err)
		}
		cycle.Snapshot = &snapshot
	}
	return &cycle, nil
}

func scanSettlementCycles(rows *sql.Rows) ([]service.SettlementPoolCycle, error) {
	cycles := make([]service.SettlementPoolCycle, 0)
	for rows.Next() {
		cycle, err := scanSettlementCycle(rows)
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, *cycle)
	}
	return cycles, rows.Err()
}

func execNoRows(_ sql.Result, err error) error {
	return err
}

func insertSettlementCycle(ctx context.Context, db settlementCycleInserter, cycle *service.SettlementPoolCycle) error {
	tiersJSON, err := json.Marshal(cycle.Tiers)
	if err != nil {
		return fmt.Errorf("encode settlement tiers: %w", err)
	}
	if cycle.StartedAt.IsZero() {
		cycle.StartedAt = time.Now()
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO settlement_pool_cycles (group_id, status, started_at, total_cost, base_ratio, market_cap, tiers)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, created_at, updated_at
	`, cycle.GroupID, cycle.Status, cycle.StartedAt, cycle.TotalCost, cycle.BaseRatio, cycle.MarketCap, string(tiersJSON)).
		Scan(&cycle.ID, &cycle.CreatedAt, &cycle.UpdatedAt)
}

func settlementSnapshotJSONValue(snapshot *service.SettlementPoolEstimate) (any, error) {
	if snapshot == nil {
		return nil, nil
	}
	snapshotJSON, err := service.MarshalSettlementEstimate(snapshot)
	if err != nil {
		return nil, err
	}
	return string(snapshotJSON), nil
}

func cloneRepositorySettlementTiers(tiers []service.SettlementPoolTier) []service.SettlementPoolTier {
	out := make([]service.SettlementPoolTier, 0, len(tiers))
	for _, tier := range tiers {
		clone := service.SettlementPoolTier{Weight: tier.Weight}
		if tier.UpTo != nil {
			v := *tier.UpTo
			clone.UpTo = &v
		}
		out = append(out, clone)
	}
	return out
}
