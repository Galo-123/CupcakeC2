# Cupcake C2 - 编译缓存清理脚本 (保留模板)

$ProjectRoot = Get-Location
$ClientDir = Join-Path $ProjectRoot "Client"
$ServerDir = Join-Path $ProjectRoot "server"
$AssetsDir = Join-Path $ServerDir "assets"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "    Cupcake C2 - Build Cache Cleaner" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# 1. 清理 Rust 编译缓存
if (Test-Path $ClientDir) {
    Write-Host "[*] 正在清理 Rust (Client) 编译缓存..." -ForegroundColor Yellow
    Set-Location $ClientDir
    # 使用 cargo clean 可以安全清除 target 目录
    # 注意：这不会影响已经 cp 到 server/assets 的 exe
    cargo clean
    Set-Location $ProjectRoot
    Write-Host "[+] Rust 缓存已清除。" -ForegroundColor Green
}

# 2. 清理 Go 编译缓存
if (Test-Path $ServerDir) {
    Write-Host "[*] 正在清理 Go (Server) 构建与模块缓存..." -ForegroundColor Yellow
    Set-Location $ServerDir
    # 清理构建缓存
    go clean -cache
    # 可选：清理 mod 缓存 (如果想彻底刷新依赖)
    # go clean -modcache 
    Set-Location $ProjectRoot
    Write-Host "[+] Go 缓存已清除。" -ForegroundColor Green
}

# 3. 验证并保护目标目录
Write-Host "[*] 正在保护并验证模板资产..." -ForegroundColor Cyan
if (Test-Path $AssetsDir) {
    $TemplateCount = (Get-ChildItem $AssetsDir -File).Count
    Write-Host "[!] 成功保留 $TemplateCount 个已生成的模板文件在 $AssetsDir" -ForegroundColor White
}

Write-Host "-----------------------------------------" -ForegroundColor Cyan
Write-Host "[DONE] 所有中间编译缓存已清理完毕，磁盘空间已释放。" -ForegroundColor Green
