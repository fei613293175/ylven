param([string]$SshTarget)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$Root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
if (-not $SshTarget) { $SshTarget=$env:YLVEN_SSH_TARGET }
$targetFile=Join-Path $Root '.ylven-local\ssh-target.txt'
if (-not $SshTarget -and (Test-Path $targetFile)) { $SshTarget=(Get-Content $targetFile -Raw).Trim() }
if (-not $SshTarget) { throw 'SSH target is not stored in the public repository. Set YLVEN_SSH_TARGET or .ylven-local/ssh-target.txt, then rerun.' }
if (-not (Get-Command ssh -ErrorAction SilentlyContinue)) { throw 'OpenSSH client is required for the remote inventory command.' }
$outDir=Join-Path $Root '.ylven-local'; New-Item -ItemType Directory -Force -Path $outDir | Out-Null
$stamp=Get-Date -Format 'yyyyMMdd-HHmmss'; $out=Join-Path $outDir "server-inventory-$stamp.txt"
$script=@'
set -u
section(){ printf '\n===== %s =====\n' "$1"; }
section OS; (cat /etc/os-release 2>/dev/null || true); uname -a || true
section CPU_MEMORY; (nproc || true); (free -h || true)
section DISK; df -h || true
section DOCKER; (docker version 2>/dev/null || true); (docker compose version 2>/dev/null || true)
section CONTAINERS; (docker ps --format '{{.Names}}\t{{.Image}}\t{{.Ports}}' 2>/dev/null || true)
section NETWORKS; (docker network ls 2>/dev/null || true)
section VOLUMES; (docker volume ls 2>/dev/null || true)
section PORTS; (ss -lntup 2>/dev/null || netstat -lntup 2>/dev/null || true)
section TOOLCHAINS
for c in git java go node pnpm python3 adb; do printf '%s: ' "$c"; command -v "$c" 2>/dev/null || true; "$c" --version 2>/dev/null | head -1 || true; done
section ANDROID_SDK; printf 'ANDROID_HOME=%s\nANDROID_SDK_ROOT=%s\n' "${ANDROID_HOME:-}" "${ANDROID_SDK_ROOT:-}"; (ls -1 "${ANDROID_SDK_ROOT:-/nonexistent}/platforms" 2>/dev/null || true)
section REVERSE_PROXY; (nginx -v 2>&1 || true); (caddy version 2>/dev/null || true); (traefik version 2>/dev/null || true)
'@
$result=& ssh $SshTarget $script 2>&1
$result | Set-Content -Path $out -Encoding UTF8
if ($LASTEXITCODE -ne 0) { throw "Remote read-only inventory failed; partial output: $out" }
Write-Host "Read-only server inventory saved locally: $out"
Write-Host 'This file is under .ylven-local and must never be committed.'
