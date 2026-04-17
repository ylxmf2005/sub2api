#!/usr/bin/env bash

set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
    echo "SKIP: docker not installed"
    exit 0
fi

temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

docker image inspect busybox:latest >/dev/null 2>&1 || docker pull busybox:latest >/dev/null

./deploy/scripts/export_release_bundle.sh \
    --image busybox:latest \
    --release smoke-bundle \
    --output-dir "${temp_dir}" >/dev/null

test -f "${temp_dir}/smoke-bundle/image.tar"
test -f "${temp_dir}/smoke-bundle/manifest.env"
test -f "${temp_dir}/smoke-bundle.tar.gz"

./deploy/scripts/load_release_bundle.sh \
    --bundle "${temp_dir}/smoke-bundle.tar.gz" >/dev/null

echo "release bundle smoke passed"
