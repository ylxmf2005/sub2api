-- Add resource supply ownership, durable billing events, and supply-credit wallet tables.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS supply_rewards_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS supply_reward_multiplier DECIMAL(20,8) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS supply_self_service_review_policy VARCHAR(32) NOT NULL DEFAULT 'manual_review';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'groups_supply_reward_multiplier_nonnegative'
    ) THEN
        ALTER TABLE groups
            ADD CONSTRAINT groups_supply_reward_multiplier_nonnegative
            CHECK (supply_reward_multiplier >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'groups_supply_self_service_review_policy_check'
    ) THEN
        ALTER TABLE groups
            ADD CONSTRAINT groups_supply_self_service_review_policy_check
            CHECK (supply_self_service_review_policy IN ('manual_review', 'auto_online'));
    END IF;
END $$;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS supply_owner_user_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS supply_source VARCHAR(32) NULL,
    ADD COLUMN IF NOT EXISTS supply_status VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS supply_status_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS supply_submitted_by BIGINT NULL,
    ADD COLUMN IF NOT EXISTS supply_reviewed_by BIGINT NULL,
    ADD COLUMN IF NOT EXISTS supply_reviewed_at TIMESTAMPTZ NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_supply_owner_user_id_fkey'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_supply_owner_user_id_fkey
            FOREIGN KEY (supply_owner_user_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_supply_submitted_by_fkey'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_supply_submitted_by_fkey
            FOREIGN KEY (supply_submitted_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_supply_reviewed_by_fkey'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_supply_reviewed_by_fkey
            FOREIGN KEY (supply_reviewed_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_supply_source_check'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_supply_source_check
            CHECK (supply_source IS NULL OR supply_source IN ('admin', 'self_service'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'accounts_supply_status_check'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_supply_status_check
            CHECK (supply_status IN ('none', 'testing', 'pending_review', 'schedulable', 'paused', 'rejected', 'revoked'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_groups_supply_rewards_enabled
    ON groups (supply_rewards_enabled);

CREATE INDEX IF NOT EXISTS idx_accounts_supply_owner_user_id
    ON accounts (supply_owner_user_id)
    WHERE supply_owner_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_supply_status
    ON accounts (supply_status);

CREATE INDEX IF NOT EXISTS idx_accounts_supply_source
    ON accounts (supply_source)
    WHERE supply_source IS NOT NULL;

CREATE TABLE IF NOT EXISTS resource_supply_balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    available_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    lifetime_earned_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    lifetime_transferred_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT resource_supply_balances_available_nonnegative CHECK (available_amount >= 0),
    CONSTRAINT resource_supply_balances_lifetime_earned_nonnegative CHECK (lifetime_earned_amount >= 0),
    CONSTRAINT resource_supply_balances_lifetime_transferred_nonnegative CHECK (lifetime_transferred_amount >= 0)
);

CREATE TABLE IF NOT EXISTS usage_billing_events (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    request_fingerprint VARCHAR(128) NOT NULL,
    request_payload_hash VARCHAR(128) NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    account_type VARCHAR(32) NULL,
    model VARCHAR(255) NOT NULL DEFAULT '',
    service_tier VARCHAR(64) NOT NULL DEFAULT '',
    reasoning_effort VARCHAR(64) NOT NULL DEFAULT '',
    billing_type SMALLINT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_5m_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_1h_tokens INTEGER NOT NULL DEFAULT 0,
    total_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    actual_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    balance_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    subscription_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    api_key_quota_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    api_key_rate_limit_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    account_quota_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    supply_reward_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    supply_owner_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    supply_reward_multiplier DECIMAL(20,8) NOT NULL DEFAULT 1,
    supply_account_status VARCHAR(32) NOT NULL DEFAULT 'none',
    supply_source VARCHAR(32) NOT NULL DEFAULT 'live',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT usage_billing_events_unique_request UNIQUE (request_id, api_key_id),
    CONSTRAINT usage_billing_events_costs_nonnegative CHECK (
        total_cost >= 0
        AND actual_cost >= 0
        AND balance_cost >= 0
        AND subscription_cost >= 0
        AND api_key_quota_cost >= 0
        AND api_key_rate_limit_cost >= 0
        AND account_quota_cost >= 0
    ),
    CONSTRAINT usage_billing_events_supply_reward_multiplier_nonnegative CHECK (supply_reward_multiplier >= 0),
    CONSTRAINT usage_billing_events_supply_account_status_check CHECK (
        supply_account_status IN ('none', 'testing', 'pending_review', 'schedulable', 'paused', 'rejected', 'revoked')
    ),
    CONSTRAINT usage_billing_events_supply_source_check CHECK (supply_source IN ('live', 'backfill'))
);

CREATE INDEX IF NOT EXISTS idx_usage_billing_events_user_created
    ON usage_billing_events (user_id, created_at);

CREATE INDEX IF NOT EXISTS idx_usage_billing_events_group_user_created
    ON usage_billing_events (group_id, user_id, created_at)
    WHERE group_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_billing_events_billing_type_created
    ON usage_billing_events (billing_type, created_at);

CREATE INDEX IF NOT EXISTS idx_usage_billing_events_account_created
    ON usage_billing_events (account_id, created_at);

CREATE TABLE IF NOT EXISTS resource_supply_ledger (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    caller_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL,
    group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    usage_billing_event_id BIGINT NULL REFERENCES usage_billing_events(id) ON DELETE SET NULL,
    ledger_type VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(255) NULL,
    amount DECIMAL(20,8) NOT NULL,
    balance_after DECIMAL(20,8) NOT NULL,
    actual_cost DECIMAL(20,10) NULL,
    reward_multiplier DECIMAL(20,8) NULL,
    billing_type SMALLINT NULL,
    model VARCHAR(255) NULL,
    request_id VARCHAR(255) NULL,
    admin_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    note TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT resource_supply_ledger_type_check CHECK (ledger_type IN ('reward', 'transfer', 'adjustment')),
    CONSTRAINT resource_supply_ledger_balance_after_nonnegative CHECK (balance_after >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_resource_supply_ledger_reward_event
    ON resource_supply_ledger (usage_billing_event_id)
    WHERE ledger_type = 'reward' AND usage_billing_event_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_resource_supply_ledger_owner_idempotency
    ON resource_supply_ledger (owner_user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_resource_supply_ledger_owner_created
    ON resource_supply_ledger (owner_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_resource_supply_ledger_caller_created
    ON resource_supply_ledger (caller_user_id, created_at DESC)
    WHERE caller_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_resource_supply_ledger_group_created
    ON resource_supply_ledger (group_id, created_at DESC)
    WHERE group_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_resource_supply_ledger_account_created
    ON resource_supply_ledger (account_id, created_at DESC)
    WHERE account_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_resource_supply_ledger_request_id
    ON resource_supply_ledger (request_id)
    WHERE request_id IS NOT NULL;

COMMENT ON TABLE resource_supply_balances IS 'Resource supply credit balance by owner user';
COMMENT ON TABLE usage_billing_events IS 'Durable billable usage events used for idempotent economic side effects';
COMMENT ON TABLE resource_supply_ledger IS 'Immutable resource supply reward, transfer, and adjustment ledger';
