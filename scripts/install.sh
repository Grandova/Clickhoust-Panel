#!/usr/bin/env bash
# ==============================================================================
# ClickHouse Manager - Linux One-Click Production Installer
# ==============================================================================
set -e

COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[1;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'

echo -e "${COLOR_CYAN}"
echo "============================================================"
echo "          Installing ClickHouse Manager                      "
echo "============================================================"
echo -e "${COLOR_RESET}"

# Check root privilege
if [ "$(id -u)" -ne 0 ]; then
    echo -e "${COLOR_RED}[Error] This script must be executed as root. Please run with sudo.${COLOR_RESET}"
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

echo -e "${COLOR_GREEN}[1/7] Detected architecture: ${ARCH} (${BIN_ARCH})${COLOR_RESET}"

# Detect OS Distribution
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS_NAME=$ID
    OS_PRETTY=$PRETTY_NAME
else
    echo -e "${COLOR_RED}[Error] Cannot detect operating system release info.${COLOR_RESET}"
    exit 1
fi

echo -e "${COLOR_GREEN}[2/7] Detected OS: ${OS_PRETTY}${COLOR_RESET}"

# Install prerequisites
echo -e "${COLOR_GREEN}[3/7] Installing basic dependencies (curl, tar)...${COLOR_RESET}"
if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y && apt-get install -y curl tar ca-certificates
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y curl tar ca-certificates
elif command -v yum >/dev/null 2>&1; then
    yum install -y curl tar ca-certificates
fi

# Create system user & directory
INSTALL_DIR="/opt/clickhouse-manager"
DATA_DIR="${INSTALL_DIR}/data"
LOG_DIR="${INSTALL_DIR}/logs"
PORT=${PORT:-8080}

echo -e "${COLOR_GREEN}[4/7] Preparing directory structure at ${INSTALL_DIR}...${COLOR_RESET}"
mkdir -p "${DATA_DIR}" "${LOG_DIR}"

if ! id "clickhouse-manager" >/dev/null 2>&1; then
    useradd -r -s /bin/false clickhouse-manager || true
fi

# Download or copy binary
BINARY_PATH="${INSTALL_DIR}/clickhouse-manager"
echo -e "${COLOR_GREEN}[5/7] Deploying ClickHouse Manager binary...${COLOR_RESET}"

# If running from local source repository, build or copy
if [ -f "./backend/clickhouse-manager" ]; then
    cp "./backend/clickhouse-manager" "${BINARY_PATH}"
elif [ -f "./clickhouse-manager" ]; then
    cp "./clickhouse-manager" "${BINARY_PATH}"
else
    # In online release environment, download from GitHub release asset:
    RELEASE_TAG=$(curl -fsSL "https://api.github.com/repos/Grandova/Clickhoust-Panel/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
    if [ -z "${RELEASE_TAG}" ]; then
        RELEASE_TAG="v1.2.0"
    fi
    DOWNLOAD_URL="https://github.com/Grandova/Clickhoust-Panel/releases/download/${RELEASE_TAG}/clickhouse-manager-${RELEASE_TAG}-linux-${BIN_ARCH}.tar.gz"
    echo -e "${COLOR_GREEN}Downloading ${RELEASE_TAG} binary from ${DOWNLOAD_URL}...${COLOR_RESET}"
    TEMP_TAR=$(mktemp)
    curl -fsSL "${DOWNLOAD_URL}" -o "${TEMP_TAR}"
    tar -xzf "${TEMP_TAR}" -C "${INSTALL_DIR}"
    rm -f "${TEMP_TAR}"
fi

if [ ! -f "${BINARY_PATH}" ]; then
    echo -e "${COLOR_RED}[Error] Failed to install binary at ${BINARY_PATH}.${COLOR_RESET}"
    exit 1
fi

chmod +x "${BINARY_PATH}"
chown -R clickhouse-manager:clickhouse-manager "${INSTALL_DIR}"

# Create systemd service
SERVICE_FILE="/etc/systemd/system/clickhouse-manager.service"
echo -e "${COLOR_GREEN}[6/7] Creating systemd service at ${SERVICE_FILE}...${COLOR_RESET}"

cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=ClickHouse Manager Web Administration Panel
After=network.target network-online.target clickhouse-server.service
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${BINARY_PATH} -port ${PORT} -db ${DATA_DIR}/clickhouse-manager.db
Restart=always
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Reload and start service
echo -e "${COLOR_GREEN}[7/7] Starting clickhouse-manager.service...${COLOR_RESET}"
systemctl daemon-reload
systemctl enable clickhouse-manager.service
systemctl restart clickhouse-manager.service

# Get Public or Local IP
SERVER_IP=$(curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')

echo ""
echo -e "${COLOR_GREEN}============================================================${COLOR_RESET}"
echo -e "${COLOR_GREEN}        ClickHouse Manager successfully installed!         ${COLOR_RESET}"
echo -e "${COLOR_GREEN}============================================================${COLOR_RESET}"
echo ""
echo -e "  Web Panel URL:   ${COLOR_CYAN}http://${SERVER_IP}:${PORT}${COLOR_RESET}"
echo -e "  Default Username: admin"
echo -e "  Default Password: ${COLOR_YELLOW}admin123456${COLOR_RESET}"
echo -e "  (You will be required to change the password on first login)"
echo ""
echo -e "  Service Commands:"
echo -e "    systemctl status clickhouse-manager"
echo -e "    systemctl restart clickhouse-manager"
echo -e "    journalctl -u clickhouse-manager -f"
echo ""
echo -e "  Update Command:"
echo -e "    curl -fsSL https://raw.githubusercontent.com/Grandova/Clickhoust-Panel/main/scripts/update.sh | bash"
echo ""
echo -e "${COLOR_GREEN}============================================================${COLOR_RESET}"
