package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type resourceSupplyRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewResourceSupplyRepository(client *dbent.Client, sqlDB *sql.DB) service.ResourceSupplyRepository {
	return &resourceSupplyRepository{client: client, db: sqlDB}
}

func (r *resourceSupplyRepository) GetUserSummary(ctx context.Context, userID int64, recentLimit int) (*service.ResourceSupplySummary, error) {
	if recentLimit <= 0 || recentLimit > 100 {
		recentLimit = 20
	}
	if err := r.ensureBalance(ctx, userID); err != nil {
		return nil, err
	}
	balance, err := r.getBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	accounts, counts, err := r.listOwnedAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	ledger, _, err := r.ListUserLedger(ctx, userID, service.ResourceSupplyLedgerFilter{Page: 1, PageSize: recentLimit})
	if err != nil {
		return nil, err
	}
	return &service.ResourceSupplySummary{
		Balance:             *balance,
		AccountStatusCounts: counts,
		Accounts:            accounts,
		RecentLedger:        ledger,
	}, nil
}

func (r *resourceSupplyRepository) ListUserLedger(ctx context.Context, userID int64, filter service.ResourceSupplyLedgerFilter) ([]service.ResourceSupplyLedgerEntry, int64, error) {
	page, pageSize := normalizeRepoPage(filter.Page, filter.PageSize)
	args := []any{userID}
	clauses := []string{"l.owner_user_id = $1"}
	if ledgerType := strings.TrimSpace(filter.LedgerType); ledgerType != "" {
		args = append(args, ledgerType)
		clauses = append(clauses, fmt.Sprintf("l.ledger_type = $%d", len(args)))
	}
	return r.queryLedger(ctx, clauses, args, page, pageSize)
}

