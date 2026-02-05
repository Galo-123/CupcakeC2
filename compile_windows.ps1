# Cupcake C2 - Windows Template Compiler (Fixed Encoding)
# This script compiles Windows Agent templates for various protocols.

$ErrorActionPreference = "Stop"

# 1. Path Setup
$BaseDir = Get-Location
$ClientDir = Join-Path $BaseDir "Client"
$AssetsDir = Join-Path $BaseDir "server\assets"

Write-Host "=========================================" -ForegroundColor Blue
Write-Host "    Cupcake C2 - Template Compiler       " -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

if (-not (Test-Path $AssetsDir)) {
    New-Item -ItemType Directory -Path $AssetsDir | Out-Null
}

# 2. Check Environment
Write-Host "[*] Checking build environment..." -ForegroundColor Yellow
if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) {
    Write-Host "[!] Error: Rust environment (cargo) not found. Please install Rust first." -ForegroundColor Red
    exit 1
}

# Check targets
$installedTargets = rustup target list --installed
$x64Target = "x86_64-pc-windows-msvc"
$x86Target = "i686-pc-windows-msvc"

if ($installedTargets -notmatch $x64Target) {
    Write-Host "[*] Adding x64 build component..." -ForegroundColor Yellow
    rustup target add $x64Target
}
if ($installedTargets -notmatch $x86Target) {
    Write-Host "[*] Adding x86 build component..." -ForegroundColor Yellow
    rustup target add $x86Target
}

# 3. Build Function
function Build-Template {
    param (
        [string]$Arch,
        [string]$Feature,
        [string]$OutputName
    )

    $Target = if ($Arch -eq "x64") { $x64Target } else { $x86Target }
    
    Write-Host "[*] Compiling: $OutputName (Arch: $Arch, Feature: $Feature)..." -ForegroundColor Yellow
    
    # Use -NoDefaultFeatures if needed, based on generate_templates.sh logic
    Push-Location $ClientDir
    try {
        # Clean build to avoid artifact contamination
        cargo clean --target $Target
        cargo build --release --target $Target --no-default-features --features $Feature
        
        $BinaryName = "sys-info-collector.exe"
        $SrcPath = Join-Path $ClientDir "target\$Target\release\$BinaryName"
        $DestPath = Join-Path $AssetsDir $OutputName

        if (Test-Path $SrcPath) {
            if (Test-Path $DestPath) { Remove-Item $DestPath -Force }
            Copy-Item -Path $SrcPath -Destination $DestPath -Force
            Write-Host "[+] Successfully generated: $OutputName" -ForegroundColor Green
        } else {
            Write-Host "[!] Error: Binary not found at $SrcPath" -ForegroundColor Red
            exit 1
        }
    }
    finally {
        Pop-Location
    }
}

# 4. Starting Build Process
Write-Host "[*] Starting batch compilation for Windows templates..." -ForegroundColor Cyan

# WebSocket Templates
Build-Template -Arch "x64" -Feature "ws" -OutputName "client_template_windows.exe"
Build-Template -Arch "x86" -Feature "ws" -OutputName "client_template_windows_x86.exe"

# TCP Template
Build-Template -Arch "x64" -Feature "tcp" -OutputName "client_template_windows_tcp.exe"

# DNS Template
Build-Template -Arch "x64" -Feature "dns" -OutputName "client_template_windows_dns.exe"

Write-Host "-----------------------------------------" -ForegroundColor Blue
Write-Host "[DONE] All Windows templates are ready." -ForegroundColor Green
Write-Host "[+] Asset directory: $AssetsDir" -ForegroundColor Green
Write-Host "-----------------------------------------" -ForegroundColor Blue
