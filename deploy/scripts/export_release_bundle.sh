#!/usr/bin/env bash
# 导出发布包，包含 Docker 镜像和元数据清单。

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/export_release_bundle.sh [options]

Options:
  --image-name NAME    Docker image name (default: local/sub2api-private)
  --image-tag TAG      Docker image tag (default: local)
  --output-dir DIR     Directory to save the bundle (default: ./release)
  -h, --help           Show this help
EOF
}

IMAGE_NAME="local/sub2api-private"
IMAGE_TAG="local"
OUTPUT_DIR="./release"

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

IMAGE_REF="${IMAGE_NAME}:${IMAGE_TAG}"
BUNDLE_NAME="sub2api-release-${IMAGE_TAG}"
BUNDLE_DIR="${OUTPUT_DIR}/${BUNDLE_NAME}"
MANIFEST_FILE="${BUNDLE_DIR}/manifest.json"
IMAGE_TAR="${BUNDLE_DIR}/image.tar"

# Check if image exists
if ! docker image inspect "${IMAGE_REF}" >/dev/null 2>&1; then
    echo "Error: Image ${IMAGE_REF} not found." >&2
    exit 1
fi

echo "Creating release bundle for ${IMAGE_REF}..."
mkdir -p "${BUNDLE_DIR}"

echo "Saving Docker image to ${IMAGE_TAR}..."
docker save -o "${IMAGE_TAR}" "${IMAGE_REF}"

echo "Generating manifest..."
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
IMAGE_SIZE=$(stat -c%s "${IMAGE_TAR}" 2>/dev/null || stat -f%z "${IMAGE_TAR}")
COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

# Checksum helper
if command -v sha256sum >/dev/null 2>&1; then
    IMAGE_SHA256=$(sha256sum "${IMAGE_TAR}" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
    IMAGE_SHA256=$(shasum -a 256 "${IMAGE_TAR}" | cut -d' ' -f1)
else
    echo "Warning: No sha256sum or shasum found. Skipping checksum."
    IMAGE_SHA256="unknown"
fi

cat > "${MANIFEST_FILE}" <<EOF
{
  "version": "${IMAGE_TAG}",
  "image_name": "${IMAGE_NAME}",
  "image_tag": "${IMAGE_TAG}",
  "image_ref": "${IMAGE_REF}",
  "git_commit": "${COMMIT_HASH}",
  "build_date": "${BUILD_DATE}",
  "image_size_bytes": ${IMAGE_SIZE},
  "image_sha256": "${IMAGE_SHA256}",
  "bundle_format": "v1"
}
EOF

echo "Manifest contents:"
cat "${MANIFEST_FILE}"

echo "Packaging bundle..."
# Use absolute path for BUNDLE_NAME.tar.gz to avoid issues with cd
BUNDLE_TAR_GZ="$(cd "${OUTPUT_DIR}" && pwd)/${BUNDLE_NAME}.tar.gz"
(cd "${OUTPUT_DIR}" && tar -czf "${BUNDLE_TAR_GZ}" "${BUNDLE_NAME}")

rm -rf "${BUNDLE_DIR}"

echo "Release bundle created: ${BUNDLE_TAR_GZ}"
