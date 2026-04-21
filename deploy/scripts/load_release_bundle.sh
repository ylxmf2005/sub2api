#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./deploy/scripts/load_release_bundle.sh --bundle PATH [--deploy-dir DIR] [--force]
EOF
}

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

BUNDLE_PATH=""
DEPLOY_DIR=""
FORCE=false
IMAGE_REF=""
IMAGE_PLATFORM=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --bundle)
            BUNDLE_PATH="$2"
            shift 2
            ;;
        --deploy-dir)
            DEPLOY_DIR="$2"
            shift 2
            ;;
        --force)
            FORCE=true
            shift
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

if [[ -z "${BUNDLE_PATH}" ]]; then
    usage >&2
    exit 1
fi

TEMP_DIR=""
cleanup() {
    if [[ -n "${TEMP_DIR}" && -d "${TEMP_DIR}" ]]; then
        rm -rf "${TEMP_DIR}"
    fi
}
trap cleanup EXIT

if [[ -d "${BUNDLE_PATH}" ]]; then
    BUNDLE_DIR="${BUNDLE_PATH}"
else
    TEMP_DIR="$(mktemp -d)"
    tar -C "${TEMP_DIR}" -xzf "${BUNDLE_PATH}"
    first_dir="$(find "${TEMP_DIR}" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
    BUNDLE_DIR="${first_dir}"
fi

if [[ ! -f "${BUNDLE_DIR}/image.tar" ]]; then
    echo "Bundle missing image.tar" >&2
    exit 1
fi

if [[ -f "${BUNDLE_DIR}/image.tar.sha256" ]]; then
    expected="$(tr -d '[:space:]' < "${BUNDLE_DIR}/image.tar.sha256")"
    actual="$(sha256_file "${BUNDLE_DIR}/image.tar")"
    if [[ "${expected}" != "${actual}" ]]; then
        echo "image.tar checksum mismatch" >&2
        exit 1
    fi
fi

if [[ -f "${BUNDLE_DIR}/manifest.env" ]]; then
    # shellcheck disable=SC1090
    source "${BUNDLE_DIR}/manifest.env"
fi

docker load -i "${BUNDLE_DIR}/image.tar" >/dev/null

if [[ -n "${IMAGE_REF}" && -n "${IMAGE_PLATFORM}" ]]; then
    loaded_platform="$(docker image inspect "${IMAGE_REF}" --format '{{.Os}}/{{.Architecture}}')"
    host_platform="$(docker version --format '{{.Server.Os}}/{{.Server.Arch}}')"
    if [[ "${loaded_platform}" != "${IMAGE_PLATFORM}" ]]; then
        echo "Loaded image platform ${loaded_platform} does not match bundle manifest ${IMAGE_PLATFORM}" >&2
        exit 1
    fi
    if [[ "${loaded_platform}" != "${host_platform}" ]]; then
        echo "Image platform ${loaded_platform} does not match Docker host platform ${host_platform}" >&2
        exit 1
    fi
fi

if [[ -n "${DEPLOY_DIR}" ]]; then
    mkdir -p "${DEPLOY_DIR}"
    for file in "${BUNDLE_DIR}/compose/"*; do
        target="${DEPLOY_DIR}/$(basename "${file}")"
        if [[ -e "${target}" && "${FORCE}" != "true" ]]; then
            echo "Refusing to overwrite ${target} without --force" >&2
            exit 1
        fi
        cp -f "${file}" "${target}"
    done
fi

printf 'Loaded image bundle from: %s\n' "${BUNDLE_DIR}"
if [[ -n "${DEPLOY_DIR}" ]]; then
    printf 'Copied compose assets into: %s\n' "${DEPLOY_DIR}"
fi
