#!/bin/sh
# node_exporter 安装脚本
# 用法: sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install_node_exporter.sh)"
# 国内加速: sh -c "$(curl -sfL https://ghfast.top/https://raw.githubusercontent.com/ablate-ai/aura/main/install_node_exporter.sh)"

set -e

info() { echo "[INFO] $1"; }
warn() { echo "[WARN] $1"; }
error() { echo "[ERROR] $1"; exit 1; }

# 配置
REPO="prometheus/node_exporter"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="node_exporter"
SERVICE_USER="node_exporter"

# 仅支持 Linux
os=$(uname -s | tr '[:upper:]' '[:lower:]')
[ "$os" != "linux" ] && error "node_exporter 安装脚本仅支持 Linux"

# 检测系统架构
info "检测系统架构..."
arch=$(uname -m)

case "$arch" in
    x86_64|amd64)   arch_name="amd64" ;;
    aarch64|arm64)  arch_name="arm64" ;;
    armv7l|arm)     arch_name="armv7" ;;
    *)              error "不支持的架构: $arch" ;;
esac

info "架构: $arch_name"

# 获取最新版本
info "获取最新版本..."
version_url=$(curl -sSfL "${API_URL}" | \
    grep "browser_download_url.*linux-${arch_name}" | \
    grep -v ".sha256" | \
    head -n 1 | \
    cut -d '"' -f 4)

[ -z "$version_url" ] && error "无法找到匹配的二进制文件"

# 如果设置了镜像，下载地址拼接镜像前缀
if [ -n "$GITHUB_MIRROR" ]; then
    version_url="${GITHUB_MIRROR}/${version_url}"
fi

version=$(echo "$version_url" | sed 's|.*/download/\([^/]*\)/.*|\1|')
info "版本: $version"

# 下载
info "下载 node_exporter..."
tmp_dir=$(mktemp -d)
trap "rm -rf $tmp_dir" EXIT

curl -fL --progress-bar "$version_url" -o "${tmp_dir}/node_exporter.tar.gz"
info "下载完成"

# 解压
tar -xzf "${tmp_dir}/node_exporter.tar.gz" -C "$tmp_dir" --strip-components=1
info "解压完成"

# 停止正在运行的服务
if systemctl is-active --quiet node_exporter 2>/dev/null; then
    info "停止运行中的 node_exporter 服务..."
    systemctl stop node_exporter
fi

# 安装二进制
if [ -w "$INSTALL_DIR" ]; then
    cp "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
    sudo cp "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
info "安装到 ${INSTALL_DIR}/${BINARY_NAME}"

# 创建系统用户（无登录权限）
if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    info "创建系统用户 ${SERVICE_USER}..."
    if command -v useradd >/dev/null 2>&1; then
        useradd --no-create-home --shell /bin/false "$SERVICE_USER" 2>/dev/null || \
        sudo useradd --no-create-home --shell /bin/false "$SERVICE_USER"
    fi
fi

# 创建 systemd 服务
SYSTEMD_DIR="/etc/systemd/system"
SERVICE_FILE="${SYSTEMD_DIR}/node_exporter.service"

if command -v systemctl >/dev/null 2>&1; then
    info "创建 systemd 服务..."

    cat > "${tmp_dir}/node_exporter.service" <<EOF
[Unit]
Description=Prometheus Node Exporter
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    if [ -w "$SYSTEMD_DIR" ]; then
        cp "${tmp_dir}/node_exporter.service" "$SERVICE_FILE"
    else
        sudo cp "${tmp_dir}/node_exporter.service" "$SERVICE_FILE"
    fi

    systemctl daemon-reload
    systemctl enable node_exporter
    systemctl restart node_exporter

    info ""
    info "============================================"
    info "  node_exporter 安装完成!"
    info "============================================"
    info ""
    info "监听地址: http://localhost:9100/metrics"
    info "查看状态: systemctl status node_exporter"
    info "查看日志: journalctl -u node_exporter -f"
    info "============================================"
else
    info ""
    info "============================================"
    info "  node_exporter 安装完成!"
    info "============================================"
    info ""
    info "手动启动: ${INSTALL_DIR}/${BINARY_NAME}"
    info "监听地址: http://localhost:9100/metrics"
    info "============================================"
fi

# 显示版本
${INSTALL_DIR}/${BINARY_NAME} --version
