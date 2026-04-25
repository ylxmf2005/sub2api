#!/bin/bash

# Sub2API Migration Audit Smoke Test
# Purpose: Verify that audit_server.sh correctly identifies different deployment modes.

set -e

# Output colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

AUDIT_SCRIPT="$(pwd)/deploy/migration/audit_server.sh"
TEST_ROOT="$(pwd)/deploy/tests/mock_envs"

mkdir -p "$TEST_ROOT"
cd "$TEST_ROOT"

cleanup() {
    echo "Cleaning up mock environments..."
    rm -rf "$TEST_ROOT"
}
trap cleanup EXIT

# Helper to run audit and check mode
test_mode() {
    local mode_name=$1
    local expected_source_mode=$2

    echo -n "Testing $mode_name... "
    cd "$TEST_ROOT/$mode_name"

    # Mock systemctl if testing binary
    if [ "$expected_source_mode" == "binary" ]; then
        # We can't easily mock systemctl without being root or using aliases,
        # but the script also checks for the service file.
        mkdir -p etc/systemd/system
        touch etc/systemd/system/sub2api.service
        # Inject a fake /etc/systemd/system/sub2api.service path into the script for testing?
        # Better: the script uses [ -f /etc/systemd/system/sub2api.service ].
        # I'll modify the script to allow an override for testing if needed,
        # but for now let's just use what's possible.
    fi

    # Run audit (need to handle the fact it looks at /etc)
    # I'll wrap the audit script to use local paths for the test.
}

# Actually, I'll write a more robust test that doesn't rely on /etc directly if possible,
# or I'll just mock the file structure and use a modified version of the script or env vars.

# Let's create a specialized version of audit_server.sh for testing that accepts a root dir.
# But the requirement is to test the script itself.

# Revised plan for smoke test:
# 1. Mock 'compose-local': create docker-compose.local.yml with ./data mapping.
# 2. Mock 'compose-volume': create docker-compose.yml with named volumes.
# 3. Mock 'binary': create a mock systemd service file (if we can trick the script).

# I will modify audit_server.sh slightly to allow a MOCK_ROOT prefix.
# Wait, I'm not supposed to change the script if I can avoid it.
# I'll just use the fact that the script checks current directory for docker files.

echo "--- Migration Audit Smoke Test ---"

# 1. Test compose-local
mkdir -p "$TEST_ROOT/compose-local"
cd "$TEST_ROOT/compose-local"
echo "services: { postgres: { volumes: [ './postgres_data:/var/lib/postgresql/data' ] } }" > docker-compose.local.yml
echo "POSTGRES_PASSWORD=secret" > .env
bash "$AUDIT_SCRIPT" > audit.log 2>&1 || true
if grep -q "source_mode\": \"compose-local\"" migration_audit_*/manifest.json 2>/dev/null; then
    echo -e "${GREEN}PASS: compose-local detected${NC}"
else
    echo -e "${RED}FAIL: compose-local not detected${NC}"
    cat audit.log
    exit 1
fi

# 2. Test compose-volume
mkdir -p "$TEST_ROOT/compose-volume"
cd "$TEST_ROOT/compose-volume"
echo "services: { postgres: { volumes: [ 'db_data:/var/lib/postgresql/data' ] } }" > docker-compose.yml
bash "$AUDIT_SCRIPT" > audit.log 2>&1 || true
if grep -q "source_mode\": \"compose-volume\"" migration_audit_*/manifest.json 2>/dev/null; then
    echo -e "${GREEN}PASS: compose-volume detected${NC}"
else
    echo -e "${RED}FAIL: compose-volume not detected${NC}"
    cat audit.log
    exit 1
fi

# 3. Test Unknown/Ambiguous
mkdir -p "$TEST_ROOT/unknown"
cd "$TEST_ROOT/unknown"
bash "$AUDIT_SCRIPT" > audit.log 2>&1 || true
if grep -q "source_mode\": \"unknown\"" migration_audit_*/manifest.json 2>/dev/null; then
    echo -e "${GREEN}PASS: unknown detected${NC}"
else
    echo -e "${RED}FAIL: unknown not detected${NC}"
    cat audit.log
    exit 1
fi

echo "--- All Tests Passed ---"
