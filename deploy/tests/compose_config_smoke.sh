#!/usr/bin/env bash

set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
    echo "SKIP: docker not installed"
    exit 0
fi

temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

cat > "${temp_dir}/.env" <<'EOF'
POSTGRES_PASSWORD=test-postgres-password
SUB2API_IMAGE=local/sub2api-private:test
SUB2API_RELEASE_TAG=test
EOF

cp deploy/docker-compose.local.yml "${temp_dir}/docker-compose.local.yml"
cp deploy/docker-compose.private.yml "${temp_dir}/docker-compose.private.yml"

docker compose \
    --env-file "${temp_dir}/.env" \
    -f "${temp_dir}/docker-compose.local.yml" \
    -f "${temp_dir}/docker-compose.private.yml" \
    config >/dev/null

echo "compose config smoke passed"
