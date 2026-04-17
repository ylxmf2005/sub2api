CREATE OR REPLACE FUNCTION temp_table_count(tab text)
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
  result bigint;
BEGIN
  EXECUTE format('SELECT count(*) FROM %I', tab) INTO result;
  RETURN result;
EXCEPTION
  WHEN undefined_table THEN
    RETURN NULL;
END;
$$;

SELECT
  tab AS table_name,
  temp_table_count(tab) AS row_count
FROM (
  VALUES
    ('users'),
    ('api_keys'),
    ('accounts'),
    ('groups'),
    ('settings'),
    ('usage_logs'),
    ('subscription_plans'),
    ('user_subscriptions'),
    ('payment_orders'),
    ('redeem_codes')
) AS t(tab);

SELECT
  COUNT(*) FILTER (WHERE role = 'admin') AS admin_users,
  COUNT(*) AS total_users
FROM users;

SELECT
  COUNT(*) AS total_accounts,
  COUNT(*) FILTER (WHERE status = 'active') AS active_accounts
FROM accounts;

SELECT
  MAX(created_at) AS latest_usage_log_at
FROM usage_logs;

DROP FUNCTION temp_table_count(text);
