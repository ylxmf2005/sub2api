-- Sub2API Migration Verification Script
-- Purpose: Verify that key data has been imported correctly into the new PostgreSQL database.

\echo '--- Sub2API Data Verification ---'

-- 1. Check User counts
SELECT
    (SELECT COUNT(*) FROM users) AS user_count,
    (SELECT COUNT(*) FROM users WHERE role = 'admin') AS admin_count;

-- 2. Check Core Entities
SELECT
    (SELECT COUNT(*) FROM accounts) AS account_count,
    (SELECT COUNT(*) FROM groups) AS group_count,
    (SELECT COUNT(*) FROM api_keys) AS apikey_count;

-- 3. Check for specific continuity indicators
-- (Example: presence of the default group or the first user)
SELECT id, email, role, status FROM users ORDER BY id ASC LIMIT 5;

-- 4. Check for recent activity/logs to ensure usage data survived
SELECT COUNT(*) AS usage_log_count FROM usage_logs;

-- 5. Verify join table normalization (per upstream notes)
-- users.allowed_groups -> user_allowed_groups
SELECT
    (SELECT COUNT(*) FROM user_allowed_groups) AS join_table_pairs;

\echo '--- Verification Complete ---'
