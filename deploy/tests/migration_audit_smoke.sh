#!/usr/bin/env bash

set -euo pipefail

temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

binary_root="${temp_dir}/binary"
mkdir -p "${binary_root}/opt/sub2api" "${binary_root}/etc/sub2api"
touch "${binary_root}/opt/sub2api/sub2api" "${binary_root}/etc/sub2api/config.yaml"

output="$(./deploy/migration/audit_server.sh --scan-root "${binary_root}")"
grep -q '^SOURCE_MODE=binary$' <<< "${output}"

docker_local_root="${temp_dir}/docker-local"
mkdir -p "${docker_local_root}/srv/sub2api"
cat > "${docker_local_root}/srv/sub2api/docker-compose.local.yml" <<'EOF'
services:
  sub2api:
    volumes:
      - ./data:/app/data
EOF

output="$(./deploy/migration/audit_server.sh --scan-root "${docker_local_root}")"
grep -q '^SOURCE_MODE=docker-local$' <<< "${output}"

mixed_root="${temp_dir}/mixed"
mkdir -p "${mixed_root}/opt/sub2api" "${mixed_root}/etc/sub2api" "${mixed_root}/srv/sub2api"
touch "${mixed_root}/opt/sub2api/sub2api" "${mixed_root}/etc/sub2api/config.yaml"
cat > "${mixed_root}/srv/sub2api/docker-compose.local.yml" <<'EOF'
services:
  sub2api:
    volumes:
      - ./data:/app/data
EOF

output="$(./deploy/migration/audit_server.sh --scan-root "${mixed_root}")"
grep -q '^SOURCE_MODE=mixed$' <<< "${output}"

echo "migration audit smoke passed"
