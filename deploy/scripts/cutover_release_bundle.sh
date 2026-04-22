#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/cutover_release_bundle.sh --deploy-dir DIR --bundle PATH --base-url URL [options]

Options:
  --deploy-dir DIR         Deployment directory containing compose files and .env
  --bundle PATH            Release bundle tar.gz created by export_release_bundle.sh
  --base-url URL           Base URL used for post-deploy health checks
  --backup-dir DIR         Optional directory that must already contain backup_*.tar.gz
  --admin-email EMAIL      Optional admin email for login smoke check
  --admin-password PASS    Optional admin password for login smoke check
  --timeout-seconds N      Health check wait timeout in seconds (default: 180)
  --load-work-dir DIR      Temporary extraction dir for bundle loading
  --dry-run                Validate inputs and show the rollout plan without mutating runtime
  --no-rollback            Disable automatic rollback on failed restart/smoke
  -h, --help               Show this help
EOF
}

DEPLOY_DIR=""
BUNDLE_PATH=""
BASE_URL=""
BACKUP_DIR=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
LOAD_WORK_DIR=""
DRY_RUN=false
ROLLBACK_ON_FAILURE=true

while [[ $# -gt 0 ]]; do
    case "$1" in
        --deploy-dir)
            DEPLOY_DIR="$2"
            shift 2
            ;;
        --bundle)
            BUNDLE_PATH="$2"
            shift 2
            ;;
        --base-url)
            BASE_URL="$2"
            shift 2
            ;;
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        --admin-email)
            ADMIN_EMAIL="$2"
            shift 2
            ;;
        --admin-password)
            ADMIN_PASSWORD="$2"
            shift 2
            ;;
        --timeout-seconds)
            TIMEOUT_SECONDS="$2"
            shift 2
            ;;
        --load-work-dir)
            LOAD_WORK_DIR="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --no-rollback)
            ROLLBACK_ON_FAILURE=false
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [[ -z "${DEPLOY_DIR}" || -z "${BUNDLE_PATH}" || -z "${BASE_URL}" ]]; then
    usage >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRECHECK_SCRIPT="${SCRIPT_DIR}/preflight_checks.sh"
LOAD_SCRIPT="${SCRIPT_DIR}/load_release_bundle.sh"
SMOKE_SCRIPT="${SCRIPT_DIR}/post_deploy_smoke.sh"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_LOCAL="${DEPLOY_DIR}/docker-compose.local.yml"
COMPOSE_PRIVATE="${DEPLOY_DIR}/docker-compose.private.yml"
SERVICE_NAME="sub2api"
TIMESTAMP="$(date -u +%Y%m%d%H%M%S)"

if [[ -z "${LOAD_WORK_DIR}" ]]; then
    LOAD_WORK_DIR="${DEPLOY_DIR}/.tmp-load-${TIMESTAMP}"
fi

compose() {
    env -u SUB2API_IMAGE -u SUB2API_RELEASE_TAG docker compose \
        --env-file "${ENV_FILE}" \
        -f "${COMPOSE_LOCAL}" \
        -f "${COMPOSE_PRIVATE}" \
        "$@"
}

read_manifest_field() {
    local field="$1"
    python3 - "${BUNDLE_PATH}" "${field}" <<'PY'
import json
import sys
import tarfile

bundle_path = sys.argv[1]
field = sys.argv[2]

with tarfile.open(bundle_path, "r:gz") as tar:
    manifest_member = next((m for m in tar.getmembers() if m.name.endswith("/manifest.json")), None)
    if manifest_member is None:
        raise SystemExit("manifest.json not found in bundle")
    data = json.load(tar.extractfile(manifest_member))
    value = data.get(field, "")
    if value is None:
        value = ""
    print(value)
PY
}

read_env_value() {
    local key="$1"
    python3 - "${ENV_FILE}" "${key}" <<'PY'
import sys
from pathlib import Path

env_path = Path(sys.argv[1])
target = sys.argv[2]

for line in env_path.read_text().splitlines():
    if not line or line.lstrip().startswith("#") or "=" not in line:
        continue
    key, value = line.split("=", 1)
    if key == target:
        print(value)
        break
PY
}

write_env_value() {
    local key="$1"
    local value="$2"
    python3 - "${ENV_FILE}" "${key}" "${value}" <<'PY'
import sys
from pathlib import Path

env_path = Path(sys.argv[1])
target = sys.argv[2]
new_value = sys.argv[3]

lines = env_path.read_text().splitlines()
updated = []
replaced = False

for line in lines:
    if line.startswith(f"{target}="):
        updated.append(f"{target}={new_value}")
        replaced = True
    else:
        updated.append(line)

if not replaced:
    if updated and updated[-1] != "":
        updated.append("")
    updated.append(f"{target}={new_value}")

env_path.write_text("\n".join(updated) + "\n")
PY
}

rollback() {
    local rollback_image="$1"
    local rollback_tag="$2"
    local env_backup="$3"

    echo "Rollback: restoring ${rollback_image} (${rollback_tag})"
    cp "${env_backup}" "${ENV_FILE}"
    write_env_value SUB2API_IMAGE "${rollback_image}"
    if [[ -n "${rollback_tag}" && "${rollback_tag}" != "<unset>" ]]; then
        write_env_value SUB2API_RELEASE_TAG "${rollback_tag}"
    fi
    compose up -d --no-deps --force-recreate "${SERVICE_NAME}"
}

