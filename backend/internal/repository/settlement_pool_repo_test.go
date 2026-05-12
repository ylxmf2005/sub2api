package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSettlementPoolRepositorySumUsageByUsersFiltersSettlementBillingType(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	groupID := int64(11)
	userID := int64(7)
	startedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)WITH durable_usage AS .*FROM usage_billing_events.*legacy_usage AS .*FROM usage_logs.*LEFT JOIN usage_billing_events.*UNION ALL.*GROUP BY user_id`).
		WithArgs(groupID, sqlmock.AnyArg(), startedAt, sqlmock.AnyArg(), service.BillingTypeSettlementPool).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "usage"}).AddRow(userID, 12.5))

	usage, err := repo.SumUsageByUsers(context.Background(), groupID, []int64{userID}, startedAt, &endedAt)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{userID: 12.5}, usage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementPoolRepositorySumManualUsageByUsers(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	cycleID := int64(22)
	userID := int64(7)

	mock.ExpectQuery(`(?s)SELECT user_id, COALESCE\(SUM\(usage_amount\), 0\).*FROM settlement_pool_manual_usage_adjustments.*WHERE cycle_id = \$1.*AND user_id = ANY\(\$2\).*GROUP BY user_id`).
		WithArgs(cycleID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "usage"}).AddRow(userID, 8.25))

	usage, err := repo.SumManualUsageByUsers(context.Background(), cycleID, []int64{userID})
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{userID: 8.25}, usage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementPoolRepositorySumManualUsageByAccounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	cycleID := int64(22)
	accountID := int64(101)

	mock.ExpectQuery(`(?s)SELECT account_id, COALESCE\(SUM\(usage_amount\), 0\).*FROM settlement_pool_manual_usage_adjustments.*WHERE cycle_id = \$1.*AND account_id = ANY\(\$2\).*GROUP BY account_id`).
		WithArgs(cycleID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "usage"}).AddRow(accountID, -2.25))

	usage, err := repo.SumManualUsageByAccounts(context.Background(), cycleID, []int64{accountID})
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{accountID: -2.25}, usage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementPoolRepositoryCreateManualUsageAdjustment(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	createdAt := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	adjustment := &service.SettlementPoolManualUsageAdjustment{
		GroupID:     11,
		CycleID:     22,
		UserID:      7,
		AccountID:   101,
		UsageAmount: 8.25,
		Reason:      "outside proxy",
		CreatedBy:   1,
	}

	mock.ExpectQuery(`(?s)INSERT INTO settlement_pool_manual_usage_adjustments .*RETURNING id, created_at`).
		WithArgs(adjustment.GroupID, adjustment.CycleID, adjustment.UserID, adjustment.AccountID, adjustment.UsageAmount, adjustment.Reason, adjustment.CreatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(99), createdAt))

	err = repo.CreateManualUsageAdjustment(context.Background(), adjustment)
	require.NoError(t, err)
	require.Equal(t, int64(99), adjustment.ID)
	require.Equal(t, createdAt, adjustment.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementPoolRepositoryListEnabledAccountUsageShowsZeroUsageAccounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)WITH enabled_accounts AS .*FROM account_groups.*a\.schedulable = TRUE.*durable_usage AS .*FROM usage_billing_events.*legacy_usage AS .*FROM usage_logs.*LEFT JOIN usage_billing_events.*weekly_usage_totals AS .*LEFT JOIN usage_totals.*LEFT JOIN weekly_usage_totals.*ORDER BY`).
		WithArgs(groupID, startedAt, sqlmock.AnyArg(), service.BillingTypeSettlementPool, service.StatusActive, sqlmock.AnyArg(), int64(0), false, false).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"platform",
			"type",
			"status",
			"schedulable",
			"requests",
			"input_tokens",
			"output_tokens",
			"cache_creation_tokens",
			"cache_read_tokens",
			"total_tokens",
			"total_usage",
			"weekly_total_usage",
		}).AddRow(
			int64(101),
			"open account",
			service.PlatformOpenAI,
			service.AccountTypeOAuth,
			service.StatusActive,
			true,
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			float64(0),
			float64(7.5),
		))

	rows, err := repo.ListEnabledAccountUsage(context.Background(), groupID, startedAt, &endedAt)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(101), rows[0].AccountID)
	require.Equal(t, "open account", rows[0].Name)
	require.True(t, rows[0].Schedulable)
	require.Zero(t, rows[0].TotalUsage)
	require.Equal(t, 7.5, rows[0].WeeklyTotalUsage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementPoolRepositoryListSettlementAccountUsageIncludesHistoricalUsedAccounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	groupID := int64(11)
	cycleID := int64(22)
	startedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)WITH enabled_accounts AS .*usage_totals AS .*manual_usage_totals AS .*usage_accounts AS .*JOIN accounts a ON a\.id = used\.account_id.*account_scope AS .*FROM account_scope.*ORDER BY`).
		WithArgs(groupID, startedAt, sqlmock.AnyArg(), service.BillingTypeSettlementPool, service.StatusActive, sqlmock.AnyArg(), cycleID, true, true).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"platform",
			"type",
			"status",
			"schedulable",
			"requests",
			"input_tokens",
			"output_tokens",
			"cache_creation_tokens",
			"cache_read_tokens",
			"total_tokens",
			"total_usage",
			"weekly_total_usage",
		}).AddRow(
			int64(101),
			"deleted historical account",
			service.PlatformAnthropic,
			service.AccountTypeOAuth,
			service.StatusError,
			false,
			int64(2),
			int64(10),
			int64(20),
			int64(3),
			int64(4),
			int64(37),
			float64(24.95),
			float64(24.95),
		))

	rows, err := repo.ListSettlementAccountUsage(context.Background(), groupID, cycleID, startedAt, &endedAt)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(101), rows[0].AccountID)
	require.Equal(t, "deleted historical account", rows[0].Name)
	require.Equal(t, service.StatusError, rows[0].Status)
	require.False(t, rows[0].Schedulable)
	require.Equal(t, int64(2), rows[0].Requests)
	require.Equal(t, int64(37), rows[0].TotalTokens)
	require.Equal(t, 24.95, rows[0].TotalUsage)
	require.NoError(t, mock.ExpectationsWereMet())
}
