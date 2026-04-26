# Sub2API Migration: Import Strategy

This document defines the mapping from detected source modes to the standardized Docker Compose production layout.

## Target Layout
All migrations target the `docker-compose.local.yml` layout:
- `./.env`: Environment variables and secrets.
- `./data/`: Application runtime data (config, logs, etc.).
- `./postgres_data/`: PostgreSQL data directory.
- `./redis_data/`: Redis data directory.

---

## 1. Source: `binary` (Systemd)

**Recovery:**
- Secrets: Extract from `/etc/systemd/system/sub2api.service` (`Environment=` lines).
- Config: `/etc/sub2api/config.yaml`.
- DB: PostgreSQL dump from the host database.
- Data: `/opt/sub2api/data/`.

**Import Path:**
1. Populate `.env` with extracted secrets (`POSTGRES_PASSWORD`, `JWT_SECRET`, etc.).
2. Copy `/opt/sub2api/data/*` to `./data/`.
3. Start the `postgres` service only.
4. Restore the DB dump into the `postgres` container.
5. Start the full Compose stack.

---

## 2. Source: `compose-local` (Existing Local Directory)

**Recovery:**
- Secrets: `.env` file.
- Config: `data/config.yaml`.
- DB: `postgres_data/` directory and optional dump.
- Data: `data/` directory.

**Import Path:**
1. This is the "Native" mode. Copy `.env`, `data/`, and `postgres_data/` directly to the new workspace.
2. Verify ownership and permissions of the copied directories.
3. Start the full Compose stack.

---

## 3. Source: `compose-volume` (Named Volumes)

**Recovery:**
- Secrets: `.env` file.
- Config: Captured from inside the `sub2api` container or `data/` volume.
- DB: Dump via `docker exec postgres pg_dump`.
- Data: Backup volume via a helper container (e.g., `busybox` mount).

**Import Path:**
1. Populate `.env` with captured secrets.
2. Extract the data volume backup into `./data/`.
3. Start the `postgres` service only.
4. Restore the DB dump into the `postgres` container (which maps to `./postgres_data`).
5. Start the full Compose stack.

---

## Critical Continuity Rules

| Data Point | Failure Impact | Mitigation |
|------------|----------------|------------|
| `JWT_SECRET` | All user sessions invalidated; all existing tokens fail. | MUST recover and carry forward to `.env`. |
| `TOTP_ENCRYPTION_KEY` | All 2FA (TOTP) setups break; users locked out. | MUST recover and carry forward to `.env`. |
| `POSTGRES_PASSWORD` | Service cannot connect to existing data. | Use existing password or update DB and `.env` simultaneously. |
| `data/config.yaml` | Application settings (ports, features) reset. | Prefer using the recovered file. |
| `data/.installed` | Auto-setup might try to recreate admin user. | Ensure this file exists in the target `./data/`. |
