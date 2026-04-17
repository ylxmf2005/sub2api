#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/migration/audit_server.sh [--scan-root DIR] [--output FILE]
EOF
}

SCAN_ROOT="/"
OUTPUT_FILE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scan-root)
            SCAN_ROOT="$2"
            shift 2
            ;;
        --output)
            OUTPUT_FILE="$2"
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

join_lines() {
    paste -sd ',' -
}

root_path() {
    local path="$1"
    if [[ "${SCAN_ROOT}" == "/" ]]; then
        printf '%s\n' "${path}"
    else
        printf '%s\n' "${SCAN_ROOT%/}${path}"
    fi
}

path_exists() {
    [[ -e "$(root_path "$1")" ]]
}

list_compose_files() {
    find "${SCAN_ROOT}" \
        \( -name 'docker-compose.local.yml' -o -name 'docker-compose.yml' \) \
        -type f 2>/dev/null | sort
}

binary_present=false
systemd_present=false
datamanagement_unit_present=false
datamanagement_socket_present=false
compose_mode="none"
compose_file=""
compose_dir=""
notes=()

if path_exists /opt/sub2api/sub2api || path_exists /etc/sub2api/config.yaml; then
    binary_present=true
fi

if path_exists /etc/systemd/system/sub2api.service || path_exists /lib/systemd/system/sub2api.service; then
    systemd_present=true
fi

if path_exists /etc/systemd/system/sub2api-datamanagementd.service || path_exists /lib/systemd/system/sub2api-datamanagementd.service; then
    datamanagement_unit_present=true
fi

if [[ -S "$(root_path /tmp/sub2api-datamanagement.sock)" ]]; then
    datamanagement_socket_present=true
fi

compose_candidates="$(list_compose_files || true)"
if [[ -n "${compose_candidates}" ]]; then
    while IFS= read -r candidate; do
        [[ -z "${candidate}" ]] && continue
        if grep -q '\./data:/app/data' "${candidate}" 2>/dev/null; then
            compose_mode="docker-local"
            compose_file="${candidate}"
            compose_dir="$(dirname "${candidate}")"
            break
        fi
        if [[ "${compose_mode}" == "none" ]]; then
            compose_mode="docker-named"
            compose_file="${candidate}"
            compose_dir="$(dirname "${candidate}")"
        fi
    done <<< "${compose_candidates}"
fi

source_mode="unknown"
if [[ "${binary_present}" == "true" && "${compose_mode}" != "none" ]]; then
    source_mode="mixed"
    notes+=("binary_and_compose_markers_present")
elif [[ "${compose_mode}" != "none" ]]; then
    source_mode="${compose_mode}"
elif [[ "${binary_present}" == "true" ]]; then
    source_mode="binary"
else
    notes+=("no_known_markers_found")
fi

if command -v docker >/dev/null 2>&1 && [[ "${SCAN_ROOT}" == "/" ]]; then
    volume_names="$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep '^sub2api' || true)"
else
    volume_names=""
fi

if [[ -n "${volume_names}" && "${source_mode}" == "unknown" ]]; then
    source_mode="docker-named"
    notes+=("derived_from_docker_volumes")
fi

{
    printf 'SOURCE_MODE=%s\n' "${source_mode}"
    printf 'BINARY_INSTALL_PRESENT=%s\n' "${binary_present}"
    printf 'SYSTEMD_SUB2API_UNIT_PRESENT=%s\n' "${systemd_present}"
    printf 'COMPOSE_MODE=%s\n' "${compose_mode}"
    printf 'COMPOSE_FILE=%s\n' "${compose_file}"
    printf 'COMPOSE_DIR=%s\n' "${compose_dir}"
    printf 'DATAMANAGEMENT_UNIT_PRESENT=%s\n' "${datamanagement_unit_present}"
    printf 'DATAMANAGEMENT_SOCKET_PRESENT=%s\n' "${datamanagement_socket_present}"
    printf 'DOCKER_VOLUMES=%s\n' "$(printf '%s\n' "${volume_names}" | join_lines)"
    printf 'SCAN_ROOT=%s\n' "${SCAN_ROOT}"
    printf 'NOTES=%s\n' "$(printf '%s\n' "${notes[@]-}" | sed '/^$/d' | join_lines)"
} > "${OUTPUT_FILE:-/dev/stdout}"
