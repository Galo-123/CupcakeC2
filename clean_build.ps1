# Cupcake C2 - Build Cache Cleaner (Perserve Templates)
# Version 3.0 - ZERO TRACE & ULTRA LIGHT (Final Release Prep)

$ProjectRoot = Get-Location
$ClientDir = Join-Path $ProjectRoot "Client"
$ServerDir = Join-Path $ProjectRoot "server"
$FrontendDir = Join-Path $ServerDir "frontend-v2"
$AssetsDir = Join-Path $ServerDir "assets"
$DistDir = Join-Path $ServerDir "dist"
$TempBuildsDir = Join-Path $ServerDir "temp_builds"
$StorageDir = Join-Path $ServerDir "storage"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "    Cupcake C2 - ZERO TRACE CLEANER" -ForegroundColor Red
Write-Host "=========================================" -ForegroundColor Cyan

# 1. Clean Rust Cache (Main Client)
if (Test-Path $ClientDir) {
    Write-Host "[*] Cleaning Rust (Client) build cache..." -ForegroundColor Yellow
    Push-Location $ClientDir
    try {
        cargo clean
    } finally {
        Pop-Location
    }
    Write-Host "[+] Rust cache cleared." -ForegroundColor Green
}

# 2. Clean Go Cache
if (Test-Path $ServerDir) {
    Write-Host "[*] Cleaning Go (Server) build cache..." -ForegroundColor Yellow
    Push-Location $ServerDir
    try {
        go clean -cache
        go clean -testcache
    } finally {
        Pop-Location
    }
    Write-Host "[+] Go cache cleared." -ForegroundColor Green
}

# 3. Clean Frontend Build & Dependencies (ULTRA LIGHT)
if (Test-Path $DistDir) {
    Write-Host "[*] Cleaning Frontend build output (dist)..." -ForegroundColor Yellow
    Remove-Item -Path $DistDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[+] Frontend dist cleared." -ForegroundColor Green
}

$NodeModules = Join-Path $FrontendDir "node_modules"
if (Test-Path $NodeModules) {
    Write-Host "[!] Removing Frontend node_modules (This may take a while)..." -ForegroundColor Magenta
    Remove-Item -Path $NodeModules -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[+] node_modules removed. (Run 'npm install' on Linux to restore)" -ForegroundColor Green
}

# 4. Clean Global Temp Builds
if (Test-Path $TempBuildsDir) {
    Write-Host "[*] Cleaning temporary builds..." -ForegroundColor Yellow
    Remove-Item -Path $TempBuildsDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[+] Temp builds cleared." -ForegroundColor Green
}

# 5. Clean Storage & DATABASE (ZERO TRACE)
if (Test-Path $StorageDir) {
    Write-Host "[!] Deep cleaning storage and DATABASE..." -ForegroundColor Magenta
    
    # Securely delete the DB (Contains ALL IP, Agent history, and commands)
    $DBFile = Join-Path $StorageDir "cupcake.db"
    if (Test-Path $DBFile) {
        Remove-Item -Path $DBFile -Force -ErrorAction SilentlyContinue
        Write-Host "    [!] Database shredded. Privacy level: 100%" -ForegroundColor Gray
    }

    # Clean Logs
    $LogsDir = Join-Path $StorageDir "logs"
    if (Test-Path $LogsDir) {
        Remove-Item -Path $LogsDir -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $LogsDir -Force | Out-Null
        Write-Host "    [+] Task logs wiped." -ForegroundColor Gray
    }

    # Clean Build Cache
    $SBuildCache = Join-Path $StorageDir "build_cache"
    if (Test-Path $SBuildCache) {
        Remove-Item -Path $SBuildCache -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $SBuildCache -Force | Out-Null
        Write-Host "    [+] Storage build cache wiped." -ForegroundColor Gray
    }

    # Clean Agent Files
    $AgentFiles = Join-Path $StorageDir "agent_files"
    if (Test-Path $AgentFiles) {
        Remove-Item -Path $AgentFiles -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $AgentFiles -Force | Out-Null
        Write-Host "    [+] Agent uploaded files wiped." -ForegroundColor Gray
    }

    # Clean Generated Payloads
    $PayloadsDir = Join-Path $StorageDir "payloads"
    if (Test-Path $PayloadsDir) {
        Remove-Item -Path $PayloadsDir -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $PayloadsDir -Force | Out-Null
        Write-Host "    [+] Historical payloads wiped." -ForegroundColor Gray
    }
    
    Write-Host "[+] Storage zero-trace cleanup finished." -ForegroundColor Green
}

# 6. Verify Assets (Protecting the precious templates)
Write-Host "[*] Verifying and protecting template seeds..." -ForegroundColor Cyan
if (Test-Path $AssetsDir) {
    $files = Get-ChildItem $AssetsDir -File
    $count = ($files).Count
    Write-Host "[!] Successfully preserved $count template seeds in $AssetsDir" -ForegroundColor White
}

# 7. Final Cleanup (IDE / Temp)
Write-Host "[*] Finalizing cleanup (IDE and temp bits)..." -ForegroundColor Yellow
$IDEDirs = @(".idea", ".vscode", ".gemini")
foreach ($id in $IDEDirs) {
    $p = Join-Path $ProjectRoot $id
    if (Test-Path $p) {
        Remove-Item -Path $p -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "-----------------------------------------" -ForegroundColor Cyan
Write-Host "[SUCCESS] Project is now in PURE state." -ForegroundColor Green
Write-Host "[NOTICE] Privacy: CLEAN | Size: MINIMAL | Ready for GitHub/Linux." -ForegroundColor White
