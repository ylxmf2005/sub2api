-- Add ccgo reverse workstation control-plane tables.

CREATE TABLE IF NOT EXISTS ccgo_workspaces (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_slug VARCHAR(64) NOT NULL UNIQUE,
    server_root VARCHAR(512) NOT NULL UNIQUE,
    local_root_hash VARCHAR(64) NOT NULL,
    local_root_display TEXT NOT NULL,
    local_root_redacted TEXT NOT NULL DEFAULT '',
    os VARCHAR(32) NOT NULL,
    path_style VARCHAR(32) NOT NULL,
    device_id VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_seen_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ccgo_workspaces_os_check CHECK (os IN ('darwin', 'linux', 'windows')),
    CONSTRAINT ccgo_workspaces_path_style_check CHECK (path_style IN ('posix', 'windows')),
    CONSTRAINT ccgo_workspaces_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT ccgo_workspaces_user_local_root_unique UNIQUE (user_id, local_root_hash)
);

CREATE INDEX IF NOT EXISTS idx_ccgo_workspaces_user_device
    ON ccgo_workspaces (user_id, device_id);

CREATE INDEX IF NOT EXISTS idx_ccgo_workspaces_status
    ON ccgo_workspaces (status);

CREATE TABLE IF NOT EXISTS ccgo_agent_credentials (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES ccgo_workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    nonce_hash VARCHAR(64) NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ccgo_agent_credentials_status_check CHECK (status IN ('active', 'used', 'revoked', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_ccgo_agent_credentials_workspace_status
    ON ccgo_agent_credentials (workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_ccgo_agent_credentials_user_status
    ON ccgo_agent_credentials (user_id, status);

CREATE INDEX IF NOT EXISTS idx_ccgo_agent_credentials_expires_at
    ON ccgo_agent_credentials (expires_at);

CREATE TABLE IF NOT EXISTS ccgo_workstation_runs (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES ccgo_workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    run_id VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'starting',
    server_pid VARCHAR(64) NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NULL,
    stopped_at TIMESTAMPTZ NULL,
    stop_reason TEXT NOT NULL DEFAULT '',
    last_heartbeat_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ccgo_workstation_runs_status_check CHECK (status IN ('starting', 'running', 'stopping', 'stopped', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_ccgo_workstation_runs_workspace_status
    ON ccgo_workstation_runs (workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_ccgo_workstation_runs_user_status
    ON ccgo_workstation_runs (user_id, status);

CREATE INDEX IF NOT EXISTS idx_ccgo_workstation_runs_last_heartbeat
    ON ccgo_workstation_runs (last_heartbeat_at);

CREATE TABLE IF NOT EXISTS ccgo_command_audits (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES ccgo_workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    run_id BIGINT NULL REFERENCES ccgo_workstation_runs(id) ON DELETE SET NULL,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    command_hash VARCHAR(64) NOT NULL,
    redacted_command TEXT NOT NULL DEFAULT '',
    server_cwd TEXT NOT NULL,
    local_cwd TEXT NOT NULL,
    exit_code INTEGER NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'requested',
    failure_reason TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ccgo_command_audits_workspace_created
    ON ccgo_command_audits (workspace_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ccgo_command_audits_user_created
    ON ccgo_command_audits (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ccgo_command_audits_status_created
    ON ccgo_command_audits (status, created_at DESC);

CREATE TABLE IF NOT EXISTS ccgo_device_logins (
    id BIGSERIAL PRIMARY KEY,
    device_code_hash VARCHAR(64) NOT NULL UNIQUE,
    user_code_hash VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ NULL,
    consumed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ccgo_device_logins_status_check CHECK (status IN ('pending', 'approved', 'consumed', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_ccgo_device_logins_status_expires
    ON ccgo_device_logins (status, expires_at);

CREATE INDEX IF NOT EXISTS idx_ccgo_device_logins_user_status
    ON ccgo_device_logins (user_id, status)
    WHERE user_id IS NOT NULL;

COMMENT ON TABLE ccgo_workspaces IS 'Stable ccgo local-root to server-root mappings';
COMMENT ON TABLE ccgo_agent_credentials IS 'Short-lived hashed credentials for ccgo local agents';
COMMENT ON TABLE ccgo_workstation_runs IS 'Disposable Claude Code process lifecycle records';
COMMENT ON TABLE ccgo_command_audits IS 'Redacted ccgo local command execution metadata';
COMMENT ON TABLE ccgo_device_logins IS 'Pending ccgo CLI device-code login attempts';