func (r *resourceSupplyRepository) ListAdminLedger(ctx context.Context, filter service.ResourceSupplyAdminLedgerFilter) ([]service.ResourceSupplyLedgerEntry, int64, error) {
	page, pageSize := normalizeRepoPage(filter.Page, filter.PageSize)
	args := make([]any, 0, 8)
	clauses := make([]string, 0, 8)
	addIntFilter := func(column string, value *int64) {
		if value == nil || *value <= 0 {
			return
		}
		args = append(args, *value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	addIntFilter("l.owner_user_id", filter.OwnerUserID)
	addIntFilter("l.caller_user_id", filter.CallerUserID)
	addIntFilter("l.group_id", filter.GroupID)
	addIntFilter("l.account_id", filter.AccountID)
	if ledgerType := strings.TrimSpace(filter.LedgerType); ledgerType != "" {
		args = append(args, ledgerType)
		clauses = append(clauses, fmt.Sprintf("l.ledger_type = $%d", len(args)))
	}
	if requestID := strings.TrimSpace(filter.RequestID); requestID != "" {
		args = append(args, requestID)
		clauses = append(clauses, fmt.Sprintf("l.request_id = $%d", len(args)))
	}
	return r.queryLedger(ctx, clauses, args, page, pageSize)
}

func (r *resourceSupplyRepository) TransferAvailableToBalance(ctx context.Context, userID int64, idempotencyKey string) (*service.ResourceSupplyTransferResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin resource supply transfer: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	txClient := tx.Client()

	if replay, ok, err := querySupplyTransferReplay(txCtx, txClient, userID, idempotencyKey); err != nil {
		return nil, err
	} else if ok {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return replay, nil
	}
	if err := ensureResourceSupplyBalanceWithClient(txCtx, txClient, userID); err != nil {
		return nil, err
	}

	available, err := lockSupplyBalance(txCtx, txClient, userID)
	if err != nil {
		return nil, err
	}
	if replay, ok, err := querySupplyTransferReplay(txCtx, txClient, userID, idempotencyKey); err != nil {
		return nil, err
	} else if ok {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return replay, nil
	}
	if available <= 0 {
		return nil, service.ErrResourceSupplyBalanceEmpty
	}

	if _, err := txClient.ExecContext(txCtx, `
		UPDATE resource_supply_balances
		SET available_amount = 0,
			lifetime_transferred_amount = lifetime_transferred_amount + $1,
			updated_at = NOW()
		WHERE user_id = $2
	`, available, userID); err != nil {
		return nil, err
	}

	var newBalance float64
	affected, err := txClient.User.Update().
		Where(user.IDEQ(userID)).
		AddBalance(available).
		AddTotalRecharged(available).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("credit user balance by resource supply transfer: %w", err)
	}
	if affected == 0 {
		return nil, service.ErrUserNotFound
	}
	if newBalance, err = queryUserBalance(txCtx, txClient, userID); err != nil {
		return nil, err
	}

	if _, err := txClient.ExecContext(txCtx, `
		INSERT INTO resource_supply_ledger (
			owner_user_id,
			ledger_type,
			idempotency_key,
			amount,
			balance_after,
			note,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, 0, 'transfer to balance', NOW(), NOW())
	`, userID, service.ResourceSupplyLedgerTypeTransfer, idempotencyKey, -available); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &service.ResourceSupplyTransferResult{Amount: available, NewBalance: newBalance}, nil
}

func (r *resourceSupplyRepository) AdjustBalance(ctx context.Context, input service.ResourceSupplyAdjustmentInput) (*service.ResourceSupplyLedgerEntry, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin resource supply adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	txClient := tx.Client()

	if err := ensureResourceSupplyBalanceWithClient(txCtx, txClient, input.OwnerUserID); err != nil {
		return nil, err
	}
	available, err := lockSupplyBalance(txCtx, txClient, input.OwnerUserID)
	if err != nil {
		return nil, err
	}
	balanceAfter := available + input.Amount
	if balanceAfter < 0 {
		return nil, service.ErrResourceSupplyWouldOverdraw
	}
	if _, err := txClient.ExecContext(txCtx, `
		UPDATE resource_supply_balances
		SET available_amount = $1,
			updated_at = NOW()
		WHERE user_id = $2
	`, balanceAfter, input.OwnerUserID); err != nil {
		return nil, err
	}

	var entryID int64
	rows, err := txClient.QueryContext(txCtx, `
		INSERT INTO resource_supply_ledger (
			owner_user_id,
			ledger_type,
			amount,
			balance_after,
			admin_user_id,
			note,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`, input.OwnerUserID, service.ResourceSupplyLedgerTypeAdjustment, input.Amount, balanceAfter, input.AdminUserID, strings.TrimSpace(input.Note))
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		if err := rows.Scan(&entryID); err != nil {
			_ = rows.Close()
			return nil, err
		}
	} else {
		_ = rows.Close()
		return nil, sql.ErrNoRows
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	entries, _, err := r.queryLedger(ctx, []string{"l.id = $1"}, []any{entryID}, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, service.ErrResourceSupplyBalanceEmpty
	}
	return &entries[0], nil
}

func (r *resourceSupplyRepository) ensureBalance(ctx context.Context, userID int64) error {
	return ensureResourceSupplyBalanceWithClient(ctx, clientFromContext(ctx, r.client), userID)
}

func (r *resourceSupplyRepository) getBalance(ctx context.Context, userID int64) (*service.ResourceSupplyBalance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT user_id,
			available_amount::double precision,
			lifetime_earned_amount::double precision,
			lifetime_transferred_amount::double precision,
			created_at,
			updated_at
		FROM resource_supply_balances
		WHERE user_id = $1
	`, userID)
	var balance service.ResourceSupplyBalance
	if err := row.Scan(
		&balance.UserID,
		&balance.AvailableAmount,
		&balance.LifetimeEarnedAmount,
		&balance.LifetimeTransferredAmount,
		&balance.CreatedAt,
		&balance.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &balance, nil
}

func (r *resourceSupplyRepository) listOwnedAccounts(ctx context.Context, userID int64) ([]service.ResourceSupplyOwnedAccount, map[string]int, error) {
	counts := make(map[string]int)
	countRows, err := r.db.QueryContext(ctx, `
		SELECT supply_status, COUNT(*)
		FROM accounts
		WHERE supply_owner_user_id = $1
			AND deleted_at IS NULL
		GROUP BY supply_status
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	for countRows.Next() {
		var status string
		var count int
		if err := countRows.Scan(&status, &count); err != nil {
			_ = countRows.Close()
			return nil, nil, err
		}
		counts[status] = count
	}
	if err := countRows.Close(); err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id,
			a.name,
			a.platform,
			a.type,
			COALESCE(a.supply_source, ''),
			a.supply_status,
			a.supply_status_reason,
			a.schedulable,
			a.created_at,
			a.updated_at,
			a.supply_reviewed_at,
			COALESCE(array_agg(g.id ORDER BY ag.priority, g.id) FILTER (WHERE g.id IS NOT NULL), ARRAY[]::bigint[]),
			COALESCE(array_agg(g.name ORDER BY ag.priority, g.id) FILTER (WHERE g.id IS NOT NULL), ARRAY[]::text[])
		FROM accounts a
		LEFT JOIN account_groups ag ON ag.account_id = a.id
		LEFT JOIN groups g ON g.id = ag.group_id AND g.deleted_at IS NULL
		WHERE a.supply_owner_user_id = $1
			AND a.deleted_at IS NULL
		GROUP BY a.id
		ORDER BY a.updated_at DESC, a.id DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	accounts := make([]service.ResourceSupplyOwnedAccount, 0)
	for rows.Next() {
		var item service.ResourceSupplyOwnedAccount
		var source string
		var reason sql.NullString
		var reviewedAt sql.NullTime
		var groupIDs pq.Int64Array
		var groupNames pq.StringArray
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Platform,
			&item.Type,
			&source,
			&item.SupplyStatus,
			&reason,
			&item.Schedulable,
			&item.CreatedAt,
			&item.UpdatedAt,
			&reviewedAt,
			&groupIDs,
			&groupNames,
		); err != nil {
			return nil, nil, err
		}
		item.SupplySource = source
		if reason.Valid {
			item.SupplyStatusReason = &reason.String
		}
		if reviewedAt.Valid {
			item.ReviewedAt = &reviewedAt.Time
		}
		item.GroupIDs = append([]int64(nil), groupIDs...)
		item.GroupNames = append([]string(nil), groupNames...)
		accounts = append(accounts, item)
	}
	return accounts, counts, rows.Err()
}

