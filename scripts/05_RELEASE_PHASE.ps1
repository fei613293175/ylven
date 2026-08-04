param([Parameter(Mandatory)][ValidatePattern('^P\d{2}$')][string]$Phase,[switch]$NoWait)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$Root=(Resolve-Path(Join-Path $PSScriptRoot '..')).Path; Set-Location $Root
if(-not(Get-Command gh -ErrorAction SilentlyContinue)){
 if(Get-Command winget -ErrorAction SilentlyContinue){& winget install --id GitHub.cli -e --accept-package-agreements --accept-source-agreements}
}
if(-not(Get-Command gh -ErrorAction SilentlyContinue)){throw 'GitHub CLI is required to dispatch and retrieve the CI release. Java/Gradle/ADB/Docker are not required locally.'}
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/38_VALIDATE_STATE_CONTINUITY.py'
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/40_PUBLIC_REPOSITORY_SAFETY.py'
$phaseState=Get-Content 'CURRENT_PHASE.yaml' -Raw
if($phaseState -notmatch 'status:\s*READY_FOR_RELEASE'){throw "$Phase is not READY_FOR_RELEASE; close all Work Packets first."}
$matrix=Get-Content 'contracts/release-version-matrix.yaml' -Raw
$version=(& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py' --phase $Phase --print-version 2>$null | Select-Object -Last 1)
if(-not $version){$entry=Select-String -Path 'contracts/release-version-matrix.yaml' -Pattern "phase: $Phase" -Context 0,5; $version=($entry.Context.PostContext|Select-String 'version_name:'|Select-Object -First 1).Line.Split(':',2)[1].Trim()}
$sha=(& git rev-parse HEAD).Trim(); $branch=(& git branch --show-current).Trim(); & git push -u origin $branch; if($LASTEXITCODE-ne 0){throw 'Push current release commit before CI dispatch.'}
& gh workflow run android-phase-acceptance.yml --ref $branch -f phase=$Phase -f version=$version -f commit_sha=$sha
if($LASTEXITCODE-ne 0){throw 'Could not dispatch GitHub Actions release acceptance.'}
Start-Sleep -Seconds 3
$run=(& gh run list --workflow android-phase-acceptance.yml --branch $branch --limit 1 --json databaseId,status,headSha --jq '.[0] | "\(.databaseId) \(.status) \(.headSha)"').Trim(); Write-Host "Dispatched CI: $run"
if(-not $NoWait){$id=($run -split ' ')[0]; & gh run watch $id --exit-status; if($LASTEXITCODE-ne 0){throw 'GitHub Actions acceptance failed.'}; Write-Host "CI passed. Deliver exact Artifact with scripts/35_DELIVER_ANDROID_RELEASE.ps1 -Phase $Phase -Version $version -RunId $id"}