wait_for_smoke() {
    local deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))
    while true; do
        if [[ -n "${ADMIN_EMAIL}" && -n "${ADMIN_PASSWORD}" ]]; then
            if "${SMOKE_SCRIPT}" --base-url "${BASE_URL}" --admin-email "${ADMIN_EMAIL}" --admin-password "${ADMIN_PASSWORD}"; then
                return 0
            fi
        else
            if "${SMOKE_SCRIPT}" --base-url "${BASE_URL}"; then
                return 0
            fi
        fi

        if (( $(date +%s) >= deadline )); then
            return 1
        fi

        sleep 3
    done
}

if [[ -n "${BACKUP_DIR}" ]]; then
    "${PRECHECK_SCRIPT}" \
        --deploy-dir "${DEPLOY_DIR}" \
        --bundle "${BUNDLE_PATH}" \
        --backup-dir "${BACKUP_DIR}"
else
    "${PRECHECK_SCRIPT}" \
        --deploy-dir "${DEPLOY_DIR}" \
        --bundle "${BUNDLE_PATH}"
fi

NEW_IMAGE_REF="$(read_manifest_field image_ref)"
NEW_IMAGE_TAG="$(read_manifest_field image_tag)"
NEW_IMAGE_NAME="$(read_manifest_field image_name)"
OLD_IMAGE_REF="$(read_env_value SUB2API_IMAGE)"
OLD_RELEASE_TAG="$(read_env_value SUB2API_RELEASE_TAG)"

if [[ -z "${NEW_IMAGE_REF}" || -z "${NEW_IMAGE_TAG}" || -z "${NEW_IMAGE_NAME}" ]]; then
    echo "Error: Bundle manifest is missing image metadata." >&2
    exit 1
fi

if [[ -z "${OLD_IMAGE_REF}" ]]; then
    echo "Error: SUB2API_IMAGE is missing from ${ENV_FILE}." >&2
    exit 1
fi

if ! docker image inspect "${OLD_IMAGE_REF}" >/dev/null 2>&1; then
    echo "Error: Current rollback image ${OLD_IMAGE_REF} is not present locally." >&2
    exit 1
fi

OLD_IMAGE_REPO="${OLD_IMAGE_REF%:*}"
if [[ "${OLD_IMAGE_REPO}" == "${OLD_IMAGE_REF}" ]]; then
    OLD_IMAGE_REPO="${OLD_IMAGE_REF}"
fi
ROLLBACK_IMAGE_REF="${OLD_IMAGE_REPO}:rollback-${TIMESTAMP}"

echo "Current image: ${OLD_IMAGE_REF}"
echo "Current tag: ${OLD_RELEASE_TAG:-<unset>}"
echo "Bundle image: ${NEW_IMAGE_REF}"
echo "Bundle tag: ${NEW_IMAGE_TAG}"
echo "Rollback snapshot tag: ${ROLLBACK_IMAGE_REF}"

if [[ "${DRY_RUN}" == "true" ]]; then
    echo "Dry run only: preflight passed and bundle metadata parsed successfully."
    exit 0
fi

ENV_BACKUP="${DEPLOY_DIR}/.env.bak.${TIMESTAMP}"
cp "${ENV_FILE}" "${ENV_BACKUP}"
echo "Saved env backup to ${ENV_BACKUP}"

echo "Snapshotting current image ${OLD_IMAGE_REF} to ${ROLLBACK_IMAGE_REF}"
docker tag "${OLD_IMAGE_REF}" "${ROLLBACK_IMAGE_REF}"

"${LOAD_SCRIPT}" --work-dir "${LOAD_WORK_DIR}" "${BUNDLE_PATH}"

if ! docker image inspect "${NEW_IMAGE_REF}" >/dev/null 2>&1; then
    echo "Error: Loaded image ${NEW_IMAGE_REF} is not present locally after docker load." >&2
    exit 1
fi

write_env_value SUB2API_IMAGE "${NEW_IMAGE_REF}"
write_env_value SUB2API_RELEASE_TAG "${NEW_IMAGE_TAG}"

echo "Restarting ${SERVICE_NAME} with ${NEW_IMAGE_REF}"
if ! compose up -d --no-deps --force-recreate "${SERVICE_NAME}"; then
    if [[ "${ROLLBACK_ON_FAILURE}" == "true" ]]; then
        rollback "${ROLLBACK_IMAGE_REF}" "${OLD_RELEASE_TAG:-<unset>}" "${ENV_BACKUP}"
    fi
    exit 1
fi

if ! wait_for_smoke; then
    echo "Error: Post-deploy smoke check failed for ${NEW_IMAGE_REF}" >&2
    compose ps
    if [[ "${ROLLBACK_ON_FAILURE}" == "true" ]]; then
        rollback "${ROLLBACK_IMAGE_REF}" "${OLD_RELEASE_TAG:-<unset>}" "${ENV_BACKUP}"
        if ! wait_for_smoke; then
            echo "Error: Rollback completed but smoke checks still failed." >&2
            exit 1
        fi
        echo "Rollback restored service health."
    fi
    exit 1
fi

echo "Cutover completed successfully with ${NEW_IMAGE_REF}"
