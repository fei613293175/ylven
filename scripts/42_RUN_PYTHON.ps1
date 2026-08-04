param(
  [Parameter(Mandatory=$true)][string]$Script,
  [Parameter(ValueFromRemainingArguments=$true)][string[]]$ScriptArguments
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$Root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
function Refresh-Path {
 $machine=[Environment]::GetEnvironmentVariable('Path','Machine'); $user=[Environment]::GetEnvironmentVariable('Path','User')
 $env:Path="$machine;$user;$env:USERPROFILE\.local\bin;$env:USERPROFILE\.cargo\bin;$env:Path"
}
Refresh-Path
if (-not (Get-Command uv -ErrorAction SilentlyContinue)) {
 if (Get-Command winget -ErrorAction SilentlyContinue) { & winget install --id astral-sh.uv -e --accept-package-agreements --accept-source-agreements }
 else { Invoke-RestMethod https://astral.sh/uv/install.ps1 | Invoke-Expression }
 Refresh-Path
}
if (-not (Get-Command uv -ErrorAction SilentlyContinue)) { throw 'uv is required; no system Python is required.' }
$scriptPath=if([IO.Path]::IsPathRooted($Script)){$Script}else{Join-Path $Root $Script}
if (-not (Test-Path $scriptPath)) { throw "Python script not found: $scriptPath" }
& uv run --python 3.12 --with-requirements (Join-Path $Root 'requirements-tools.txt') python $scriptPath @ScriptArguments
if ($LASTEXITCODE -ne 0) { throw "Repository tool failed: $Script" }
