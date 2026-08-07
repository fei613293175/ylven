param(
  [Parameter(Mandatory)][ValidatePattern('^P\d{2}$')][string]$Phase,
  [string]$DesktopRoot
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$Source = Join-Path $Root "dist\releases\$Phase"
if (-not (Test-Path $Source)) { throw "Missing authoritative release directory: $Source" }
if (-not $DesktopRoot) {
  $desktop = [Environment]::GetFolderPath([Environment+SpecialFolder]::Desktop)
  if (-not $desktop) { throw 'Windows Desktop path could not be resolved. Use -DesktopRoot explicitly or -SkipDesktopCopy on release.' }
  $DesktopRoot = Join-Path $desktop 'YLVEN_交付'
}
$Target = Join-Path $DesktopRoot $Phase
$Temp = "$Target.tmp-$([Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Force -Path $Temp | Out-Null
try {
  Copy-Item -Path (Join-Path $Source '*') -Destination $Temp -Recurse -Force
  Push-Location $Temp
  try {
    $checksumPath = Join-Path $Temp '校验文件_SHA256.txt'
    if (-not (Test-Path $checksumPath)) { throw "Desktop copy is missing 校验文件_SHA256.txt" }
    foreach ($line in Get-Content $checksumPath) {
      if ($line -match '^([0-9a-fA-F]{64})\s+(.+)$') {
        $name = $Matches[2]
        $file = Join-Path $Temp $name
        if (-not (Test-Path $file)) { throw "Desktop copy is missing checksummed file $name" }
        $actual = (Get-FileHash -Algorithm SHA256 $file).Hash.ToLowerInvariant()
        if ($actual -ne $Matches[1].ToLowerInvariant()) { throw "Desktop copy SHA256 mismatch for $name" }
      }
    }
  } finally { Pop-Location }
  if (Test-Path $Target) { Remove-Item -Path $Target -Recurse -Force }
  Move-Item -Path $Temp -Destination $Target
} catch {
  if (Test-Path $Temp) { Remove-Item -Path $Temp -Recurse -Force }
  throw
}
Write-Host "Desktop delivery synchronized: $Target"
