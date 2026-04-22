#!/bin/bash

# Sub2API Server Backup Script
# Purpose: Create a complete backup of the current deployment state.

set -e

# Output colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

SOURCE_MODE=$1
if [ -z "$SOURCE_MODE" ]; then
    echo "Usage: $0 <source_mode> [compose_file]"
    echo "Modes: binary, compose-local, compose-volume"
    exit 1
fi

COMPOSE_FILE=${2:-"docker-compose.local.yml"}
BACKUP_DIR="./backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo -e "${YELLOW}Starting backup for mode: $SOURCE_MODE...${NC}"

# 1. Backup Config and Secrets (already partially done by audit, but let's be thorough)
mkdir -p "$BACKUP_DIR/config"
if [ "$SOURCE_MODE" == "binary" ]; then
    [ -f /etc/sub2api/config.yaml ] && cp /etc/sub2api/config.yaml "$BACKUP_DIR/config/"
    [ -f /etc/systemd/system/sub2api.service ] && cp /etc/systemd/system/sub2api.service "$BACKUP_DIR/config/"
elif [[ "$SOURCE_MODE" == compose-* ]]; then
    [ -f .env ] && cp .env "$BACKUP_DIR/config/"
    [ -f "$COMPOSE_FILE" ] && cp "$COMPOSE_FILE" "$BACKUP_DIR/config/"
    [ -f data/config.yaml ] && cp data/config.yaml "$BACKUP_DIR/config/"
fi

# 2. Backup Database
echo -e "${YELLOW}Backing up database...${NC}"
mkdir -p "$BACKUP_DIR/database"

if [ "$SOURCE_MODE" == "binary" ]; then
    # Assuming local postgres, try to dump
    if command -v pg_dump > /dev/null; then
        # This might require PGPASSWORD or .pgpass
        echo "Attempting pg_dump (may require credentials)..."
        pg_dump -U postgres sub2api > "$BACKUP_DIR/database/dump.sql" || echo "Warning: pg_dump failed. Ensure database is accessible."
    fi
elif [ "$SOURCE_MODE" == "compose-local" ] || [ "$SOURCE_MODE" == "compose-volume" ]; then
    # Try to dump from container
    POSTGRES_CONTAINER=$(docker compose -f "$COMPOSE_FILE" ps -q postgres 2>/dev/null || docker ps -q --filter "name=postgres")
    if [ -n "$POSTGRES_CONTAINER" ]; then
        docker exec "$POSTGRES_CONTAINER" pg_dump -U postgres sub2api > "$BACKUP_DIR/database/dump.sql" || echo "Warning: Docker pg_dump failed."
    fi
    
    # For compose-local, also backup the directory
    if [ "$SOURCE_MODE" == "compose-local" ] && [ -d "postgres_data" ]; then
        tar czf "$BACKUP_DIR/database/postgres_data.tar.gz" postgres_data/
    fi
fi

# 3. Backup Runtime Data
echo -e "${YELLOW}Backing up runtime data...${NC}"
if [ "$SOURCE_MODE" == "binary" ] && [ -d "/opt/sub2api/data" ]; then
    tar czf "$BACKUP_DIR/runtime_data.tar.gz" -C /opt/sub2api data/
elif [ -d "data" ]; then
    tar czf "$BACKUP_DIR/runtime_data.tar.gz" data/
fi

# 4. Create Manifest
cat <<EOF > "$BACKUP_DIR/manifest.json"
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "source_mode": "$SOURCE_MODE",
  "contents": {
    "config": "$(ls "$BACKUP_DIR/config" | tr '\n' ',' | sed 's/,$//')",
    "database_dump": "$(f="$BACKUP_DIR/database/dump.sql"; [ -f "$f" ] && echo "present" || echo "absent")",
    "runtime_data": "$(f="$BACKUP_DIR/runtime_data.tar.gz"; [ -f "$f" ] && echo "present" || echo "absent")"
  }
}
EOF

# 5. Finalize
tar czf "${BACKUP_DIR}.tar.gz" "$BACKUP_DIR"
echo -e "${GREEN}Backup complete: ${BACKUP_DIR}.tar.gz${NC}"
rm -rf "$BACKUP_DIR"
