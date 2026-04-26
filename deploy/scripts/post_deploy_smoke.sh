#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/post_deploy_smoke.sh --base-url URL [--admin-email EMAIL --admin-password PASSWORD]
EOF
}

BASE_URL=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-8}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base-url)
            BASE_URL="$2"
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

if [[ -z "${BASE_URL}" ]]; then
    usage >&2
    exit 1
fi

health_code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time "${TIMEOUT_SECONDS}" "${BASE_URL%/}/health")"
if [[ "${health_code}" != "200" ]]; then
    echo "Health check failed: HTTP ${health_code}" >&2
    exit 1
fi

printf 'Health check OK: %s/health\n' "${BASE_URL%/}"

if [[ -n "${ADMIN_EMAIL}" && -n "${ADMIN_PASSWORD}" ]]; then
    login_code="$(curl -sS -o /tmp/sub2api-login-response.json -w '%{http_code}' \
        --max-time "${TIMEOUT_SECONDS}" \
        -H 'Content-Type: application/json' \
        -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
        "${BASE_URL%/}/api/v1/auth/login")"
    if [[ "${login_code}" != "200" ]]; then
        echo "Admin login check failed: HTTP ${login_code}" >&2
        cat /tmp/sub2api-login-response.json >&2 || true
        exit 1
    fi
    printf 'Admin login check OK: %s\n' "${ADMIN_EMAIL}"
fi
