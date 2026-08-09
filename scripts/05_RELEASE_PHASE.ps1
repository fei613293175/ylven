param(
    [Parameter(Mandatory=$true)][ValidatePattern('^P\d{2}$')][string]$Phase,
    [string]$SshTarget,
    [string]$Serial
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $Root

& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/38_VALIDATE_STATE_CONTINUITY.py'
if ($LASTEXITCODE -ne 0) { throw 'State continuity validation failed.' }
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/40_PUBLIC_REPOSITORY_SAFETY.py'
if ($LASTEXITCODE -ne 0) { throw 'Public repository safety validation failed.' }
$phaseState = Get-Content 'CURRENT_PHASE.yaml' -Raw
if ($phaseState -notmatch "(?m)^phase_id:\s*$Phase\s*$" -or $phaseState -notmatch '(?m)^status:\s*READY_FOR_RELEASE\s*$') {
    throw "$Phase is not the active READY_FOR_RELEASE phase."
}
$version = (& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py' --phase $Phase --print-version 2>$null | Select-Object -Last 1).Trim()
if (-not $version) { throw "Could not resolve the version for $Phase." }
$upgradeFrom = (& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py' --phase $Phase --print-upgrade-from 2>$null | Select-Object -Last 1).Trim()
if (-not $upgradeFrom -or $upgradeFrom -eq 'NONE') {
    throw "Could not resolve the required previous owner version for $Phase physical-device acceptance."
}

$serverParameters = @{ Phase=$Phase; Version=$version }
if ($SshTarget) { $serverParameters.SshTarget = $SshTarget }
& (Join-Path $PSScriptRoot '49_BUILD_ANDROID_ONLINE_SERVER.ps1') @serverParameters | Tee-Object -Variable serverOutput
$serverLine = $serverOutput | Where-Object { $_ -like 'SERVER_BUILD_DIR=*' } | Select-Object -Last 1
if (-not $serverLine) { throw 'Online-server build completed without an artifact directory marker.' }
$serverBuildDir = $serverLine.Substring('SERVER_BUILD_DIR='.Length)

$deviceParameters = @{ Phase=$Phase; Version=$version; PreviousVersion=$upgradeFrom; ServerBuildDir=$serverBuildDir }
if ($Serial) { $deviceParameters.Serial = $Serial }
& (Join-Path $PSScriptRoot '50_RUN_PHYSICAL_DEVICE_ACCEPTANCE.ps1') @deviceParameters | Tee-Object -Variable deviceOutput
$deviceLine = $deviceOutput | Where-Object { $_ -like 'PHYSICAL_ACCEPTANCE_DIR=*' } | Select-Object -Last 1
if (-not $deviceLine) { throw 'Physical-device acceptance completed without an evidence directory marker.' }
$acceptanceDir = $deviceLine.Substring('PHYSICAL_ACCEPTANCE_DIR='.Length)

& (Join-Path $PSScriptRoot '35_DELIVER_ANDROID_RELEASE.ps1') -Phase $Phase -Version $version -AcceptanceDir $acceptanceDir
if ($LASTEXITCODE -ne 0) { throw 'Owner delivery failed.' }
Write-Host "$Phase release evidence is ready for project-owner APPROVED/REJECTED review. The next phase has not started."
