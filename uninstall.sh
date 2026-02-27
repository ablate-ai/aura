#!/bin/sh
# Aura 探针监控面板卸载脚本
# 用法: sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/uninstall.sh)"
# 国内加速: sh -c "$(curl -sfL https://ghfast.top/https://raw.githubusercontent.com/ablate-ai/aura/main/uninstall.sh)"
# 删除配置: REMOVE_CONFIG=1 sh uninstall.sh

set -e

info() { echo "[INFO] $1"; }
warn() { echo "[WARN] $1"; }
error() { echo "[ERROR] $1"; exit 1; }

BINARY_PATH="/usr/local/bin/aura"
CONFIG_DIR="/etc/aura"
SERVICE_FILE="/etc/systemd/system/aura.service"

remove_with_sudo_if_needed() {
    target="$1"

    if [ ! -e "$target" ]; then
        return 0
    fi

    if rm -rf "$target" 2>/dev/null; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo rm -rf "$target"
    else
        error "没有权限删除 $target，请使用 root 或安装 sudo"
    fi
}

run_systemctl() {
    cmd="$1"

    if systemctl "$cmd" aura >/dev/null 2>&1; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo systemctl "$cmd" aura >/dev/null 2>&1 || true
    fi
}

reload_systemd() {
    if systemctl daemon-reload >/dev/null 2>&1; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo systemctl daemon-reload >/dev/null 2>&1 || true
    fi
}

if [ "${REMOVE_CONFIG:-0}" = "1" ] || [ "${PURGE:-0}" = "1" ]; then
    remove_config=1
else
    remove_config=0
fi

info "开始卸载 Aura..."

# 停止并禁用 systemd 服务
if command -v systemctl >/dev/null 2>&1; then
    info "停止并禁用 aura 服务..."
    run_systemctl stop
    run_systemctl disable
fi

# 删除 systemd 服务文件
if [ -f "$SERVICE_FILE" ]; then
    info "删除服务文件: $SERVICE_FILE"
    remove_with_sudo_if_needed "$SERVICE_FILE"
    if command -v systemctl >/dev/null 2>&1; then
        reload_systemd
    fi
else
    info "未发现服务文件: $SERVICE_FILE"
fi

# 删除二进制
if [ -f "$BINARY_PATH" ]; then
    info "删除二进制: $BINARY_PATH"
    remove_with_sudo_if_needed "$BINARY_PATH"
else
    info "未发现二进制: $BINARY_PATH"
fi

# 配置保留策略
if [ "$remove_config" -eq 1 ]; then
    if [ -d "$CONFIG_DIR" ]; then
        info "删除配置目录: $CONFIG_DIR"
        remove_with_sudo_if_needed "$CONFIG_DIR"
    else
        info "未发现配置目录: $CONFIG_DIR"
    fi
else
    warn "已保留配置目录: $CONFIG_DIR"
    warn "如需删除配置，请使用 REMOVE_CONFIG=1 或 PURGE=1"
fi

info "卸载完成"
