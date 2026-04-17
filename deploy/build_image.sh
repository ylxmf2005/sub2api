#!/usr/bin/env bash
# 本地构建私有镜像，避免把构建职责丢给生产服务器。

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/build_image.sh [options]

Options:
  --image-name NAME    Docker image name (default: sub2api-private)
  --image-tag TAG      Docker image tag (default: local)
  --tag-latest         Also tag IMAGE_NAME:latest
  --goproxy URL        GOPROXY value
  --gosumdb VALUE      GOSUMDB value
  -h, --help           Show this help
EOF
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE_NAME="${IMAGE_NAME:-sub2api-private}"
IMAGE_TAG="${IMAGE_TAG:-local}"
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

docker build \
    -t "${IMAGE_REF}" \
    --build-arg GOPROXY="${GOPROXY_VALUE}" \
    --build-arg GOSUMDB="${GOSUMDB_VALUE}" \
    -f "${REPO_ROOT}/Dockerfile" \
    "${REPO_ROOT}"

if [[ "${TAG_LATEST}" == "true" ]]; then
    docker tag "${IMAGE_REF}" "${IMAGE_NAME}:latest"
fi

printf 'Built image: %s\n' "${IMAGE_REF}"
if [[ "${TAG_LATEST}" == "true" ]]; then
    printf 'Also tagged: %s\n' "${IMAGE_NAME}:latest"
fi
