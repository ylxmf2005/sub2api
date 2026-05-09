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

func TestSettlementPoolRepositoryListEnabledAccountUsageShowsZeroUsageAccounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &settlementPoolRepository{db: db}
	groupID := int64(11)
	startedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)

	mock.ExpectQuery(`(?s)WITH enabled_accounts AS .*FROM account_groups.*a\.schedulable = TRUE.*durable_usage AS .*FROM usage_billing_events.*legacy_usage AS .*FROM usage_logs.*LEFT JOIN usage_billing_events.*LEFT JOIN usage_totals.*ORDER BY`).
		WithArgs(groupID, startedAt, sqlmock.AnyArg(), service.BillingTypeSettlementPool, service.StatusActive).
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
		))

	rows, err := repo.ListEnabledAccountUsage(context.Background(), groupID, startedAt, &endedAt)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(101), rows[0].AccountID)
	require.Equal(t, "open account", rows[0].Name)
	require.True(t, rows[0].Schedulable)
	require.Zero(t, rows[0].TotalUsage)
	require.NoError(t, mock.ExpectationsWereMet())
}
