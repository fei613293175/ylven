param([Parameter(Mandatory)][ValidatePattern('^P\d{2}$')][string]$Phase)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$Root=(Resolve-Path(Join-Path $PSScriptRoot '..')).Path
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/07_VALIDATE_CONTRACTS.py' --phase $Phase
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/37_PROJECT_STATE.py' resume
$state=Get-Content (Join-Path $Root 'CURRENT_WORK_PACKET.yaml') -Raw
Write-Host "Phase validation complete. Start only the current Work Packet shown by .\\ylven.ps1 resume; do not select by chat."
