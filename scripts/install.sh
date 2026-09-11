#!/usr/bin/env bash

set -Eeuo pipefail

PROJECT_NAME="ktauth"
DOWNLOAD_URL="https://ktauth.kaju.win"
DEPLOY_DIR="/opt/${PROJECT_NAME}"
COMPOSE_FILE="${DEPLOY_DIR}/compose.yaml"
ENV_FILE="${DEPLOY_DIR}/.env"
IMAGE="stellashiina/ktauth:latest"

ADMIN_NAME="ktauth"
ADMIN_PASSWD="ktauth"
JWT_SECRET="ktauthsecret"
ADDRESS="51214"
TRUSTED_PROXIES="127.0.0.0/8,::1/128,172.16.0.0/12,192.168.0.0/16,10.0.0.0/8"

if [[ -t 2 && -z "${NO_COLOR:-}" ]]; then
    RED=$'\033[0;31m'
    GREEN=$'\033[0;32m'
    YELLOW=$'\033[1;33m'
    NC=$'\033[0m'
else
    RED=""
    GREEN=""
    YELLOW=""
    NC=""
fi

log_info() {
    printf '%s[INFO]%s %s\n' "${GREEN}" "${NC}" "$*" >&2
}

log_warn() {
    printf '%s[WARN]%s %s\n' "${YELLOW}" "${NC}" "$*" >&2
}

log_error() {
    printf '%s[ERROR]%s %s\n' "${RED}" "${NC}" "$*" >&2
}

die() {
    log_error "$*"
    exit 1
}

on_error() {
    local exit_code=$?
    local line=$1
    log_error "Command failed at line ${line} (exit ${exit_code})."
    exit "${exit_code}"
}

trap 'on_error "$LINENO"' ERR

require_command() {
    command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

require_root() {
    [[ ${EUID} -eq 0 ]] || die "Run this command as root (for example, with sudo)."
}

confirm_yes() {
    local prompt=$1
    local response

    read -r -p "${prompt} [Y/n]: " response
    case "${response}" in
        [nN]|[nN][oO]) return 1 ;;
        *) return 0 ;;
    esac
}

confirm_no() {
    local prompt=$1
    local response

    read -r -p "${prompt} [y/N]: " response
    case "${response}" in
        [yY]|[yY][eE][sS]) return 0 ;;
        *) return 1 ;;
    esac
}

compose() {
    docker compose \
        --project-name "${PROJECT_NAME}" \
        --project-directory "${DEPLOY_DIR}" \
        --file "${COMPOSE_FILE}" \
        "$@"
}

download_file() {
    local source_url=$1
    local destination=$2
    local destination_dir
    local temporary

    destination_dir=$(dirname "${destination}")
    mkdir -p "${destination_dir}"
    temporary=$(mktemp "${destination}.tmp.XXXXXX")

    if ! curl \
        --fail \
        --silent \
        --show-error \
        --location \
        --retry 3 \
        --retry-delay 1 \
        --connect-timeout 10 \
        --proto '=https' \
        --tlsv1.2 \
        --output "${temporary}" \
        "${source_url}"; then
        rm -f "${temporary}"
        die "Download failed: ${source_url}"
    fi

    if [[ ! -s ${temporary} ]]; then
        rm -f "${temporary}"
        die "Downloaded file is empty: ${source_url}"
    fi

    mv -f "${temporary}" "${destination}"
    log_info "Downloaded ${source_url} -> ${destination}"
}

ensure_docker() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        docker info >/dev/null 2>&1 || die "Docker is installed, but the daemon is not available."
        return
    fi

    if ! confirm_yes "Docker Engine or Docker Compose is unavailable. Install it now?"; then
        die "Install Docker Engine and Docker Compose, then run this command again."
    fi

    local installer
    installer=$(mktemp)
    download_file "https://get.docker.com" "${installer}"
    sh "${installer}"
    rm -f "${installer}"

    command -v docker >/dev/null 2>&1 || die "Docker installation did not provide the docker command."
    docker compose version >/dev/null 2>&1 || die "Docker Compose plugin is unavailable after installation."
    docker info >/dev/null 2>&1 || die "Docker was installed, but the daemon is not available."
}

require_docker() {
    require_command docker
    docker compose version >/dev/null 2>&1 || die "Docker Compose plugin is unavailable."
    docker info >/dev/null 2>&1 || die "Docker daemon is not available."
}

download_compose_file() {
    download_file "${DOWNLOAD_URL}/compose.yaml" "${COMPOSE_FILE}"
}

