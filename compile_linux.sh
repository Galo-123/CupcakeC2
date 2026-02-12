#!/bin/bash

# Cupcake C2 - Linux Agent 独立编译脚本
# 仅编译 Linux 版本的 Agent，不进行交叉编译 Windows 版本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 确保脚本在根目录运行
cd "$(dirname "$0")"
PROJECT_ROOT=$(pwd)
CLIENT_DIR="$PROJECT_ROOT/Client"
ASSETS_DIR="$PROJECT_ROOT/server/assets"

echo -e "${YELLOW}[*] Cupcake C2 - Linux Agent Compiler${NC}"

# 1. 检查 Rust 环境
if ! command -v cargo &> /dev/null; then
    echo -e "${RED}[!] 未检测到 Cargo，请先安装 Rust 环境。${NC}"
    echo -e "    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
    exit 1
fi

# 2. 准备输出目录
mkdir -p "$ASSETS_DIR"

# 3. 编译函数
build_linux_variant() {
    local arch=$1
    local proto=$2
    local output_name=$3
    local target=""

    echo -e "${YELLOW}[*] details: $arch - $proto ...${NC}"

    if [ "$arch" == "x64" ]; then
        target="x86_64-unknown-linux-gnu"
    elif [ "$arch" == "arm64" ]; then
        target="aarch64-unknown-linux-gnu"
        # 检查交叉编译工具链
        if ! command -v aarch64-linux-gnu-gcc &> /dev/null; then
            echo -e "${RED}[!] 警告: 未找到 aarch64-linux-gnu-gcc，跳过 Linux ARM64 编译。${NC}"
            echo -e "    Debian/Ubuntu: sudo apt install gcc-aarch64-linux-gnu"
            return
        fi
        export CC_aarch64_unknown_linux_gnu=aarch64-linux-gnu-gcc
        export AR_aarch64_unknown_linux_gnu=aarch64-linux-gnu-ar
        export CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc
    fi

    # 确保 target 安装
    rustup target add "$target" >/dev/null 2>&1 || true

    cd "$CLIENT_DIR"
    # 清理以确保纯净构建
    cargo clean -p sys-info-collector >/dev/null 2>&1

    echo -e "    Target: $target, Feature: $proto"
    
    # 编译命令
    if cargo build --release --target "$target" --no-default-features --features "$proto"; then
        local src_path="$CLIENT_DIR/target/$target/release/sys-info-collector"
        if [ -f "$src_path" ]; then
            cp "$src_path" "$ASSETS_DIR/$output_name"
            chmod +x "$ASSETS_DIR/$output_name"
            echo -e "${GREEN}[+] 成功: $output_name${NC}"
        else
            echo -e "${RED}[!] 错误: 产物未生成${NC}"
        fi
    else
        echo -e "${RED}[!] 编译失败: $arch - $proto${NC}"
    fi
}

# 4. 执行全量编译
echo -e "${YELLOW}[*] 开始构建全系 Linux Agent 模板...${NC}"

# --- x64 ---
build_linux_variant "x64" "ws"  "client_template_linux_x64_ws"
build_linux_variant "x64" "tcp" "client_template_linux_x64_tcp"
build_linux_variant "x64" "dns" "client_template_linux_x64_dns"

# --- ARM64 (需要交叉编译工具) ---
build_linux_variant "arm64" "ws"  "client_template_linux_arm64_ws"
build_linux_variant "arm64" "tcp" "client_template_linux_arm64_tcp"

echo -e "${GREEN}[DONE] 所有 Linux 模板构建任务完成。${NC}"
echo -e "${BLUE}-----------------------------------------${NC}"

# 退出脚本，保留后续逻辑作为注释或移除
exit 0
