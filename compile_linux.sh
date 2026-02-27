#!/bin/bash

# Cupcake C2 - Linux Agent 独立编译脚本
# 仅编译 Linux 版本的 Agent 模板，存放在 server/assets 中。

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 确保脚本在根目录运行
cd "$(dirname "$0")"
PROJECT_ROOT=$(pwd)
CLIENT_DIR="$PROJECT_ROOT/Client"
ASSETS_DIR="$PROJECT_ROOT/server/assets"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}    Cupcake C2 - Linux Template Compiler ${NC}"
echo -e "${BLUE}=========================================${NC}"

# 1. 检查 Rust 环境
if ! command -v cargo &> /dev/null; then
    echo -e "${RED}[!] 未检测到 Cargo，请先安装 Rust 环境。${NC}"
    exit 1
fi

# 2. 准备输出目录
mkdir -p "$ASSETS_DIR"

# 3. 编译函数
build_linux_template() {
    local arch=$1
    local proto=$2
    local output_name=$3
    local target=""

    echo -e "${YELLOW}[*] 正在构建 Linux 模板: $output_name (Arch: $arch, Feature: $proto)...${NC}"

    if [ "$arch" == "x64" ]; then
        target="x86_64-unknown-linux-musl"
    elif [ "$arch" == "arm64" ]; then
        target="aarch64-unknown-linux-musl"
        # 针对 ARM64 的交叉编译检查 (如果需要)
        # 这里默认假设用户已经配置好 musl 链，或者环境支持
    fi

    # 尝试安装 target
    rustup target add "$target" >/dev/null 2>&1 || true

    cd "$CLIENT_DIR"
    
    # 🛡️ STEALTH: 移除本地路径前缀
    export RUSTFLAGS="--remap-path-prefix $CLIENT_DIR=/cupcake"

    # 执行编译 (使用 --no-default-features 确保功能解耦)
    if cargo build --release --target "$target" --no-default-features --features "$proto"; then
        local src_path="$CLIENT_DIR/target/$target/release/sys-info-collector"
        if [ -f "$src_path" ]; then
            cp "$src_path" "$ASSETS_DIR/$output_name"
            chmod +x "$ASSETS_DIR/$output_name"
            echo -e "${GREEN}[+] 成功生成: $output_name${NC}"
        else
            echo -e "${RED}[!] 错误: 产物文件丢失${NC}"
            exit 1
        fi
    else
        echo -e "${RED}[!] 编译失败: $output_name${NC}"
        exit 1
    fi
    cd ..
}

# 4. 执行批量编译任务
echo -e "${YELLOW}[*] 开始全量 Linux 模板编译进程...${NC}"

# --- x64 架构 ---
build_linux_template "x64" "ws"       "client_template_linux"
build_linux_template "x64" "tcp"      "client_template_linux_tcp"
build_linux_template "x64" "dns"      "client_template_linux_dns"
build_linux_template "x64" "tcp_bind" "client_template_linux_bind"

# --- ARM64 架构 ---
build_linux_template "arm64" "ws"       "client_template_linux_arm64"
build_linux_template "arm64" "tcp_bind" "client_template_linux_bind_arm64"

echo -e "${BLUE}-----------------------------------------${NC}"
echo -e "${GREEN}[DONE] 所有 Linux 模板已就绪。${NC}"
echo -e "${GREEN}[+] 产物目录: $ASSETS_DIR${NC}"
echo -e "${BLUE}-----------------------------------------${NC}"