download_sql_init() {
    download_file "${DOWNLOAD_URL}/00-init.sql" "${DEPLOY_DIR}/sql/00-init.sql"
}

download_env_example() {
    download_file "${DOWNLOAD_URL}/.env.example" "${DEPLOY_DIR}/.env.example"
}

get_address() {
    local value

    [[ -f ${COMPOSE_FILE} ]] || return 1
    value=$(sed -nE 's/^[[:space:]]*-[[:space:]]*"([^\"]+):51214"[[:space:]]*$/\1/p' "${COMPOSE_FILE}" | head -n 1)
    [[ -n ${value} ]] || return 1
    ADDRESS=${value}
}

valid_address() {
    local value=$1
    local host
    local port

    case "${value}" in
        *[!0-9]*)
            case "${value}" in
                \[*\]:*)
                    host=${value%:*}
                    port=${value##*:}
                    [[ ${host} == \[*\] && ${host} != *[!0-9a-fA-F:\[\]]* ]] || return 1
                    ;;
                *:*)
                    host=${value%:*}
                    port=${value##*:}
                    [[ -n ${host} && ${host} != *:* && ${host} != *[!a-zA-Z0-9._-]* ]] || return 1
                    ;;
                *) return 1 ;;
            esac
            ;;
        *)
            port=${value}
            ;;
    esac

    [[ -n ${port} && ${#port} -le 5 && ${port} != *[!0-9]* ]] || return 1

    ((10#${port} >= 1 && 10#${port} <= 65535))
}

# Rough validation of a comma-separated TRUSTED_PROXIES list: every element must
# be an IPv4 or IPv6 address, optionally followed by /prefix. The value is handed
# to gin's SetTrustedProxies verbatim, which rejects anything it cannot parse by
# panicking at startup, so catching obvious mistakes here is enough. Like
# valid_address, use this in a condition context only.
valid_trusted_proxies() {
    local value=$1
    local elements
    local element
    local host
    local prefix
    local octets
    local octet

    [[ -n ${value} ]] || return 1

    # read -a drops a trailing empty field, so the delimiter edges need their
    # own check; an interior ",," still shows up as an empty element below.
    case ${value} in
        ,*|*,) return 1 ;;
    esac

    # read -a avoids pathname expansion, so a stray "*" cannot glob the cwd.
    IFS=',' read -r -a elements <<<"${value}"

    for element in "${elements[@]}"; do
        case ${element} in
            ''|*[[:space:]]*) return 1 ;;
        esac

        host=${element}
        prefix=""
        case ${element} in
            */*)
                host=${element%/*}
                prefix=${element##*/}
                case ${prefix} in
                    ''|*[!0-9]*|????*) return 1 ;;
                esac
                ((10#${prefix} <= 128)) || return 1
                ;;
        esac

        case ${host} in
            *:*)
                # IPv6: hex digits and colons, at most one "::" run.
                case ${host} in
                    *[!0-9a-fA-F:]*) return 1 ;;
                    *:::*|*::*::*) return 1 ;;
                    :*)
                        case ${host} in
                            ::*) ;;
                            *) return 1 ;;
                        esac
                        ;;
                    *:)
                        case ${host} in
                            *::) ;;
                            *) return 1 ;;
                        esac
                        ;;
                esac
                ;;
            *.*)
                # IPv4: exactly four octets, each 0-255 without a leading zero.
                case ${host} in
                    *[!0-9.]*|.*|*.) return 1 ;;
                esac
                IFS='.' read -r -a octets <<<"${host}"
                [[ ${#octets[@]} -eq 4 ]] || return 1
                for octet in "${octets[@]}"; do
                    case ${octet} in
                        ''|*[!0-9]*|????*|0?*) return 1 ;;
                    esac
                    ((10#${octet} <= 255)) || return 1
                done
                ;;
            *) return 1 ;;
        esac
    done

    return 0
}

# Validation for the keys set_env can write. Unlisted keys are accepted as-is.
valid_env_value() {
    local key=$1
    local value=$2

    case "${key}" in
        TRUSTED_PROXIES) valid_trusted_proxies "${value}" ;;
        *) return 0 ;;
    esac
}

env_hint() {
    case "$1" in
        TRUSTED_PROXIES)
            log_info "Expected comma-separated IPv4/IPv6 addresses or CIDRs, for example: 127.0.0.0/8,::1/128"
            ;;
    esac
}

