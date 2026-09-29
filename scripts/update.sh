#!/usr/bin/env bash
# ==============================================================================
# ClickHouse Manager - Linux One-Click Production Updater
# Repository: https://github.com/Grandova/Clickhoust-Panel
# ==============================================================================
set -e

COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[1;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'

echo -e "${COLOR_CYAN}"
echo "============================================================"
echo "          Updating ClickHouse Manager                       "
echo "============================================================"
echo -e "${COLOR_RESET}"

# Check root privilege
if [ "$(id -u)" -ne 0 ]; then
    echo -e "${COLOR_RED}[Error] This script must be executed as root. Please run with sudo.${COLOR_RESET}"
    exit 1
fi

INSTALL_DIR="/opt/clickhouse-manager"
BINARY_PATH="${INSTALL_DIR}/clickhouse-manager"
BACKUP_PATH="${INSTALL_DIR}/clickhouse-manager.bak"

if [ ! -d "${INSTALL_DIR}" ]; then
    echo -e "${COLOR_RED}[Error] Installation directory ${INSTALL_DIR} does not exist.${COLOR_RESET}"
    echo -e "${COLOR_YELLOW}Please install ClickHouse Manager first via install.sh${COLOR_RESET}"
    exit 1
fi

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        BIN_ARCH="amd64"
        ;;
    aarch64|arm64)
        BIN_ARCH="arm64"
        ;;
    *)
        echo -e "${COLOR_RED}[Error] Unsupported architecture: ${ARCH}${COLOR_RESET}"
        exit 1
        ;;
esac

echo -e "${COLOR_GREEN}[1/5] Detected architecture: ${ARCH} (${BIN_ARCH})${COLOR_RESET}"

# Fetch latest release tag from GitHub API
echo -e "${COLOR_GREEN}[2/5] Checking for latest release from GitHub...${COLOR_RESET}"
RELEASE_TAG=$(curl -fsSL "https://api.github.com/repos/Grandova/Clickhoust-Panel/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)

if [ -z "${RELEASE_TAG}" ]; then
    echo -e "${COLOR_RED}[Error] Could not determine the latest release. No files were changed.${COLOR_RESET}"
    exit 1
fi

echo -e "${COLOR_CYAN}Target release version: ${RELEASE_TAG}${COLOR_RESET}"

# Create temp download directory
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT

DOWNLOAD_URL="https://github.com/Grandova/Clickhoust-Panel/releases/download/${RELEASE_TAG}/clickhouse-manager-${RELEASE_TAG}-linux-${BIN_ARCH}.tar.gz"

echo -e "${COLOR_GREEN}[3/5] Downloading latest release from ${DOWNLOAD_URL}...${COLOR_RESET}"
curl -fsSL "${DOWNLOAD_URL}" -o "${TEMP_DIR}/release.tar.gz"

echo -e "${COLOR_GREEN}[4/5] Extracting release files...${COLOR_RESET}"
tar -xzf "${TEMP_DIR}/release.tar.gz" -C "${TEMP_DIR}"

if [ ! -f "${TEMP_DIR}/clickhouse-manager" ]; then
    echo -e "${COLOR_RED}[Error] Extracted archive did not contain clickhouse-manager executable.${COLOR_RESET}"
    exit 1
fi

# Backup current binary
if [ -f "${BINARY_PATH}" ]; then
    cp -f "${BINARY_PATH}" "${BACKUP_PATH}"
fi

# Rename the staged binary to avoid overwriting the running executable.
echo -e "${COLOR_GREEN}[5/5] Upgrading binary and restarting service...${COLOR_RESET}"
install -m 755 "${TEMP_DIR}/clickhouse-manager" "${BINARY_PATH}.new"
chown clickhouse-manager:clickhouse-manager "${BINARY_PATH}.new" 2>/dev/null || true
mv -f "${BINARY_PATH}.new" "${BINARY_PATH}"

if systemctl is-enabled clickhouse-manager.service >/dev/null 2>&1 || systemctl is-active --quiet clickhouse-manager.service; then
    if systemctl restart clickhouse-manager.service && sleep 2 && systemctl is-active --quiet clickhouse-manager.service; then
        rm -f "${BACKUP_PATH}"
        echo ""
        echo -e "${COLOR_GREEN}============================================================${COLOR_RESET}"
        echo -e "${COLOR_GREEN}    ClickHouse Manager successfully updated to ${RELEASE_TAG}! ${COLOR_RESET}"
        echo -e "${COLOR_GREEN}============================================================${COLOR_RESET}"
        systemctl status clickhouse-manager.service --no-pager -n 5
    else
        echo -e "${COLOR_RED}[Warning] Service failed to restart with new binary. Rolling back...${COLOR_RESET}"
        if [ -f "${BACKUP_PATH}" ]; then
            install -m 755 "${BACKUP_PATH}" "${BINARY_PATH}.new"
            mv -f "${BINARY_PATH}.new" "${BINARY_PATH}"
            systemctl restart clickhouse-manager.service
            echo -e "${COLOR_YELLOW}Rolled back to previous working version.${COLOR_RESET}"
        fi
        exit 1
    fi
else
    echo -e "${COLOR_GREEN}Binary updated at ${BINARY_PATH}.${COLOR_RESET}"
fi
