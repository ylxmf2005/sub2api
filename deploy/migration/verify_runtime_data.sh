#!/usr/bin/env bash

set -euo pipefail

DEPLOY_DIR="${1:-.}"

check_path() {
    local path="$1"
    if [[ -e "${path}" ]]; then
        printf 'OK %s\n' "${path}"
    else
        printf 'MISSING %s\n' "${path}"
    fi
}

check_path "${DEPLOY_DIR}/.env"
check_path "${DEPLOY_DIR}/docker-compose.local.yml"
check_path "${DEPLOY_DIR}/docker-compose.private.yml"
check_path "${DEPLOY_DIR}/data"
check_path "${DEPLOY_DIR}/postgres_data"
check_path "${DEPLOY_DIR}/redis_data"
check_path "${DEPLOY_DIR}/data/config.yaml"
check_path "${DEPLOY_DIR}/data/.installed"

for dir in data postgres_data redis_data; do
    if [[ -d "${DEPLOY_DIR}/${dir}" ]]; then
        du -sh "${DEPLOY_DIR}/${dir}" 2>/dev/null || true
    fi
done
