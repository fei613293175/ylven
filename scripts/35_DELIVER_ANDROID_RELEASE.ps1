param(
    [Parameter(Mandatory=$true)][ValidatePattern('^P\d{2}$')][string]$Phase,
    [Parameter(Mandatory=$true)][string]$Version,
    [Parameter(Mandatory=$true)][string]$AcceptanceDir
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$AcceptanceDir = (Resolve-Path -LiteralPath $AcceptanceDir).Path

$ExpectedOrigin = 'https://github.com/fei613293175/ylven.git'
$ActualOrigin = (git -C $Root remote get-url origin).Trim()
if ($ActualOrigin -ne $ExpectedOrigin) { throw "origin mismatch: $ActualOrigin" }

$Commit = (git -C $Root rev-parse HEAD).Trim()
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/33_VERIFY_ANDROID_ACCEPTANCE_OUTPUT.py' --artifact-dir $AcceptanceDir --phase $Phase --version $Version --commit $Commit
if ($LASTEXITCODE -ne 0) { throw 'Physical-device acceptance output validation failed.' }
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/34_PREPARE_DESKTOP_DELIVERY.py' --artifact-dir $AcceptanceDir --phase $Phase --version $Version
if ($LASTEXITCODE -ne 0) { throw 'Desktop delivery validation failed.' }
Write-Host "YLVEN $Version exact server APK and physical-device evidence were delivered to the desktop." -ForegroundColor Green
