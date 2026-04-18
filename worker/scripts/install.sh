#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_NAME='ktauth'
PROJECT_BRANCH='main'
DOWNLOAD_URL="https://raw.githubusercontent.com/stellashiina/${PROJECT_NAME}/${PROJECT_BRANCH}"
DEPLOY_DIR="/opt/${PROJECT_NAME}"
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

update() {
    if [[ ! -f "${DEPLOY_DIR}/.PORT" ]]; then
        touch 644 "${DEPLOY_DIR}/.PORT"
        echo "10000" > "${DEPLOY_DIR}/.PORT"
    fi
}

deploy() {
    if [[ -d "${DEPLOY_DIR}" ]]; then
        if yesno "It seems that you've already install ktauth, do you want to update?"; then
            update
            docker compose -f "${DEPLOY_DIR}/docker-compose.yaml" down
            docker pull stellashiina/ktauth:latest
        else
            exit 0
        fi
    else
        mkdir -p 755 "${DEPLOY_DIR}/init"
        echo 51214 > "${DEPLOY_DIR}/.PORT"
    fi

    log_info "Downloading project files"
    
    curl -fsSL -o "${DEPLOY_DIR}/docker-compose.yaml" "${DOWNLOAD_URL}/docker-compose.yaml"
    log_info "${DOWNLOAD_URL}/docker-compose.yaml -> ${DEPLOY_DIR}/docker-compose.yaml"

    curl -fsSL -o "${DEPLOY_DIR}/init/00-init.sql" "${DOWNLOAD_URL}/init/00-init.sql"
    log_info "${DOWNLOAD_URL}/init/00-init.sql -> ${DEPLOY_DIR}/init/00-init.sql"



    if [[ ! -f "${DEPLOY_DIR}/.env" ]]; then
        curl -fsSL -o "${DEPLOY_DIR}/.env.example" "${DOWNLOAD_URL}/.env.example"
        log_info "${DOWNLOAD_URL}/.env.example -> ${DEPLOY_DIR}/.env.example"
        cp "${DEPLOY_DIR}/.env.example" "${DEPLOY_DIR}/.env"
        log_info "${DEPLOY_DIR}/.env.example -> ${DEPLOY_DIR}/.env"
        config
    else
        if noyes "Overwrite current .env with latest .env.example?"; then
            curl -fsSL -o "${DEPLOY_DIR}/.env" "${DOWNLOAD_URL}/.env.example"
            log_info "${DOWNLOAD_URL}/.env.example -> ${DEPLOY_DIR}/.env"
            config
        fi
    fi

    cd "${DEPLOY_DIR}"

    docker compose up -d

    log_info "Success! Ktauth is running on port ${ADDRESS}..."
}

config() {
    CUR_ADDRESS=$(cat "${DEPLOY_DIR}/.PORT")
    read -r -p "Set admin username (Press enter to use default/current): " admin_name
    if [[ -z "${admin_name//[[:space:]]/}" ]]; then
        log_warn "Empty input, use default/current username"
    else
        sed -i "s/^ADMIN_NAME=.*/ADMIN_NAME=\"${admin_name}\"/" "${DEPLOY_DIR}/.env"       
    fi

    read -r -p "Set admin password (Press enter to use default/current): " admin_passwd
    if [[ -z "${admin_passwd//[[:space:]]/}" ]]; then
        log_warn "Empty input, use default/current password"
    else 
        sed -i "s/^ADMIN_PASSWD=.*/ADMIN_PASSWD=\"${admin_passwd}\"/" "${DEPLOY_DIR}/.env"
    fi  

    read -r -p "Set JWT Secret (Press enter to use default/current): " jwt_secret
    if [[ -z "${jwt_secret//[[:space:]]/}" ]]; then
        log_warn "Empty input, use default/current jwt secret"
    else
        sed -i "s/^JWT_SECRET=.*/JWT_SECRET=\"${jwt_secret}\"/" "${DEPLOY_DIR}/.env"
    fi

    read -r -p "Set ktauth listen addree (IP:Port or Port, eg. 127.0.0.1:10000, 30492)(Press enter to use default/current [${CUR_ADDRESS}]): " ADDRESS
    if [[ -z "${ADDRESS//[[:space:]]/}" ]]; then
        ADDRESS="${CUR_ADDRESS}"
        log_warn "Empty input, use default/current listen port"
    else
        sed -i "s|^\( *- \"\).*\(:51214\"\)$|\1${ADDRESS}\2|" "${DEPLOY_DIR}/docker-compose.yaml"
    fi

    echo "$ADDRESS" > "${DEPLOY_DIR}/.PORT"
}

uninstall() {
    if yesno "Sure to uninstall?"; then
        if noyes "Remove all data?"; then
            docker compose -f "${DEPLOY_DIR}/docker-compose.yaml" down -v
        else
            docker compose -f "${DEPLOY_DIR}/docker-compose.yaml" down
        fi
        docker image rm stellashiina/ktauth:latest || log_error "Maybe already removed or in use?"
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

case "$1" in
    "uninstall")
        uninstall
        exit 0
        ;;
    "config")
        config
        cd "${DEPLOY_DIR}"
        docker compose down && docker compose up -d
        exit 0
        ;;
    "install")
        main
        exit 0
        ;;
    *)
        echo "Usage: $0 [command]"
        echo "Commands:"
        echo "  uninstall   uninstall ktauth"
        echo "  config      update configuration"
        echo "  install     install or update"
        exit 0
        ;;
esac