set_address() {
    local new_address

    while true; do
        log_info "Set the listen address (for example: 127.0.0.1:10000 or 30492)."
        read -r -p "Address [${ADDRESS}]: " new_address
        new_address=${new_address:-${ADDRESS}}

        if valid_address "${new_address}"; then
            break
        fi
        log_warn "Invalid address. Use a port or HOST:PORT with a port between 1 and 65535."
    done

    sed -i -E "s|^([[:space:]]*-[[:space:]]*\").*(:51214\"[[:space:]]*)$|\1${new_address}\2|" "${COMPOSE_FILE}"
    ADDRESS=${new_address}
    log_info "Listen address: ${ADDRESS}"
}

read_env_value() {
    local key=$1
    awk -v key="${key}" 'index($0, key "=") == 1 { print substr($0, length(key) + 2); exit }' "${ENV_FILE}"
}

get_env() {
    local value

    [[ -f ${ENV_FILE} ]] || return 1

    value=$(read_env_value "ADMIN_NAME")
    if [[ -n ${value} ]]; then
        ADMIN_NAME=${value}
    fi
    value=$(read_env_value "ADMIN_PASSWD")
    if [[ -n ${value} ]]; then
        ADMIN_PASSWD=${value}
    fi
    value=$(read_env_value "JWT_SECRET")
    if [[ -n ${value} ]]; then
        JWT_SECRET=${value}
    fi
    value=$(read_env_value "TRUSTED_PROXIES")
    if [[ -n ${value} ]]; then
        TRUSTED_PROXIES=${value}
    fi
}

write_env_value() {
    local key=$1
    local value=$2
    local temporary
    local line
    local found=false

    temporary=$(mktemp "${ENV_FILE}.tmp.XXXXXX")
    while IFS= read -r line || [[ -n ${line} ]]; do
        if [[ ${line} == "${key}="* ]]; then
            printf '%s=%s\n' "${key}" "${value}" >>"${temporary}"
            found=true
        else
            printf '%s\n' "${line}" >>"${temporary}"
        fi
    done <"${ENV_FILE}"

    if [[ ${found} == false ]]; then
        printf '%s=%s\n' "${key}" "${value}" >>"${temporary}"
    fi

    chmod 600 "${temporary}"
    mv -f "${temporary}" "${ENV_FILE}"
}

set_env() {
    local key=$1
    local current
    local new_value

    case "${key}" in
        ADMIN_NAME) current=${ADMIN_NAME} ;;
        ADMIN_PASSWD) current=${ADMIN_PASSWD} ;;
        JWT_SECRET) current=${JWT_SECRET} ;;
        TRUSTED_PROXIES) current=${TRUSTED_PROXIES} ;;
        *) die "Unsupported configuration key: ${key}" ;;
    esac

    while true; do
        if [[ ${key} == "ADMIN_NAME" || ${key} == "TRUSTED_PROXIES" ]]; then
            read -r -p "Set ${key} (press Enter to keep the current value): " new_value
        else
            read -r -s -p "Set ${key} (press Enter to keep the current value): " new_value
            printf '\n' >&2
        fi

        # Enter keeps the current value without validating it, so a value that
        # predates this prompt can always be left alone.
        if [[ -z ${new_value} ]]; then
            new_value=${current}
            break
        fi
        if valid_env_value "${key}" "${new_value}"; then
            break
        fi

        log_warn "Invalid value for ${key}."
        env_hint "${key}"
    done

    write_env_value "${key}" "${new_value}"

    case "${key}" in
        ADMIN_NAME) ADMIN_NAME=${new_value} ;;
        ADMIN_PASSWD) ADMIN_PASSWD=${new_value} ;;
        JWT_SECRET) JWT_SECRET=${new_value} ;;
        TRUSTED_PROXIES) TRUSTED_PROXIES=${new_value} ;;
    esac

    if [[ ${key} == "ADMIN_NAME" || ${key} == "TRUSTED_PROXIES" ]]; then
        log_info "${key}: ${new_value}"
    else
        log_info "${key}: configured"
    fi
}

set_all_env() {
    set_env "ADMIN_NAME"
    set_env "ADMIN_PASSWD"
    set_env "JWT_SECRET"
    set_env "TRUSTED_PROXIES"
}

log_config() {
    log_info "Current configuration"
    printf '  ADDRESS=%s\n' "${ADDRESS}" >&2
    printf '  TRUSTED_PROXIES=%s\n' "${TRUSTED_PROXIES}" >&2
    printf '  ADMIN_NAME=%s\n' "${ADMIN_NAME}" >&2
    printf '  ADMIN_PASSWD=<configured>\n' >&2
    printf '  JWT_SECRET=<configured>\n' >&2
}

