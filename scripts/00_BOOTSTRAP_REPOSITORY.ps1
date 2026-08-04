param(
  [string]$ProjectRoot=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path,
  [switch]$SkipSpecKitInstall,
  [switch]$SkipInitialPush
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$ExpectedOrigin='https://github.com/fei613293175/ylven.git'
$VisibilityDecision='OWNER_CONFIRMED_PUBLIC'
function Refresh-Path { $machine=[Environment]::GetEnvironmentVariable('Path','Machine'); $user=[Environment]::GetEnvironmentVariable('Path','User'); $env:Path="$machine;$user;$env:USERPROFILE\.local\bin;$env:USERPROFILE\.cargo\bin;$env:Path" }
function Ensure-WingetCommand([string]$Name,[string]$Id) {
 if (Get-Command $Name -ErrorAction SilentlyContinue) { return }
 if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { throw "$Name is required and winget is unavailable." }
 & winget install --id $Id -e --accept-package-agreements --accept-source-agreements; if($LASTEXITCODE-ne 0){throw "Failed to install $Name"}; Refresh-Path
}
Refresh-Path
Ensure-WingetCommand git Git.Git
if (-not (Get-Command uv -ErrorAction SilentlyContinue)) { Ensure-WingetCommand uv astral-sh.uv }
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) { Write-Warning 'GitHub CLI is not installed yet. Bootstrap can continue; release dispatch/download will install or require it later.' }
Set-Location $ProjectRoot
if (-not (Test-Path '.git')) {
 Write-Host 'No .git directory found. Initializing the owner-confirmed public repository locally.'
 & git init -b main; if($LASTEXITCODE-ne 0){throw 'git init failed'}
}
$previousErrorActionPreference=$ErrorActionPreference
$ErrorActionPreference='Continue'
$origin=& git remote get-url origin 2>$null
$originExitCode=$LASTEXITCODE
$ErrorActionPreference=$previousErrorActionPreference
if($originExitCode -ne 0){$origin=$null}
if (-not $origin) { & git remote add origin $ExpectedOrigin }
elseif ($origin.Trim() -ne $ExpectedOrigin) { throw "origin mismatch: $origin" }
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/40_PUBLIC_REPOSITORY_SAFETY.py'
if (-not $SkipSpecKitInstall) { & (Join-Path $PSScriptRoot '01_INSTALL_SPECKIT_WINDOWS.ps1') -ProjectRoot $ProjectRoot }
New-Item -ItemType Directory -Force -Path '.specify\templates\overrides','.agents\skills' | Out-Null
Copy-Item 'speckit\overrides\*' '.specify\templates\overrides\' -Recurse -Force
Copy-Item 'speckit\ylven-skills\*' '.agents\skills\' -Recurse -Force
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/12_REGENERATE_CONTRACTS.py'
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/07_VALIDATE_CONTRACTS.py'
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/38_VALIDATE_STATE_CONTINUITY.py'
& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/37_PROJECT_STATE.py' init
if (-not (& git config user.email)) { & git config user.email 'codex@local.invalid' }
if (-not (& git config user.name)) { & git config user.name 'Codex Project Agent' }
$previousErrorActionPreference=$ErrorActionPreference
$ErrorActionPreference='Continue'
$null=& git rev-parse --verify HEAD 2>$null
$hasHead=($LASTEXITCODE -eq 0)
$ErrorActionPreference=$previousErrorActionPreference
if (-not $hasHead) {
 & git add .; & git commit -m 'chore: initialize public YLVEN development baseline [P00-W01]'; if($LASTEXITCODE-ne 0){throw 'Initial commit failed'}
}
if (-not $SkipInitialPush) {
 & git push -u origin main
 if ($LASTEXITCODE-ne 0) {
  New-Item -ItemType Directory -Force '.ylven-local' | Out-Null
  "Initial push pending. Authenticate GitHub CLI/Git and run: git push -u origin main" | Set-Content '.ylven-local\PUSH_PENDING.md' -Encoding UTF8
  Write-Warning 'Initial push could not complete. This blocks remote CI only; local repository work may continue.'
 }
}
$branch='phase/p00-engineering-foundation'; $current=(& git branch --show-current).Trim()
if ($current -ne $branch) { & git switch -C $branch }
if (-not $SkipInitialPush) { & git push -u origin $branch 2>$null; if($LASTEXITCODE-ne 0){Write-Warning 'Phase branch push is pending.'} }
Write-Host "Bootstrap complete under $VisibilityDecision. Local machine remains a thin control client; heavy builds run in GitHub Actions or on the SSH-connected server."
Write-Host 'Run .\ylven.ps1 resume in every new Codex conversation.'