func (r *resourceSupplyRepository) queryLedger(ctx context.Context, clauses []string, args []any, page, pageSize int) ([]service.ResourceSupplyLedgerEntry, int64, error) {
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	countQuery := "SELECT COUNT(*) FROM resource_supply_ledger l " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	queryArgs := append([]any(nil), args...)
	queryArgs = append(queryArgs, pageSize, offset)
	limitPlaceholder := len(queryArgs) - 1
	offsetPlaceholder := len(queryArgs)
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id,
			l.owner_user_id,
			l.caller_user_id,
			l.api_key_id,
			l.group_id,
			l.account_id,
			l.usage_billing_event_id,
			l.ledger_type,
			l.amount::double precision,
			l.balance_after::double precision,
			l.actual_cost::double precision,
			l.reward_multiplier::double precision,
			l.billing_type,
			l.model,
			l.request_id,
			l.admin_user_id,
			l.note,
			l.created_at,
			l.updated_at,
			owner.email,
			caller.email,
			g.name,
			a.name
		FROM resource_supply_ledger l
		LEFT JOIN users owner ON owner.id = l.owner_user_id
		LEFT JOIN users caller ON caller.id = l.caller_user_id
		LEFT JOIN groups g ON g.id = l.group_id
		LEFT JOIN accounts a ON a.id = l.account_id
		`+where+`
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT $`+fmt.Sprint(limitPlaceholder)+` OFFSET $`+fmt.Sprint(offsetPlaceholder),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	entries, err := scanResourceSupplyLedgerRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func scanResourceSupplyLedgerRows(rows *sql.Rows) ([]service.ResourceSupplyLedgerEntry, error) {
	entries := make([]service.ResourceSupplyLedgerEntry, 0)
	for rows.Next() {
		var item service.ResourceSupplyLedgerEntry
		var callerUserID, apiKeyID, groupID, accountID, eventID, adminUserID sql.NullInt64
		var actualCost, rewardMultiplier sql.NullFloat64
		var billingType sql.NullInt64
		var model, requestID, note, ownerEmail, callerEmail, groupName, accountName sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.OwnerUserID,
			&callerUserID,
			&apiKeyID,
			&groupID,
			&accountID,
			&eventID,
			&item.LedgerType,
			&item.Amount,
			&item.BalanceAfter,
			&actualCost,
			&rewardMultiplier,
			&billingType,
			&model,
			&requestID,
			&adminUserID,
			&note,
			&item.CreatedAt,
			&item.UpdatedAt,
			&ownerEmail,
			&callerEmail,
			&groupName,
			&accountName,
		); err != nil {
			return nil, err
		}
		item.CallerUserID = nullableInt64Ptr(callerUserID)
		item.APIKeyID = nullableInt64Ptr(apiKeyID)
		item.GroupID = nullableInt64Ptr(groupID)
		item.AccountID = nullableInt64Ptr(accountID)
		item.UsageBillingEventID = nullableInt64Ptr(eventID)
		item.ActualCost = nullableFloat64Ptr(actualCost)
		item.RewardMultiplier = nullableFloat64Ptr(rewardMultiplier)
		if billingType.Valid {
			v := int8(billingType.Int64)
			item.BillingType = &v
		}
		item.Model = nullableStringPtr(model)
		item.RequestID = nullableStringPtr(requestID)
		item.AdminUserID = nullableInt64Ptr(adminUserID)
		item.Note = nullableStringPtr(note)
		item.OwnerEmail = nullableStringPtr(ownerEmail)
		item.CallerEmail = nullableStringPtr(callerEmail)
		item.GroupName = nullableStringPtr(groupName)
		item.AccountName = nullableStringPtr(accountName)
		entries = append(entries, item)
	}
	return entries, rows.Err()
}

func ensureResourceSupplyBalanceWithClient(ctx context.Context, client *dbent.Client, userID int64) error {
	_, err := client.ExecContext(ctx, `
		INSERT INTO resource_supply_balances (user_id, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func lockSupplyBalance(ctx context.Context, client *dbent.Client, userID int64) (float64, error) {
	var available float64
	rows, err := client.QueryContext(ctx, `
		SELECT available_amount::double precision
		FROM resource_supply_balances
		WHERE user_id = $1
		FOR UPDATE
	`, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrUserNotFound
	}
	err = rows.Scan(&available)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return available, err
}

func querySupplyTransferReplay(ctx context.Context, client *dbent.Client, userID int64, idempotencyKey string) (*service.ResourceSupplyTransferResult, bool, error) {
	var amount float64
	rows, err := client.QueryContext(ctx, `
		SELECT (-amount)::double precision
		FROM resource_supply_ledger
		WHERE owner_user_id = $1
			AND idempotency_key = $2
			AND ledger_type = $3
	`, userID, idempotencyKey, service.ResourceSupplyLedgerTypeTransfer)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	err = rows.Scan(&amount)
	if err != nil {
		return nil, false, err
	}
	balance, err := queryUserBalance(ctx, client, userID)
	if err != nil {
		return nil, false, err
	}
	return &service.ResourceSupplyTransferResult{Amount: amount, NewBalance: balance}, true, nil
}

func normalizeRepoPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullableFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}
