#!/bin/bash

# Sub2API Server Audit Script
# Purpose: Detect the current deployment mode and capture essential configuration.

set -e

# Output colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Sub2API Server Audit...${NC}"

AUDIT_DIR="./migration_audit_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$AUDIT_DIR"
MANIFEST="$AUDIT_DIR/manifest.json"

echo "{" > "$MANIFEST"
echo "  \"audit_timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"," >> "$MANIFEST"

# 1. Detect Binary/Systemd Installation
BINARY_DETECTED=false
if systemctl is-active --quiet sub2api 2>/dev/null || [ -f /etc/systemd/system/sub2api.service ]; then
    BINARY_DETECTED=true
    echo -e "${GREEN}[+] Detected Systemd service (Binary Install)${NC}"
fi

# 2. Detect Docker Compose
DOCKER_COMPOSE_DETECTED=false
COMPOSE_FILE=""
if [ -f "docker-compose.local.yml" ]; then
    DOCKER_COMPOSE_DETECTED=true
    COMPOSE_FILE="docker-compose.local.yml"
    echo -e "${GREEN}[+] Detected docker-compose.local.yml${NC}"
elif [ -f "docker-compose.yml" ]; then
    DOCKER_COMPOSE_DETECTED=true
    COMPOSE_FILE="docker-compose.yml"
    echo -e "${GREEN}[+] Detected docker-compose.yml${NC}"
fi

# 3. Determine Source Mode
SOURCE_MODE="unknown"
if $BINARY_DETECTED && ! $DOCKER_COMPOSE_DETECTED; then
    SOURCE_MODE="binary"
elif $DOCKER_COMPOSE_DETECTED; then
    # Check if it uses local directories or named volumes
    if grep -q "\./postgres_data" "$COMPOSE_FILE" 2>/dev/null || grep -q "\./data" "$COMPOSE_FILE" 2>/dev/null; then
        SOURCE_MODE="compose-local"
    else
        SOURCE_MODE="compose-volume"
    fi
fi

echo -e "${YELLOW}Inferred Source Mode: $SOURCE_MODE${NC}"
echo "  \"source_mode\": \"$SOURCE_MODE\"," >> "$MANIFEST"

# 4. Capture Config and Secrets
echo "  \"captured_files\": [" >> "$MANIFEST"

capture_file() {
    local src=$1
    local name=$2
    if [ -f "$src" ]; then
        cp "$src" "$AUDIT_DIR/$name"
        echo "    \"$name\"," >> "$MANIFEST"
        echo -e "${GREEN}[+] Captured $src as $name${NC}"
    fi
}

# Capture .env if exists (for Docker)
capture_file ".env" "env_file"

# Capture config.yaml if exists
if [ "$SOURCE_MODE" == "binary" ]; then
    capture_file "/etc/sub2api/config.yaml" "config.yaml"
    # Also capture service file to see env vars
    capture_file "/etc/systemd/system/sub2api.service" "sub2api.service"
elif [ "$SOURCE_MODE" == "compose-local" ] || [ "$SOURCE_MODE" == "compose-volume" ]; then
    # Try to find config.yaml in data dir
    if [ -f "data/config.yaml" ]; then
        capture_file "data/config.yaml" "config.yaml"
    fi
    capture_file "$COMPOSE_FILE" "docker-compose.src.yml"
fi

# Remove trailing comma from last captured file entry
sed -i '$ s/,$//' "$MANIFEST"
echo "  ]," >> "$MANIFEST"

# 5. Extract Key Secrets for verification
echo "  \"extracted_secrets\": {" >> "$MANIFEST"

extract_env() {
    local file=$1
    local var=$2
    local val=$(grep "^$var=" "$file" | cut -d'=' -f2- | tr -d '"' | tr -d "'")
    if [ -n "$val" ]; then
        echo "    \"$var\": \"$val\"," >> "$MANIFEST"
    fi
}

if [ -f "$AUDIT_DIR/env_file" ]; then
    extract_env "$AUDIT_DIR/env_file" "POSTGRES_PASSWORD"
    extract_env "$AUDIT_DIR/env_file" "JWT_SECRET"
    extract_env "$AUDIT_DIR/env_file" "TOTP_ENCRYPTION_KEY"
    extract_env "$AUDIT_DIR/env_file" "ADMIN_EMAIL"
fi

# Extract from service file if binary
if [ -f "$AUDIT_DIR/sub2api.service" ]; then
    grep "Environment=" "$AUDIT_DIR/sub2api.service" | while read -r line; do
        pair=$(echo "$line" | cut -d'=' -f2-)
        key=$(echo "$pair" | cut -d'=' -f1)
        val=$(echo "$pair" | cut -d'=' -f2-)
        echo "    \"$key\": \"$val\"," >> "$MANIFEST"
    done
fi

# Remove trailing comma from last secret entry
sed -i '$ s/,$//' "$MANIFEST"
echo "  }" >> "$MANIFEST"

echo "}" >> "$MANIFEST"

echo -e "${GREEN}Audit complete. Results in $AUDIT_DIR${NC}"
