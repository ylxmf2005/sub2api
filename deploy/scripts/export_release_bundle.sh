#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/export_release_bundle.sh --image IMAGE_REF --release RELEASE_NAME [--output-dir DIR]
EOF
}

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-${REPO_ROOT}/release}"
IMAGE_REF=""
RELEASE_NAME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --image)
            IMAGE_REF="$2"
            shift 2
            ;;
        --release)
            RELEASE_NAME="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
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

if [[ -z "${IMAGE_REF}" || -z "${RELEASE_NAME}" ]]; then
    usage >&2
    exit 1
fi

docker image inspect "${IMAGE_REF}" >/dev/null

mkdir -p "${OUTPUT_DIR}"
BUNDLE_DIR="${OUTPUT_DIR}/${RELEASE_NAME}"
rm -rf "${BUNDLE_DIR}"
mkdir -p "${BUNDLE_DIR}/compose"

IMAGE_ARCHIVE="${BUNDLE_DIR}/image.tar"
docker save -o "${IMAGE_ARCHIVE}" "${IMAGE_REF}"
sha256_file "${IMAGE_ARCHIVE}" > "${BUNDLE_DIR}/image.tar.sha256"

cp "${REPO_ROOT}/deploy/docker-compose.local.yml" "${BUNDLE_DIR}/compose/"
cp "${REPO_ROOT}/deploy/docker-compose.private.yml" "${BUNDLE_DIR}/compose/"
cp "${REPO_ROOT}/deploy/private/.env.production.example" "${BUNDLE_DIR}/compose/"
cp "${REPO_ROOT}/deploy/scripts/load_release_bundle.sh" "${BUNDLE_DIR}/compose/"

GIT_COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
GIT_BRANCH="$(git -C "${REPO_ROOT}" branch --show-current)"
CREATED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
IMAGE_PLATFORM="$(docker image inspect "${IMAGE_REF}" --format '{{.Os}}/{{.Architecture}}')"

cat > "${BUNDLE_DIR}/manifest.env" <<EOF
RELEASE_NAME=${RELEASE_NAME}
IMAGE_REF=${IMAGE_REF}
IMAGE_PLATFORM=${IMAGE_PLATFORM}
CREATED_AT_UTC=${CREATED_AT}
GIT_COMMIT=${GIT_COMMIT}
GIT_BRANCH=${GIT_BRANCH}
COMPOSE_BASE=docker-compose.local.yml
COMPOSE_OVERRIDE=docker-compose.private.yml
ENV_TEMPLATE=.env.production.example
EOF

tar -C "${OUTPUT_DIR}" -czf "${OUTPUT_DIR}/${RELEASE_NAME}.tar.gz" "${RELEASE_NAME}"
sha256_file "${OUTPUT_DIR}/${RELEASE_NAME}.tar.gz" > "${OUTPUT_DIR}/${RELEASE_NAME}.tar.gz.sha256"

printf 'Bundle directory: %s\n' "${BUNDLE_DIR}"
printf 'Bundle archive: %s\n' "${OUTPUT_DIR}/${RELEASE_NAME}.tar.gz"
