# CupcakeC2 v3.0.1 - Wasm Skill Compiler
$ErrorActionPreference = "Stop"

$SkillName = "base_template"
if ($args.Count -gt 0) { $SkillName = $args[0] }

$BaseDir = Get-Location
$OutputDir = Join-Path $BaseDir "server\assets\plugins"

Write-Host "--- CupcakeC2 v3.0.1 Skill Factory ---"

# 1. Target check
$installedTargets = rustup target list --installed
if ($installedTargets -notmatch "wasm32-unknown-unknown") {
    rustup target add wasm32-unknown-unknown
}

# 2. Check source
$SkillSource = Join-Path $BaseDir "Skills\$SkillName"
if (-not (Test-Path $SkillSource)) {
    Write-Host "Error: Source not found at $SkillSource"
    exit 1
}

# 3. Build to Wasm
Write-Host "Compiling skill: $SkillName ..."
Push-Location $SkillSource

# Fixed character escaping for powershell
rustc --target wasm32-unknown-unknown --crate-type cdylib -C panic=abort -C opt-level=z lib.rs -o "$SkillName.wasm"

if (Test-Path "$SkillName.wasm") {
    if (-not (Test-Path $OutputDir)) { New-Item -ItemType Directory -Path $OutputDir }
    Move-Item -Path "$SkillName.wasm" -Destination (Join-Path $OutputDir "$SkillName.wasm") -Force
    Write-Host "Success: server/assets/plugins/$SkillName.wasm"
} else {
    Write-Error "Build failed."
}
Pop-Location
