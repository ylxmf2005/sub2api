-- Add settlement pool billing configuration, participants and cycle snapshots.

CREATE TABLE IF NOT EXISTS settlement_pool_configs (
    group_id BIGINT PRIMARY KEY REFERENCES groups(id) ON DELETE CASCADE,
    base_ratio DECIMAL(10, 6) NOT NULL DEFAULT 0.2,
    market_cap DECIMAL(20, 10) NOT NULL DEFAULT 0.35,
    tiers JSONB NOT NULL DEFAULT '[{"up_to":300,"weight":1},{"up_to":1200,"weight":0.8},{"up_to":null,"weight":0.64}]'::jsonb,
    active_cycle_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT settlement_pool_configs_base_ratio_range CHECK (base_ratio >= 0 AND base_ratio <= 1),
    CONSTRAINT settlement_pool_configs_market_cap_nonnegative CHECK (market_cap >= 0)
);

CREATE TABLE IF NOT EXISTS settlement_pool_cycles (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    total_cost DECIMAL(20, 10) NOT NULL DEFAULT 0,
    base_ratio DECIMAL(10, 6) NOT NULL DEFAULT 0.2,
    market_cap DECIMAL(20, 10) NOT NULL DEFAULT 0.35,
    tiers JSONB NOT NULL DEFAULT '[{"up_to":300,"weight":1},{"up_to":1200,"weight":0.8},{"up_to":null,"weight":0.64}]'::jsonb,
    snapshot JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT settlement_pool_cycles_status_check CHECK (status IN ('active', 'locked')),
    CONSTRAINT settlement_pool_cycles_nonnegative_cost CHECK (total_cost >= 0),
    CONSTRAINT settlement_pool_cycles_base_ratio_range CHECK (base_ratio >= 0 AND base_ratio <= 1),
    CONSTRAINT settlement_pool_cycles_market_cap_nonnegative CHECK (market_cap >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_settlement_pool_cycles_one_active
    ON settlement_pool_cycles (group_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_settlement_pool_cycles_group_started
    ON settlement_pool_cycles (group_id, started_at DESC);

CREATE TABLE IF NOT EXISTS settlement_pool_participants (
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_settlement_pool_participants_user
    ON settlement_pool_participants (user_id, group_id);

CREATE INDEX IF NOT EXISTS idx_usage_logs_settlement_pool_window
    ON usage_logs (group_id, user_id, billing_type, created_at)
    WHERE group_id IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'settlement_pool_configs_active_cycle_fk'
    ) THEN
        ALTER TABLE settlement_pool_configs
            ADD CONSTRAINT settlement_pool_configs_active_cycle_fk
            FOREIGN KEY (active_cycle_id) REFERENCES settlement_pool_cycles(id) ON DELETE SET NULL;
    END IF;
END $$;
