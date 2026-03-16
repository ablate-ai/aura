#!/bin/sh
# node_exporter 卸载脚本
# 用法: sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/uninstall_node_exporter.sh)"
# 国内加速: sh -c "$(curl -sfL https://ghfast.top/https://raw.githubusercontent.com/ablate-ai/aura/main/uninstall_node_exporter.sh)"

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

BINARY_PATH="/usr/local/bin/node_exporter"
SERVICE_FILE="/etc/systemd/system/node_exporter.service"
SERVICE_NAME="node_exporter"
SERVICE_USER="node_exporter"

run_with_sudo_if_needed() {
    if "$@" 2>/dev/null; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo "$@"
    else
        error "没有权限执行: $*，请使用 root 或安装 sudo"
    fi
}

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

    if systemctl "$cmd" "$SERVICE_NAME" >/dev/null 2>&1; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo systemctl "$cmd" "$SERVICE_NAME" >/dev/null 2>&1 || true
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

info "开始卸载 node_exporter..."

if command -v systemctl >/dev/null 2>&1; then
    info "停止并禁用 node_exporter 服务..."
    run_systemctl stop
    run_systemctl disable
fi

if [ -f "$SERVICE_FILE" ]; then
    info "删除服务文件: $SERVICE_FILE"
    remove_with_sudo_if_needed "$SERVICE_FILE"
    if command -v systemctl >/dev/null 2>&1; then
        reload_systemd
    fi
else
    info "未发现服务文件: $SERVICE_FILE"
fi

if [ -f "$BINARY_PATH" ]; then
    info "删除二进制: $BINARY_PATH"
    remove_with_sudo_if_needed "$BINARY_PATH"
else
    info "未发现二进制: $BINARY_PATH"
fi

if id "$SERVICE_USER" >/dev/null 2>&1; then
    info "删除系统用户: $SERVICE_USER"
    run_with_sudo_if_needed userdel "$SERVICE_USER" >/dev/null 2>&1 || \
        warn "删除用户失败，请手动检查 $SERVICE_USER 是否仍被其他进程占用"
else
    info "未发现系统用户: $SERVICE_USER"
fi

info "卸载完成"
