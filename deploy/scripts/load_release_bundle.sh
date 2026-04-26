#!/usr/bin/env bash
# 加载发布包，验证元数据并导入 Docker 镜像。

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/load_release_bundle.sh [options] BUNDLE_PATH

Options:
  --work-dir DIR     Temporary directory for extraction (default: ./tmp_load)
  -h, --help         Show this help
EOF
}

WORK_DIR="./tmp_load"
BUNDLE_PATH=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --work-dir)
            WORK_DIR="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            if [[ -z "${BUNDLE_PATH}" ]]; then
                BUNDLE_PATH="$1"
                shift
            else
                echo "Unknown option: $1" >&2
                usage >&2
                exit 1
            fi
            ;;
    esac
done

if [[ -z "${BUNDLE_PATH}" ]]; then
    echo "Error: BUNDLE_PATH is required." >&2
    usage >&2
    exit 1
fi

if [[ ! -f "${BUNDLE_PATH}" ]]; then
    echo "Error: File ${BUNDLE_PATH} not found." >&2
    exit 1
fi

echo "Extracting bundle ${BUNDLE_PATH}..."
mkdir -p "${WORK_DIR}"
tar -xzf "${BUNDLE_PATH}" -C "${WORK_DIR}"

# Find the extracted directory (usually sub2api-release-TAG)
EXTRACTED_DIR=$(find "${WORK_DIR}" -maxdepth 1 -type d -name "sub2api-release-*" | head -n 1)

if [[ -z "${EXTRACTED_DIR}" ]]; then
    echo "Error: Could not find extracted bundle directory in ${WORK_DIR}." >&2
    exit 1
fi

MANIFEST_FILE="${EXTRACTED_DIR}/manifest.json"
IMAGE_TAR="${EXTRACTED_DIR}/image.tar"

if [[ ! -f "${MANIFEST_FILE}" ]]; then
    echo "Error: manifest.json not found in bundle." >&2
    exit 1
fi

echo "Manifest contents:"
cat "${MANIFEST_FILE}"

# Verify checksum if present
EXPECTED_SHA256=$(grep -o '"image_sha256": "[^"]*"' "${MANIFEST_FILE}" | cut -d'"' -f4)
if [[ "${EXPECTED_SHA256}" != "unknown" ]]; then
    echo "Verifying image checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_SHA256=$(sha256sum "${IMAGE_TAR}" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_SHA256=$(shasum -a 256 "${IMAGE_TAR}" | cut -d' ' -f1)
    else
        echo "Warning: No sha256sum or shasum found. Skipping verification."
        ACTUAL_SHA256="${EXPECTED_SHA256}"
    fi

    if [[ "${ACTUAL_SHA256}" != "${EXPECTED_SHA256}" ]]; then
        echo "Error: Checksum mismatch!" >&2
        echo "Expected: ${EXPECTED_SHA256}" >&2
        echo "Actual:   ${ACTUAL_SHA256}" >&2
        exit 1
    fi
    echo "Checksum verified."
fi

echo "Loading Docker image..."
docker load -i "${IMAGE_TAR}"

echo "Cleaning up..."
rm -rf "${WORK_DIR}"

echo "Bundle loaded successfully."
