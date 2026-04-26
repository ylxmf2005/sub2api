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

	mock.ExpectQuery(`(?s)SELECT user_id, COALESCE\(SUM\(total_cost\), 0\).*AND billing_type = \$5.*GROUP BY user_id`).
		WithArgs(groupID, sqlmock.AnyArg(), startedAt, sqlmock.AnyArg(), service.BillingTypeSettlementPool).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "usage"}).AddRow(userID, 12.5))

	usage, err := repo.SumUsageByUsers(context.Background(), groupID, []int64{userID}, startedAt, &endedAt)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{userID: 12.5}, usage)
	require.NoError(t, mock.ExpectationsWereMet())
}
