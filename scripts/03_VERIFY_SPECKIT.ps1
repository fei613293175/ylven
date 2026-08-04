param([string]$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$Expected = '0.15.2'
if (-not (Get-Command specify -ErrorAction SilentlyContinue)) { throw 'specify is not installed.' }
Push-Location $ProjectRoot
try {
  $versionOutput = (& specify version 2>&1 | Out-String)
  if ($LASTEXITCODE -ne 0) { throw 'specify version failed.' }
  if ($versionOutput -notmatch [regex]::Escape($Expected)) { throw "Spec Kit version mismatch; expected $Expected." }
  $statusJson = (& specify integration status --json 2>&1 | Out-String)
  if ($LASTEXITCODE -ne 0) { throw "Spec Kit integration status reported an error: $statusJson" }
  try { $status = $statusJson | ConvertFrom-Json } catch { throw "Spec Kit status was not valid JSON: $statusJson" }
  if ($status.status -eq 'error') { throw 'Spec Kit integration state is error.' }
} finally {
  Pop-Location
}
$required = @('speckit-constitution', 'speckit-specify', 'speckit-plan', 'speckit-tasks', 'speckit-implement')
foreach ($skill in $required) {
  $path = Join-Path $ProjectRoot ".agents\skills\$skill\SKILL.md"
  if (-not (Test-Path $path)) { throw "Missing Codex skill: $path" }
}
$overrides = @('constitution-template.md', 'spec-template.md', 'plan-template.md', 'tasks-template.md')
foreach ($file in $overrides) {
  $path = Join-Path $ProjectRoot ".specify\templates\overrides\$file"
  if (-not (Test-Path $path)) { throw "Missing project override: $path" }
}
Write-Host 'Spec Kit 0.15.2, Codex skills and YLVEN local overrides verified.'
