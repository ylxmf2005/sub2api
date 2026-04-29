-- Split settlement pool long-term eligibility from per-cycle participation.

DO $$
BEGIN
    IF to_regclass('public.settlement_pool_candidates') IS NULL
       AND to_regclass('public.settlement_pool_participants') IS NOT NULL THEN
        ALTER TABLE settlement_pool_participants RENAME TO settlement_pool_candidates;
    END IF;
END $$;

DO $$
BEGIN
    IF to_regclass('public.settlement_pool_participants_pkey') IS NOT NULL
       AND to_regclass('public.settlement_pool_candidates_pkey') IS NULL THEN
        ALTER INDEX settlement_pool_participants_pkey RENAME TO settlement_pool_candidates_pkey;
    END IF;

    IF to_regclass('public.idx_settlement_pool_participants_user') IS NOT NULL
       AND to_regclass('public.idx_settlement_pool_candidates_user') IS NULL THEN
        ALTER INDEX idx_settlement_pool_participants_user RENAME TO idx_settlement_pool_candidates_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'settlement_pool_participants_group_id_fkey'
    ) THEN
        ALTER TABLE settlement_pool_candidates
            RENAME CONSTRAINT settlement_pool_participants_group_id_fkey TO settlement_pool_candidates_group_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'settlement_pool_participants_user_id_fkey'
    ) THEN
        ALTER TABLE settlement_pool_candidates
            RENAME CONSTRAINT settlement_pool_participants_user_id_fkey TO settlement_pool_candidates_user_id_fkey;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS settlement_pool_cycle_participants (
    cycle_id BIGINT NOT NULL REFERENCES settlement_pool_cycles(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cycle_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_settlement_pool_cycle_participants_user
    ON settlement_pool_cycle_participants (user_id, cycle_id);

INSERT INTO settlement_pool_cycle_participants (cycle_id, user_id, joined_at)
SELECT c.id, cand.user_id, cand.created_at
FROM settlement_pool_candidates cand
JOIN settlement_pool_cycles c
    ON c.group_id = cand.group_id
   AND c.status = 'active'
ON CONFLICT (cycle_id, user_id) DO NOTHING;