prepare_new_deployment() {
    install -d -m 0755 "${DEPLOY_DIR}" "${DEPLOY_DIR}/sql"

    log_info "Downloading deployment files"
    download_compose_file
    download_sql_init
    download_env_example
    cp "${DEPLOY_DIR}/.env.example" "${ENV_FILE}"
    chmod 600 "${ENV_FILE}"

    get_address || die "Unable to read the listen address from ${COMPOSE_FILE}."
    get_env
    chmod 600 "${ENV_FILE}"
    set_address
    set_all_env
}

install_project() {
    require_root
    require_command curl
    ensure_docker

    if [[ -f ${COMPOSE_FILE} || -f ${ENV_FILE} ]]; then
        if confirm_yes "An existing KTAUTH deployment was found. Repair and update it?"; then
            update_project
        fi
        return
    fi

    prepare_new_deployment
    compose up --detach
    log_config
    log_info "KTAUTH is running."
}

update_project() {
    require_root
    require_command curl
    ensure_docker
    install -d -m 0755 "${DEPLOY_DIR}" "${DEPLOY_DIR}/sql"

    if [[ ! -f ${COMPOSE_FILE} ]]; then
        log_warn "${COMPOSE_FILE} is missing; downloading it."
        download_compose_file
        get_address || die "Unable to read the listen address from ${COMPOSE_FILE}."
        set_address
    fi

    if [[ ! -f ${DEPLOY_DIR}/sql/00-init.sql ]]; then
        log_warn "${DEPLOY_DIR}/sql/00-init.sql is missing; downloading it."
        download_sql_init
    fi

    if [[ ! -f ${ENV_FILE} ]]; then
        log_warn "${ENV_FILE} is missing; creating it from the latest example."
        download_env_example
        cp "${DEPLOY_DIR}/.env.example" "${ENV_FILE}"
        chmod 600 "${ENV_FILE}"
        get_env
        set_all_env
    fi

    get_address || die "Unable to read the listen address from ${COMPOSE_FILE}."
    get_env
    chmod 600 "${ENV_FILE}"

    # Complete all network and file preparation before stopping the current service.
    docker pull "${IMAGE}"
    compose down || log_warn "KTAUTH was not running or could not be stopped cleanly."

    log_config
    compose up --detach
    log_info "KTAUTH was updated successfully."
}

configure_project() {
    require_root
    [[ -f ${COMPOSE_FILE} ]] || die "Deployment file not found: ${COMPOSE_FILE}"
    [[ -f ${ENV_FILE} ]] || die "Environment file not found: ${ENV_FILE}"

    get_address || die "Unable to read the listen address from ${COMPOSE_FILE}."
    get_env
    chmod 600 "${ENV_FILE}"
    log_config

    if ! confirm_yes "Change these settings?"; then
        return
    fi

    set_address
    set_all_env
    log_config

    if confirm_yes "Restart KTAUTH now?"; then
        require_docker
        compose restart
    fi
}

uninstall_project() {
    require_root
    require_docker
    [[ -f ${COMPOSE_FILE} ]] || die "Deployment file not found: ${COMPOSE_FILE}"

    if ! confirm_yes "Uninstall KTAUTH?"; then
        return
    fi

    if confirm_no "Remove all database and Redis data?"; then
        compose down --volumes
    else
        compose down
    fi

    if confirm_no "Remove the KTAUTH image?"; then
        docker image rm "${IMAGE}" || log_warn "The KTAUTH image is already absent or still in use."
    fi

    if confirm_no "Remove the postgres:latest and redis:latest images?"; then
        docker image rm postgres:latest redis:latest || log_warn "One or more dependency images are absent or still in use."
    fi

    log_info "KTAUTH was uninstalled successfully."
    log_warn "Make sure Caddy or Nginx no longer forwards authentication requests to KTAUTH."
}

usage() {
    printf '%s\n' "\
Usage: $0 <command>

Commands:
  install     Install KTAUTH or repair an existing deployment
  update      Pull the latest image and restart the deployment
  config      Update the listen address, core credentials and trusted proxies
  uninstall   Stop and uninstall the deployment"
}

case "${1:-}" in
    install) install_project ;;
    update) update_project ;;
    config) configure_project ;;
    uninstall) uninstall_project ;;
    -h|--help|help) usage ;;
    *)
        usage >&2
        exit 2
        ;;
esac
