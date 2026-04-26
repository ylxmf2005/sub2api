#!/bin/bash

# Sub2API Runtime Data Verification Script
# Purpose: Verify the presence and integrity of the runtime data directory.

set -e

# Output colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

DATA_DIR=${1:-"./data"}

echo -e "${YELLOW}Verifying runtime data in $DATA_DIR...${NC}"

if [ ! -d "$DATA_DIR" ]; then
    echo -e "${RED}[!] Error: Data directory $DATA_DIR does not exist.${NC}"
    exit 1
fi

# 1. Check for essential files
ESSENTIAL_FILES=("config.yaml" ".installed")
for file in "${ESSENTIAL_FILES[@]}"; do
    if [ -f "$DATA_DIR/$file" ]; then
        echo -e "${GREEN}[+] Found $file${NC}"
    else
        echo -e "${YELLOW}[!] Warning: $file is missing.${NC}"
    fi
done

# 2. Check for logs directory
if [ -d "$DATA_DIR/logs" ]; then
    echo -e "${GREEN}[+] Logs directory exists.${NC}"
else
    echo -e "${YELLOW}[!] Warning: logs directory is missing.${NC}"
fi

# 3. Check for certs/keys if applicable
if [ -d "$DATA_DIR/certs" ]; then
    echo -e "${GREEN}[+] Certs directory exists.${NC}"
fi

# 4. Permissions check (should be writable by the container user)
# Assuming typical Docker user (uid 1000 or root)
if [ -w "$DATA_DIR" ]; then
    echo -e "${GREEN}[+] Data directory is writable.${NC}"
else
    echo -e "${RED}[!] Error: Data directory is NOT writable.${NC}"
    exit 1
fi

echo -e "${GREEN}Runtime data verification complete.${NC}"
