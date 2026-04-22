#!/usr/bin/env bash
# 冒烟测试：验证发布包的导出和加载流程。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

EXPORT_SCRIPT="${REPO_ROOT}/deploy/scripts/export_release_bundle.sh"
LOAD_SCRIPT="${REPO_ROOT}/deploy/scripts/load_release_bundle.sh"

TEST_IMAGE="sub2api-test"
TEST_TAG="smoke-$(date +%s)"
TEST_REF="${TEST_IMAGE}:${TEST_TAG}"
OUTPUT_DIR="${REPO_ROOT}/tmp_test_release"

cleanup() {
    echo "Cleaning up..."
    rm -rf "${OUTPUT_DIR}"
    docker rmi "${TEST_REF}" 2>/dev/null || true
}

trap cleanup EXIT

echo "Creating dummy test image..."
# Use a very small base image for testing
if ! docker pull alpine:latest; then
    echo "Error: Failed to pull alpine:latest"
    exit 1
fi
docker tag alpine:latest "${TEST_REF}"

echo "Step 1: Exporting release bundle..."
chmod +x "${EXPORT_SCRIPT}"
"${EXPORT_SCRIPT}" --image-name "${TEST_IMAGE}" --image-tag "${TEST_TAG}" --output-dir "${OUTPUT_DIR}"

BUNDLE_PATH="${OUTPUT_DIR}/sub2api-release-${TEST_TAG}.tar.gz"
if [[ ! -f "${BUNDLE_PATH}" ]]; then
    echo "Error: Release bundle not created at ${BUNDLE_PATH}"
    exit 1
fi

echo "Step 2: Removing local test image to verify load..."
docker rmi "${TEST_REF}"

echo "Step 3: Loading release bundle..."
chmod +x "${LOAD_SCRIPT}"
"${LOAD_SCRIPT}" --work-dir "${OUTPUT_DIR}/tmp_load" "${BUNDLE_PATH}"

echo "Step 4: Verifying loaded image..."
if ! docker image inspect "${TEST_REF}" >/dev/null 2>&1; then
    echo "Error: Loaded image ${TEST_REF} not found in Docker."
    exit 1
fi

echo "Smoke test passed!"
