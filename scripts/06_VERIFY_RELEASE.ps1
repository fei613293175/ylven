param(
  [Parameter(Mandatory)][ValidatePattern('^P\d{2}$')][string]$Phase,
  [switch]$AllowSelfTestFixture
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Push-Location $Root
try {
  $arguments = @('--phase', $Phase)
  if ($AllowSelfTestFixture) { $arguments += '--allow-self-test-fixture' }
  & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/16_VERIFY_RELEASE.py' @arguments
  if ($LASTEXITCODE -ne 0) { throw "Release contract verification failed for $Phase." }
} finally {
  Pop-Location
}
Write-Host "Release contract verified for $Phase."
