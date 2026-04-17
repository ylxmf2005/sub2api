#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/preflight_checks.sh --deploy-dir DIR [--bundle PATH]
EOF
}

DEPLOY_DIR=""
BUNDLE_PATH=""
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

required_paths=(
    "${DEPLOY_DIR}/docker-compose.local.yml"
    "${DEPLOY_DIR}/docker-compose.private.yml"
    "${DEPLOY_DIR}/.env"
)

for path in "${required_paths[@]}"; do
    [[ -f "${path}" ]] || { echo "Missing required file: ${path}" >&2; exit 1; }
done

mkdir -p "${DEPLOY_DIR}/data" "${DEPLOY_DIR}/postgres_data" "${DEPLOY_DIR}/redis_data"

available_kb="$(df -Pk "${DEPLOY_DIR}" | awk 'NR==2 {print $4}')"
available_mb="$(( available_kb / 1024 ))"
if (( available_mb < MIN_DISK_MB )); then
    echo "Insufficient free space: ${available_mb}MB < ${MIN_DISK_MB}MB" >&2
    exit 1
fi

if [[ -n "${BUNDLE_PATH}" && ! -e "${BUNDLE_PATH}" ]]; then
    echo "Bundle path does not exist: ${BUNDLE_PATH}" >&2
    exit 1
fi

if [[ "${SKIP_DOCKER}" != "true" ]]; then
    command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
    docker compose -f "${DEPLOY_DIR}/docker-compose.local.yml" -f "${DEPLOY_DIR}/docker-compose.private.yml" config >/dev/null
fi

printf 'Preflight checks passed for %s\n' "${DEPLOY_DIR}"
