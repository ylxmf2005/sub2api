//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSettlementPoolRepositoryListSettlementAccountUsageIncludesDeletedUnboundCycleUsage(t *testing.T) {
	ctx := context.Background()
	repo := &settlementPoolRepository{db: integrationDB}

	now := time.Now().UTC()
	startedAt := now.Add(-24 * time.Hour)
	groupID := insertSettlementFixtureGroup(t, integrationDB, "settlement-history")
	userID := insertSettlementFixtureUser(t, integrationDB, "settlement-history-user@example.com")
	apiKeyID := insertSettlementFixtureAPIKey(t, integrationDB, userID, groupID)
	accountID := insertSettlementFixtureAccount(t, integrationDB, "ClaudeAcc", service.StatusActive, true, nil)
	boundAccountID := insertSettlementFixtureAccount(t, integrationDB, "current account", service.StatusActive, true, nil)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM groups WHERE id = $1`, groupID)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM accounts WHERE id = ANY($1)`, pq.Array([]int64{accountID, boundAccountID}))
	})
	cycleID := insertSettlementFixtureCycle(t, integrationDB, groupID, startedAt)
	insertSettlementFixtureAccountGroup(t, integrationDB, accountID, groupID, 1)
	insertSettlementFixtureAccountGroup(t, integrationDB, boundAccountID, groupID, 2)
	insertSettlementFixtureUsageLog(t, integrationDB, userID, apiKeyID, accountID, groupID, "req-history", startedAt.Add(time.Hour), 24.9478075)

	_, err := integrationDB.ExecContext(ctx, `DELETE FROM account_groups WHERE account_id = $1 AND group_id = $2`, accountID, groupID)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE accounts SET status = $2, schedulable = FALSE, deleted_at = $3 WHERE id = $1`, accountID, service.StatusError, startedAt.Add(2*time.Hour))
	require.NoError(t, err)

	displayRows, err := repo.ListSettlementAccountUsage(ctx, groupID, cycleID, startedAt, nil)
	require.NoError(t, err)
	require.Len(t, displayRows, 2)
	require.Equal(t, accountID, displayRows[0].AccountID)
	require.Equal(t, "ClaudeAcc", displayRows[0].Name)
	require.Equal(t, service.StatusError, displayRows[0].Status)
	require.False(t, displayRows[0].Schedulable)
	require.Equal(t, int64(1), displayRows[0].Requests)
	require.InDelta(t, 24.9478075, displayRows[0].TotalUsage, 1e-9)

	enabledRows, err := repo.ListEnabledAccountUsage(ctx, groupID, startedAt, nil)
	require.NoError(t, err)
	require.Len(t, enabledRows, 1)
	require.Equal(t, boundAccountID, enabledRows[0].AccountID)
	require.Zero(t, enabledRows[0].TotalUsage)
}

type settlementFixtureDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func insertSettlementFixtureGroup(t *testing.T, db settlementFixtureDB, name string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO groups (name, platform, subscription_type, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, name+"-"+time.Now().Format("150405.000000000"), service.PlatformAnthropic, service.SubscriptionTypeSettlementPool, service.StatusActive).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSettlementFixtureUser(t *testing.T, db settlementFixtureDB, email string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, 'test-password-hash', $2, $3)
		RETURNING id
	`, time.Now().Format("150405.000000000")+"-"+email, service.RoleUser, service.StatusActive).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSettlementFixtureAPIKey(t *testing.T, db settlementFixtureDB, userID, groupID int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO api_keys (user_id, key, name, group_id, status)
		VALUES ($1, $2, 'settlement-test-key', $3, $4)
		RETURNING id
	`, userID, "sk-settlement-"+time.Now().Format("150405.000000000"), groupID, service.StatusActive).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSettlementFixtureAccount(t *testing.T, db settlementFixtureDB, name, status string, schedulable bool, deletedAt *time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO accounts (name, platform, type, credentials, extra, status, schedulable, deleted_at)
		VALUES ($1, $2, $3, '{}'::jsonb, '{}'::jsonb, $4, $5, $6)
		RETURNING id
	`, name, service.PlatformAnthropic, service.AccountTypeOAuth, status, schedulable, deletedAt).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSettlementFixtureCycle(t *testing.T, db settlementFixtureDB, groupID int64, startedAt time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO settlement_pool_cycles (group_id, status, started_at, total_cost)
		VALUES ($1, $2, $3, 100)
		RETURNING id
	`, groupID, service.SettlementPoolCycleStatusActive, startedAt).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertSettlementFixtureAccountGroup(t *testing.T, db settlementFixtureDB, accountID, groupID int64, priority int) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO account_groups (account_id, group_id, priority)
		VALUES ($1, $2, $3)
	`, accountID, groupID, priority)
	require.NoError(t, err)
}

func insertSettlementFixtureUsageLog(t *testing.T, db settlementFixtureDB, userID, apiKeyID, accountID, groupID int64, requestID string, createdAt time.Time, totalCost float64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO usage_logs (
			user_id, api_key_id, account_id, request_id, model, group_id, billing_type,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			total_cost, actual_cost, created_at
		)
		VALUES ($1, $2, $3, $4, 'claude-test', $5, $6, 10, 20, 3, 4, $7, $7, $8)
	`, userID, apiKeyID, accountID, requestID, groupID, service.BillingTypeSettlementPool, totalCost, createdAt)
	require.NoError(t, err)
}
