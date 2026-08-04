param([string]$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Pinned = 'v0.15.2'
$Expected = '0.15.2'
$Source = "git+https://github.com/github/spec-kit.git@$Pinned"

function Refresh-Path {
  $machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
  $user = [Environment]::GetEnvironmentVariable('Path', 'User')
  $env:Path = "$machine;$user;$env:USERPROFILE\.local\bin;$env:USERPROFILE\.cargo\bin;$env:Path"
}

Refresh-Path
if (-not (Get-Command uv -ErrorAction SilentlyContinue)) {
  if (Get-Command winget -ErrorAction SilentlyContinue) {
    & winget install --id astral-sh.uv -e --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) { throw "winget failed to install uv (exit $LASTEXITCODE)." }
  } else {
    Invoke-RestMethod https://astral.sh/uv/install.ps1 | Invoke-Expression
  }
  Refresh-Path
}
if (-not (Get-Command uv -ErrorAction SilentlyContinue)) { throw 'uv installation was not found on PATH.' }

& uv tool install specify-cli --force --from $Source
if ($LASTEXITCODE -ne 0) { throw 'Pinned Spec Kit installation failed.' }
Refresh-Path
if (-not (Get-Command specify -ErrorAction SilentlyContinue)) {
  throw 'Spec Kit installed but specify is not visible on PATH. Restart PowerShell/Codex and rerun.'
}

Push-Location $ProjectRoot
try {
  $versionOutput = (& specify version 2>&1 | Out-String)
  if ($LASTEXITCODE -ne 0) { throw 'specify version failed.' }
  Write-Host $versionOutput.Trim()
  if ($versionOutput -notmatch [regex]::Escape($Expected)) {
    throw "Expected Spec Kit $Expected but got a different version. Refusing unpinned workflow behavior."
  }

  $integrationState = Join-Path $ProjectRoot '.specify\integration.json'
  $coreSkill = Join-Path $ProjectRoot '.agents\skills\speckit-specify\SKILL.md'
  if (-not (Test-Path $integrationState)) {
    & specify init --here --force --integration codex --script ps --ignore-agent-tools
    if ($LASTEXITCODE -ne 0) {
      Write-Warning 'Persistent specify init failed; trying the same pinned source through uvx.'
      & uvx --from $Source specify init --here --force --integration codex --script ps --ignore-agent-tools
      if ($LASTEXITCODE -ne 0) { throw 'Spec Kit project initialization failed.' }
    }
  } elseif (-not (Test-Path $coreSkill)) {
    & specify integration install codex --script ps --force
    if ($LASTEXITCODE -ne 0) { throw 'Codex integration installation failed.' }
    & specify integration use codex --force
    if ($LASTEXITCODE -ne 0) { throw 'Could not make Codex the default integration.' }
  }

  & specify integration status
  if ($LASTEXITCODE -ne 0) { throw 'Spec Kit integration status reported an error.' }
} finally {
  Pop-Location
}
Write-Host "Spec Kit $Pinned installation and Codex initialization completed."
