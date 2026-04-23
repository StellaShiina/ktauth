#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_NAME='ktauth'
DOWNLOAD_URL="https://ktauth.kaju.win"
DEPLOY_DIR="/opt/${PROJECT_NAME}"

# Configuration
ADMIN_NAME='ktauth'
ADMIN_PASSWD='ktauth'
JWT_SECRET='ktauthsecret'
ADDRESS='51214'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_configs() {
    echo "ADDRESS=${ADDRESS}"
    echo "ADMIN_NAME=${ADMIN_NAME}"
    echo "ADMIN_PASSWD=${ADMIN_PASSWD}"
    echo "JWT_SECRET=${JWT_SECRET}"
}

yesno() {
    log_info "$1[${GREEN}Y${NC}/n]: "
    read -r response
    case "$response" in
        [nN][oO]|[nN])
            return 1
            ;;
        *)
            return 0
            ;;
    esac
}

noyes() {
    log_info "$1[y/${RED}N${NC}]: "
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

check_cmd() {
    if ! command -v $1 >/dev/null 2>&1; then
        return 1
    fi
    return 0
}

install_docker() {
    if ! (docker version && docker compose version) &>/dev/null; then

        if yesno "Install docker and docker compose?"; then
            sh <(curl -fsSL https://get.docker.com -o get-docker.sh)
        else
            log_warn "Please install docker & docker compose and run this script again..."
            exit 0
        fi
    fi
}

get_address() {
    if [[ -f "${DEPLOY_DIR}/docker-compose.yaml" ]]; then
        cd "${DEPLOY_DIR}"
        ADDRESS=$(awk -F'[:"]' '/^ *- "/ && /:51214"/ {print $2}' docker-compose.yaml)
        return 0
    else
        return 1
    fi
}

set_address() {
    log_info "Set ktauth listen address (IP:Port or Port, eg. 127.0.0.1:10000, 30492)"
    read -r -p "Press enter to use default/current [${ADDRESS}]: " new_address
    if [[ ! -z "${new_address//[[:space:]]/}" ]]; then
        ADDRESS="${new_address}"
        sed -i -E "s|^( *- \").*(:51214\")$|\1${ADDRESS}\2|" "${DEPLOY_DIR}/docker-compose.yaml"
    fi
    log_info "Address is set: ${ADDRESS}"
}

get_env() {
    if [[ -f "${DEPLOY_DIR}/.env" ]]; then
        cd "${DEPLOY_DIR}"
        ADMIN_NAME=$(awk -F'=' '$1=="ADMIN_NAME" {print $2}' .env)
        ADMIN_PASSWD=$(awk -F'=' '$1=="ADMIN_PASSWD" {print $2}' .env)
        JWT_SECRET=$(awk -F'=' '$1=="JWT_SECRET" {print $2}' .env)
        return 0
    else
        return 1
    fi
}

set_env() {
    ENV_NAME="$1"
    read -r -p "Set ${ENV_NAME} (Press enter to use default/current [${!ENV_NAME}]): " new_env
    if [[ ! -z "${new_env//[[:space:]]/}" ]]; then
        eval "$ENV_NAME=$new_env"
        sed -i "s|^${ENV_NAME}=.*|${ENV_NAME}=${new_env}|" "${DEPLOY_DIR}/.env"
    fi
    log_info "${ENV_NAME} is set to: ${!ENV_NAME}"
}

set_all_env() {
    set_env ADMIN_NAME
    set_env ADMIN_PASSWD
    set_env JWT_SECRET
}

dl_dockercomposeyaml() {
    curl -fsSL -o "${DEPLOY_DIR}/docker-compose.yaml" "${DOWNLOAD_URL}/docker-compose.yaml"
    log_info "${DOWNLOAD_URL}/docker-compose.yaml -> ${DEPLOY_DIR}/docker-compose.yaml"
}

dl_sqlinit() {
    curl -fsSL -o "${DEPLOY_DIR}/sql/00-init.sql" "${DOWNLOAD_URL}/00-init.sql"
    log_info "${DOWNLOAD_URL}/00-init.sql -> ${DEPLOY_DIR}/sql/00-init.sql"
}

dl_envexample() {
    curl -fsSL -o "${DEPLOY_DIR}/.env.example" "${DOWNLOAD_URL}/.env.example"
    log_info "${DOWNLOAD_URL}/.env.example -> ${DEPLOY_DIR}/.env.example"
}

fix_and_update() {
    # to solve conflict
    docker compose -p ktauth down || log_warn "Stop ktauth failed, maybe not running?"
    docker pull stellashiina/ktauth:latest

    # check
    if ! get_address; then
        log_warn "${DEPLOY_DIR}/docker-compose.yaml not found, downloading latest version"
        dl_dockercomposeyaml
        set_address
    fi

    if ! get_env; then
        log_warn "${DEPLOY_DIR}/.env not found, downloading latest version"
        dl_envexample
        cp "${DEPLOY_DIR}/.env.example" "${DEPLOY_DIR}/.env"
        set_all_env
    fi

    log_info "Current settings"
    log_configs

    docker compose -f "${DEPLOY_DIR}/docker-compose.yaml" up -d

    exit 0
}

config() {
    if ! get_address; then
        log_warn "${DEPLOY_DIR}/docker-compose.yaml not found"
        exit 1
    fi
    if ! get_env; then
        log_warn "${DEPLOY_DIR}/.env not found"
        exit 1
    fi

    log_info "Current settings"
    log_configs

    if ! yesno "Change settings?"; then
        exit 0
    fi

    set_address
    set_all_env

    log_info "New settings"
    log_configs

    if yesno "Restart ktauth now?"; then
        docker compose -p ktauth restart
    fi

    exit 0
}

deploy() {
    if [[ -d "${DEPLOY_DIR}" ]]; then
        if yesno "It seems that you've already install ktauth, do you want to fix and update?"; then
            fix_and_update
        else
            exit 0
        fi
    fi

    mkdir -p 755 "${DEPLOY_DIR}/sql"

    cd "${DEPLOY_DIR}"

    log_info "Downloading project files"
    
    dl_dockercomposeyaml

    dl_sqlinit

    dl_envexample

    cp .env.example .env

    get_address || log_error "Failed to read ${DEPLOY_DIR}/docker-compose.yaml"

    get_env || log_error "Failed to read ${DEPLOY_DIR}/.env"

    set_address

    set_all_env

    docker compose up -d
    
    log_info "Current settings"
    log_configs
}

uninstall() {
    if yesno "Sure to uninstall?"; then
        if noyes "Remove all data?"; then
            docker compose -p ktauth down -v || log_warn "Stop ktauth failed, maybe not running?"
        else
            docker compose -p ktauth down || log_warn "Stop ktauth failed, maybe not running?"
        fi
        if noyes "Remove ktauth image?"; then
            docker image rm stellashiina/ktauth:latest || log_error "Maybe already removed or in use?"
        fi
        if noyes "Remove postgres:latest and redis:latest image?"; then
            docker image rm postgres:latest redis:latest || log_error "Maybe already removed or in use?"
        fi
        docker network prune -f
    else
        exit 0
    fi
    log_info "Successfully uninstall ktauth!"
    log_warn "Make sure your caddy/nginx no longer forward_auth to ktauth!"
}

main() {
    if ! check_cmd curl; then
        log_warn "Please make sure you've installed curl"
        exit 0
    fi

    install_docker
    
    deploy
}

add_whitelist() {
    get_address
    get_env
    host="${ADDRESS}"
    if [[ ! host =~ ':' ]]; then
        host="127.0.0.1:${ADDRESS}"
    fi
    # login
    token=$(curl -fsSL -X POST \
        -d @- \
        "${host}/api/users/login?format=string" <<EOF
{"user":"${ADMIN_NAME}","password":"${ADMIN_PASSWD}"}
EOF
)
    # listips
    ips=$(curl -sSL -H "Authorization: Bearer ${token}" \
        "${host}/api/ips")
    log_info 'current ip whitelist'
    if check_cmd jq; then
        echo "$ips" | jq '.rules[].IPCIDR'
    else
        log_info 'install jq to get prettier output'
        echo "$ips"
    fi
    # addip
    while yesno "Continue to add whitelist IP?"; do
        read -r -p "Input IP: " ip
        res=$(curl -sSL -X POST \
            -H "Authorization: Bearer ${token}" \
            -d @- \
            "${host}/api/ips/new" <<EOF
{"ip":"${ip}"}
EOF
)
        echo $res
    done
    # logout
    curl -fsSL -H "Authorization: Bearer ${token}" \
        "${host}/api/users/logout"
}

case "$1" in
    "uninstall")
        uninstall
        exit 0
        ;;
    "update")
        fix_and_update
        exit 0
        ;;
    "config")
        config
        exit 0
        ;;
    "install")
        main
        exit 0
        ;;
    "allow")
        add_whitelist
        exit 0
        ;;
    *)
        echo "Usage: $0 [command]"
        echo "Commands:"
        echo "  install     install or update"
        echo "  uninstall   uninstall ktauth"
        echo "  update      update ktauth"
        echo "  config      update configuration"
        echo "  allow       add acl whitelist"
        exit 0
        ;;
esac