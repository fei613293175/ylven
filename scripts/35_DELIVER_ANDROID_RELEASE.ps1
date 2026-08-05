param(
    [Parameter(Mandatory=$true)][string]$Phase,
    [Parameter(Mandatory=$true)][string]$Version,
    [Parameter(Mandatory=$true)][long]$RunId
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Repo = 'fei613293175/ylven'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$Temp = Join-Path $Root ".ylven-local\ci-artifacts\$RunId"
if (Test-Path $Temp) { Remove-Item -Recurse -Force $Temp }
New-Item -ItemType Directory -Path $Temp -Force | Out-Null

$ExpectedOrigin = 'https://github.com/fei613293175/ylven.git'
$ActualOrigin = (git -C $Root remote get-url origin).Trim()
if ($ActualOrigin -ne $ExpectedOrigin) { throw "origin mismatch: $ActualOrigin" }

$ArtifactPattern = "ylven-$Phase-$Version-*-android-acceptance"
& gh run download $RunId --repo $Repo --pattern $ArtifactPattern --dir $Temp
if ($LASTEXITCODE -ne 0) { throw 'Failed to download the exact GitHub Actions artifact.' }
$ArtifactDirs = @(Get-ChildItem -Path $Temp -Directory)
if ($ArtifactDirs.Count -ne 1) { throw "Expected one exact artifact directory, got $($ArtifactDirs.Count)." }

& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/34_PREPARE_DESKTOP_DELIVERY.py' --artifact-dir $ArtifactDirs[0].FullName --phase $Phase --version $Version
if ($LASTEXITCODE -ne 0) { throw 'Desktop delivery validation failed.' }
Write-Host "YLVEN $Version has been delivered from exact CI Artifact to the desktop." -ForegroundColor Green
