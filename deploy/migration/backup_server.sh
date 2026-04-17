#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/migration/backup_server.sh \
    --source-mode MODE \
    --output-dir DIR \
    [--compose-dir DIR] \
    [--binary-root DIR] \
    [--etc-root DIR] \
    [--pg-volume NAME] \
    [--redis-volume NAME] \
    [--data-volume NAME]
EOF
}

SOURCE_MODE=""
OUTPUT_ROOT=""
COMPOSE_DIR=""
BINARY_ROOT="/opt/sub2api"
ETC_ROOT="/etc/sub2api"
PG_VOLUME=""
REDIS_VOLUME=""
DATA_VOLUME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --source-mode)
            SOURCE_MODE="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_ROOT="$2"
            shift 2
            ;;
        --compose-dir)
            COMPOSE_DIR="$2"
            shift 2
            ;;
        --binary-root)
            BINARY_ROOT="$2"
            shift 2
            ;;
        --etc-root)
            ETC_ROOT="$2"
            shift 2
            ;;
        --pg-volume)
            PG_VOLUME="$2"
            shift 2
            ;;
        --redis-volume)
            REDIS_VOLUME="$2"
            shift 2
            ;;
        --data-volume)
            DATA_VOLUME="$2"
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

if [[ -z "${SOURCE_MODE}" || -z "${OUTPUT_ROOT}" ]]; then
    usage >&2
    exit 1
fi

timestamp="$(date -u +"%Y%m%dT%H%M%SZ")"
backup_dir="${OUTPUT_ROOT%/}/${timestamp}"
mkdir -p "${backup_dir}"

tar_if_exists() {
    local source="$1"
    local target="$2"
    if [[ -e "${source}" ]]; then
        tar -C "$(dirname "${source}")" -czf "${target}" "$(basename "${source}")"
    fi
}

dump_volume() {
    local volume_name="$1"
    local archive_name="$2"
    [[ -z "${volume_name}" ]] && return 0
    docker run --rm -v "${volume_name}:/source:ro" -v "${backup_dir}:/backup" alpine \
        tar -C /source -czf "/backup/${archive_name}" .
}

case "${SOURCE_MODE}" in
    docker-local)
        if [[ -z "${COMPOSE_DIR}" ]]; then
            echo "--compose-dir is required for docker-local mode" >&2
            exit 1
        fi
        tar_if_exists "${COMPOSE_DIR}" "${backup_dir}/compose-dir.tar.gz"
        tar_if_exists "${COMPOSE_DIR}/data" "${backup_dir}/data.tar.gz"
        tar_if_exists "${COMPOSE_DIR}/postgres_data" "${backup_dir}/postgres_data.tar.gz"
        tar_if_exists "${COMPOSE_DIR}/redis_data" "${backup_dir}/redis_data.tar.gz"
        ;;
    docker-named)
        dump_volume "${DATA_VOLUME}" "sub2api_data_volume.tar.gz"
        dump_volume "${PG_VOLUME}" "postgres_volume.tar.gz"
        dump_volume "${REDIS_VOLUME}" "redis_volume.tar.gz"
        if [[ -n "${COMPOSE_DIR}" ]]; then
            tar_if_exists "${COMPOSE_DIR}" "${backup_dir}/compose-dir.tar.gz"
        fi
        ;;
    binary)
        tar_if_exists "${BINARY_ROOT}" "${backup_dir}/binary-root.tar.gz"
        tar_if_exists "${ETC_ROOT}" "${backup_dir}/etc-root.tar.gz"
        ;;
    mixed)
        if [[ -n "${COMPOSE_DIR}" ]]; then
            tar_if_exists "${COMPOSE_DIR}" "${backup_dir}/compose-dir.tar.gz"
        fi
        tar_if_exists "${BINARY_ROOT}" "${backup_dir}/binary-root.tar.gz"
        tar_if_exists "${ETC_ROOT}" "${backup_dir}/etc-root.tar.gz"
        [[ -n "${DATA_VOLUME}" ]] && dump_volume "${DATA_VOLUME}" "sub2api_data_volume.tar.gz"
        [[ -n "${PG_VOLUME}" ]] && dump_volume "${PG_VOLUME}" "postgres_volume.tar.gz"
        [[ -n "${REDIS_VOLUME}" ]] && dump_volume "${REDIS_VOLUME}" "redis_volume.tar.gz"
        ;;
    *)
        echo "Unsupported source mode: ${SOURCE_MODE}" >&2
        exit 1
        ;;
esac

if command -v systemctl >/dev/null 2>&1; then
    systemctl cat sub2api > "${backup_dir}/systemd-sub2api.service.txt" 2>/dev/null || true
    systemctl cat sub2api-datamanagementd > "${backup_dir}/systemd-datamanagementd.service.txt" 2>/dev/null || true
fi

if command -v docker >/dev/null 2>&1; then
    docker ps --format '{{.Names}} {{.Image}} {{.Status}}' > "${backup_dir}/docker-ps.txt" 2>/dev/null || true
    docker volume ls > "${backup_dir}/docker-volumes.txt" 2>/dev/null || true
fi

printf 'SOURCE_MODE=%s\n' "${SOURCE_MODE}" > "${backup_dir}/manifest.env"
printf 'CREATED_AT_UTC=%s\n' "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${backup_dir}/manifest.env"
printf 'COMPOSE_DIR=%s\n' "${COMPOSE_DIR}" >> "${backup_dir}/manifest.env"
printf 'BINARY_ROOT=%s\n' "${BINARY_ROOT}" >> "${backup_dir}/manifest.env"
printf 'ETC_ROOT=%s\n' "${ETC_ROOT}" >> "${backup_dir}/manifest.env"

printf 'Backup written to: %s\n' "${backup_dir}"
