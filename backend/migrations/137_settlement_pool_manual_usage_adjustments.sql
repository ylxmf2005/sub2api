-- Add auditable manual usage adjustments for settlement pool cycles.

CREATE TABLE IF NOT EXISTS settlement_pool_manual_usage_adjustments (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    cycle_id BIGINT NOT NULL REFERENCES settlement_pool_cycles(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    usage_amount DECIMAL(20, 10) NOT NULL,
    reason TEXT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT settlement_pool_manual_usage_adjustments_usage_non_zero CHECK (usage_amount <> 0)
);

CREATE INDEX IF NOT EXISTS idx_settlement_pool_manual_usage_adjustments_cycle_user
    ON settlement_pool_manual_usage_adjustments (cycle_id, user_id);

CREATE INDEX IF NOT EXISTS idx_settlement_pool_manual_usage_adjustments_cycle_account
    ON settlement_pool_manual_usage_adjustments (cycle_id, account_id);

CREATE INDEX IF NOT EXISTS idx_settlement_pool_manual_usage_adjustments_group_created
    ON settlement_pool_manual_usage_adjustments (group_id, created_at DESC);
