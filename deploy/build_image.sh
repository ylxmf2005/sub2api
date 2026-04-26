#!/usr/bin/env bash
# 本地构建私有镜像，避免把构建职责丢给生产服务器。

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/build_image.sh [options]

Options:
  --image-name NAME    Docker image name (default: local/sub2api-private)
  --image-tag TAG      Docker image tag (default: local)
  --app-version VALUE  Application version injected into binary (default: backend/cmd/server/VERSION)
  --platform VALUE     Target image platform (default: linux/amd64)
  --tag-latest         Also tag IMAGE_NAME:latest
  --goproxy URL        GOPROXY value
  --gosumdb VALUE      GOSUMDB value
  -h, --help           Show this help
EOF
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-local/sub2api-private}"
IMAGE_TAG="${IMAGE_TAG:-local}"
APP_VERSION="${APP_VERSION:-}"
PLATFORM="${PLATFORM:-linux/amd64}"
TAG_LATEST=false
GOPROXY_VALUE="${GOPROXY:-https://goproxy.cn,direct}"
GOSUMDB_VALUE="${GOSUMDB:-sum.golang.google.cn}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --image-name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        --image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --app-version)
            APP_VERSION="$2"
            shift 2
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --tag-latest)
            TAG_LATEST=true
            shift
            ;;
        --goproxy)
            GOPROXY_VALUE="$2"
            shift 2
            ;;
        --gosumdb)
            GOSUMDB_VALUE="$2"
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

IMAGE_REF="${IMAGE_NAME}:${IMAGE_TAG}"

if [[ -z "${APP_VERSION}" ]]; then
    APP_VERSION="$(tr -d '\r\n' < "${REPO_ROOT}/backend/cmd/server/VERSION")"
fi

# Get metadata for build-args
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if docker buildx version >/dev/null 2>&1; then
    docker buildx build \
        --load \
        --platform "${PLATFORM}" \
        -t "${IMAGE_REF}" \
        --build-arg GOPROXY="${GOPROXY_VALUE}" \
        --build-arg GOSUMDB="${GOSUMDB_VALUE}" \
        --build-arg VERSION="${APP_VERSION}" \
        --build-arg COMMIT="${COMMIT_HASH}" \
        --build-arg DATE="${BUILD_DATE}" \
        -f "${REPO_ROOT}/Dockerfile" \
        "${REPO_ROOT}"
else
    docker build \
        --platform "${PLATFORM}" \
        -t "${IMAGE_REF}" \
        --build-arg GOPROXY="${GOPROXY_VALUE}" \
        --build-arg GOSUMDB="${GOSUMDB_VALUE}" \
        --build-arg VERSION="${APP_VERSION}" \
        --build-arg COMMIT="${COMMIT_HASH}" \
        --build-arg DATE="${BUILD_DATE}" \
        -f "${REPO_ROOT}/Dockerfile" \
        "${REPO_ROOT}"
fi

if [[ "${TAG_LATEST}" == "true" ]]; then
    docker tag "${IMAGE_REF}" "${IMAGE_NAME}:latest"
fi

printf 'Built image: %s\n' "${IMAGE_REF}"
printf 'App version: %s\n' "${APP_VERSION}"
printf 'Target platform: %s\n' "${PLATFORM}"
printf 'Commit: %s\n' "${COMMIT_HASH}"
printf 'Build date: %s\n' "${BUILD_DATE}"
if [[ "${TAG_LATEST}" == "true" ]]; then
    printf 'Also tagged: %s\n' "${IMAGE_NAME}:latest"
fi
