#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/preflight_checks.sh --deploy-dir DIR [--bundle PATH] [--backup-dir DIR]
EOF
}

DEPLOY_DIR=""
BUNDLE_PATH=""
BACKUP_DIR=""
SKIP_DOCKER="${SKIP_DOCKER:-false}"
MIN_DISK_MB="${MIN_DISK_MB:-2048}"

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
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
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

if [[ -z "${DEPLOY_DIR}" ]]; then
    usage >&2
    exit 1
fi

# 1. Check required files
required_paths=(
    "${DEPLOY_DIR}/docker-compose.local.yml"
    "${DEPLOY_DIR}/docker-compose.private.yml"
    "${DEPLOY_DIR}/.env"
)

for path in "${required_paths[@]}"; do
    [[ -f "${path}" ]] || { echo "Error: Missing required file: ${path}" >&2; exit 1; }
done

# 2. Check .env completeness
mandatory_keys=(
    "SUB2API_IMAGE"
    "JWT_SECRET"
    "POSTGRES_PASSWORD"
    "REDIS_PASSWORD"
)
for key in "${mandatory_keys[@]}"; do
    grep -q "^${key}=" "${DEPLOY_DIR}/.env" || { echo "Error: Missing mandatory key ${key} in .env" >&2; exit 1; }
done

# 3. Check for backup artifacts if backup-dir is provided
if [[ -n "${BACKUP_DIR}" ]]; then
    if [[ ! -d "${BACKUP_DIR}" ]]; then
        echo "Error: Backup directory ${BACKUP_DIR} does not exist." >&2
        exit 1
    fi
    # Look for at least one .tar.gz backup
    if ! ls "${BACKUP_DIR}"/backup_*.tar.gz >/dev/null 2>&1; then
        echo "Error: No backup artifacts (backup_*.tar.gz) found in ${BACKUP_DIR}." >&2
        exit 1
    fi
    echo "Backup artifacts found in ${BACKUP_DIR}."
fi

# 4. Check disk headroom
mkdir -p "${DEPLOY_DIR}/data" "${DEPLOY_DIR}/postgres_data" "${DEPLOY_DIR}/redis_data"

available_kb="$(df -Pk "${DEPLOY_DIR}" | awk 'NR==2 {print $4}')"
available_mb="$(( available_kb / 1024 ))"
if (( available_mb < MIN_DISK_MB )); then
    echo "Error: Insufficient free space: ${available_mb}MB < ${MIN_DISK_MB}MB" >&2
    exit 1
fi

# 5. Check bundle
if [[ -n "${BUNDLE_PATH}" ]]; then
    if [[ ! -f "${BUNDLE_PATH}" ]]; then
        echo "Error: Bundle file does not exist: ${BUNDLE_PATH}" >&2
        exit 1
    fi
    # Check if it's a valid tarball
    tar -tzf "${BUNDLE_PATH}" >/dev/null 2>&1 || { echo "Error: Bundle ${BUNDLE_PATH} is not a valid gzip tarball." >&2; exit 1; }
fi

# 6. Check Docker/Compose
if [[ "${SKIP_DOCKER}" != "true" ]]; then
    command -v docker >/dev/null 2>&1 || { echo "Error: docker is required" >&2; exit 1; }
    docker compose -f "${DEPLOY_DIR}/docker-compose.local.yml" -f "${DEPLOY_DIR}/docker-compose.private.yml" config >/dev/null || { echo "Error: Invalid Docker Compose configuration." >&2; exit 1; }
fi

printf 'Preflight checks passed for %s\n' "${DEPLOY_DIR}"
