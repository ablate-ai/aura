#!/bin/sh
# Aura 探针监控面板安装脚本
# 用法: sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"
# 国内加速: sh -c "$(curl -sfL https://ghfast.top/https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"

set -e

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
    C_INFO='\033[1;34m'
    C_WARN='\033[1;33m'
    C_ERROR='\033[1;31m'
    C_RESET='\033[0m'
else
    C_INFO=''
    C_WARN=''
    C_ERROR=''
    C_RESET=''
fi

info() { printf '%b\n' "${C_INFO}[INFO]${C_RESET} $1"; }
warn() { printf '%b\n' "${C_WARN}[WARN]${C_RESET} $1"; }
error() { printf '%b\n' "${C_ERROR}[ERROR]${C_RESET} $1"; exit 1; }

# 配置
REPO="ablate-ai/aura"
GITHUB_URL="https://github.com/${REPO}"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="aura"

# 检测系统架构
info "检测系统架构..."
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

case "$os" in
    linux)   os_name="Linux" ;;
    darwin)  os_name="Darwin" ;;
    *)       error "不支持的操作系统: $os" ;;
esac

case "$arch" in
    x86_64|amd64)   arch_name="x86_64" ;;
    aarch64|arm64)  arch_name="arm64" ;;
    armv7l|arm)     arch_name="armv7" ;;
    *)              error "不支持的架构: $arch" ;;
esac

info "系统: $os_name, 架构: $arch_name"

# 获取最新版本
info "获取最新版本..."
version_url=$(curl -sSfL "${API_URL}" | \
    grep "browser_download_url.*${os_name}_${arch_name}" | \
    grep -v ".sha256" | \
    grep -v ".txt" | \
    head -n 1 | \
    cut -d '"' -f 4)

[ -z "$version_url" ] && error "无法找到匹配的二进制文件"

# 如果设置了镜像，下载地址拼接镜像前缀
if [ -n "$GITHUB_MIRROR" ]; then
    version_url="${GITHUB_MIRROR}/${version_url}"
fi

version=$(echo "$version_url" | grep -oP 'tag/\K[^/]*' || echo "latest")
info "版本: $version"

# 下载二进制文件
info "下载二进制文件..."
tmp_dir=$(mktemp -d)
trap "rm -rf $tmp_dir" EXIT

curl -fL --progress-bar "$version_url" -o "${tmp_dir}/${BINARY_NAME}.tar.gz"
info "下载完成"

# 解压并安装
info "安装到 ${INSTALL_DIR}..."
tar -xzf "${tmp_dir}/${BINARY_NAME}.tar.gz" -C "$tmp_dir"

# 停止正在运行的服务（避免 Text file busy）
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet aura 2>/dev/null; then
    info "停止运行中的 aura 服务..."
    systemctl stop aura
fi

# 检查是否需要 sudo
if [ -w "$INSTALL_DIR" ]; then
    install_cmd="cp"
else
    install_cmd="sudo cp"
    info "需要 sudo 权限来安装到 ${INSTALL_DIR}"
fi

$install_cmd "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

# 设置可执行权限
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

info "安装完成!"

# 创建配置文件
CONFIG_DIR="/etc/aura"
CONFIG_FILE="${CONFIG_DIR}/env"

if [ ! -f "$CONFIG_FILE" ]; then
    info "创建配置文件 ${CONFIG_FILE}"
    if [ -w "/etc" ]; then
        mkdir -p "$CONFIG_DIR"
    else
        sudo mkdir -p "$CONFIG_DIR"
    fi

    cat > "${tmp_dir}/aura.env" <<EOF
# Aura 配置文件
# Prometheus 地址
PROM_BASEURL="${PROM_BASEURL:-http://prom.ooxo.cc/}"

# 监听端口
PORT="${PORT:-8080}"
EOF

    $install_cmd "${tmp_dir}/aura.env" "$CONFIG_FILE"
    chmod 644 "$CONFIG_FILE"
fi

# 创建 systemd 服务
SYSTEMD_DIR="/etc/systemd/system"
SERVICE_FILE="${SYSTEMD_DIR}/aura.service"

if [ "$os" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
    info "创建 systemd 服务..."

    cat > "${tmp_dir}/aura.service" <<EOF
[Unit]
Description=Aura Probe Monitor
After=network.target

[Service]
Type=simple
EnvironmentFile=${CONFIG_FILE}
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    if [ -w "$SYSTEMD_DIR" ]; then
        cp "${tmp_dir}/aura.service" "$SERVICE_FILE"
    else
        sudo cp "${tmp_dir}/aura.service" "$SERVICE_FILE"
    fi

    systemctl daemon-reload
    systemctl enable aura
    systemctl restart aura

    info ""
    info "============================================"
    info "  安装完成!"
    info "============================================"
    info ""
    info "配置文件: ${CONFIG_FILE}"
    info "启动服务: systemctl start aura"
    info "开机自启: systemctl enable aura"
    info "查看状态: systemctl status aura"
    info "查看日志: journalctl -u aura -f"
    info ""
    info "访问地址: http://localhost:${PORT:-8080}"
    info "============================================"
else
    info ""
    info "============================================"
    info "  安装完成!"
    info "============================================"
    info ""
    info "配置文件: ${CONFIG_FILE}"
    info ""
    info "启动服务:"
    info "  ${BINARY_NAME}"
    info ""
    info "或指定配置:"
    info "  PROM_BASEURL=http://your-prom/ ${BINARY_NAME}"
    info "============================================"
fi

# 显示版本
${INSTALL_DIR}/${BINARY_NAME} -version
