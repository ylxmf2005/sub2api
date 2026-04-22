#!/usr/bin/env bash

# =============================================================================
# Compose Configuration Smoke Test
# =============================================================================
# Verifies that the merged Docker Compose configuration (base + private)
# is valid and that required environment variables are handled correctly.
# =============================================================================

set -e

# Base directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "--- Checking Compose Configuration ---"

# 1. Create a dummy .env file if it doesn't exist for the test
TEST_ENV="${DEPLOY_DIR}/.env.test"
cat <<EOF > "${TEST_ENV}"
POSTGRES_PASSWORD=test_password
SUB2API_IMAGE=local/sub2api-private:test_tag
SUB2API_RELEASE_TAG=test_tag
EOF

# 2. Run docker compose config to verify merging
# We use --env-file to avoid polluting the environment
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    echo "Error: docker-compose or docker compose not found."
    exit 1
fi

echo "Running: ${DOCKER_COMPOSE} -f docker-compose.local.yml -f docker-compose.private.yml --env-file .env.test config"

cd "${DEPLOY_DIR}"
CONFIG_OUTPUT=$(${DOCKER_COMPOSE} -f docker-compose.local.yml -f docker-compose.private.yml --env-file .env.test config 2>&1)

if [ $? -ne 0 ]; then
    echo "FAILED: Merged configuration is invalid."
    echo "${CONFIG_OUTPUT}"
    rm "${TEST_ENV}"
    exit 1
fi

echo "SUCCESS: Merged configuration is valid."

# 3. Verify specific overrides are present in the output
echo "--- Verifying Overrides ---"

# Check for the private image name
if echo "${CONFIG_OUTPUT}" | grep -q "local/sub2api-private:test_tag"; then
    echo "✓ Found private image: local/sub2api-private:test_tag"
else
    echo "FAILED: Private image override not found in config."
    rm "${TEST_ENV}"
    exit 1
fi

# Check for the private-fork label
if echo "${CONFIG_OUTPUT}" | grep -q "io.sub2api.deployment: private-fork"; then
    echo "✓ Found label: io.sub2api.deployment: private-fork"
else
    echo "FAILED: Private deployment label not found in config."
    rm "${TEST_ENV}"
    exit 1
fi

# Cleanup
rm "${TEST_ENV}"

echo "--- All Compose Smoke Tests Passed ---"
