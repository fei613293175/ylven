param(
    [Parameter(Mandatory=$true)][ValidatePattern('^P\d{2}$')][string]$Phase,
    [string]$Version,
    [string]$SshTarget,
    [string]$PreviousApk
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path

function Copy-FileToOnlineServer {
    param(
        [Parameter(Mandatory=$true)][string]$SourcePath,
        [Parameter(Mandatory=$true)][string]$SshTarget,
        [Parameter(Mandatory=$true)][string]$RemotePath,
        [Parameter(Mandatory=$true)][string]$LocalChunkRoot
    )

    # This server closes long SCP streams; bounded chunks make transfer restartable and hash-verifiable.
    $source = (Resolve-Path -LiteralPath $SourcePath).Path
    $sourceHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $source).Hash.ToLowerInvariant()
    $chunkDirectory = Join-Path $LocalChunkRoot ([System.IO.Path]::GetFileName($source))
    $remoteChunkDirectory = "$RemotePath.parts"
    $chunkSize = 4MB
    New-Item -ItemType Directory -Path $chunkDirectory -Force | Out-Null

    $prepareCommand = "set -eu; mkdir -p '$remoteChunkDirectory'"
    & ssh $SshTarget $prepareCommand
    if ($LASTEXITCODE -ne 0) { throw "Could not prepare chunk upload directory for $RemotePath." }

    $input = [System.IO.File]::OpenRead($source)
    $partCount = 0
    try {
        $buffer = New-Object byte[] $chunkSize
        while (($read = $input.Read($buffer, 0, $buffer.Length)) -gt 0) {
            $partName = ('part-{0:D6}' -f $partCount)
            $localPart = Join-Path $chunkDirectory $partName
            $output = [System.IO.File]::Open($localPart, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
            try {
                $output.Write($buffer, 0, $read)
            } finally {
                $output.Dispose()
            }
            & scp $localPart "${SshTarget}:$remoteChunkDirectory/$partName"
            if ($LASTEXITCODE -ne 0) { throw "Could not upload chunk $partName for $RemotePath." }
            $partCount++
        }
    } finally {
        $input.Dispose()
    }

    if ($partCount -eq 0) { throw "Refusing to upload empty file $source." }
    $assembleCommand = @"
set -eu
part_count=`$(find '$remoteChunkDirectory' -maxdepth 1 -type f -name 'part-*' -printf '%f\n' | sort | wc -l)
test "`$part_count" -eq $partCount
cat '$remoteChunkDirectory'/part-* > '$RemotePath'
printf '%s  %s\n' '$sourceHash' '$RemotePath' | sha256sum -c -
"@
    & ssh $SshTarget $assembleCommand
    if ($LASTEXITCODE -ne 0) { throw "Server-side reconstruction or SHA-256 verification failed for $RemotePath." }
}

Push-Location $Root
try {
    if (-not $SshTarget) { $SshTarget = $env:YLVEN_SSH_TARGET }
    $targetFile = Join-Path $Root '.ylven-local\server-build-target.txt'
    if (-not $SshTarget -and (Test-Path -LiteralPath $targetFile)) {
        $SshTarget = (Get-Content -Raw -LiteralPath $targetFile).Trim()
    }
    if (-not $SshTarget) { throw 'Set -SshTarget, YLVEN_SSH_TARGET, or .ylven-local/server-build-target.txt.' }
    if ($SshTarget -notmatch '^[A-Za-z0-9._-]+$') { throw 'SSH target must be a configured alias, not an inline host command.' }

    $resolvedVersion = (& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py' --phase $Phase --print-version 2>$null | Select-Object -Last 1)
    if ($LASTEXITCODE -ne 0 -or -not $resolvedVersion) { throw "Could not resolve the release version for $Phase." }
    if ($Version -and $Version -ne $resolvedVersion) { throw "Version mismatch: expected $resolvedVersion, got $Version." }
    $Version = $resolvedVersion.Trim()

    & git diff --quiet -- . ':(exclude)status/PROJECT_STATE_SUMMARY.md'
    if ($LASTEXITCODE -ne 0) { throw 'Online-server release build requires a clean Git worktree.' }
    & git diff --cached --quiet -- . ':(exclude)status/PROJECT_STATE_SUMMARY.md'
    if ($LASTEXITCODE -ne 0) { throw 'Online-server release build requires a clean Git index.' }
    $Commit = (& git rev-parse HEAD).Trim()
    if ($Commit -notmatch '^[0-9a-f]{40}$') { throw 'Could not resolve the exact Git commit.' }

    if (-not $PreviousApk) {
        $upgradeFrom = (& (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py' --phase $Phase --print-upgrade-from 2>$null | Select-Object -Last 1).Trim()
        if ($upgradeFrom -eq 'NONE') { throw 'P00 requires an explicit prior same-package APK for reinstall verification.' }
        $previousPhaseIndex = [int]$Phase.Substring(1) - 1
        $previousPhase = 'P{0:D2}' -f $previousPhaseIndex
        $candidate = Get-ChildItem -LiteralPath (Join-Path $Root "dist\releases\$previousPhase") -Filter "YLVEN-$upgradeFrom-$previousPhase.apk" -File -ErrorAction SilentlyContinue | Select-Object -First 1
        if (-not $candidate) { throw "Previous owner APK $previousPhase/$upgradeFrom is missing." }
        $PreviousApk = $candidate.FullName
    }
    $PreviousApk = (Resolve-Path -LiteralPath $PreviousApk).Path

    $buildId = ('{0}-{1}-{2}-{3}' -f $Phase.ToLowerInvariant(), $Version, $Commit.Substring(0,12), (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ'))
    $localRoot = Join-Path $Root ".ylven-local\server-builds\$buildId"
    $incoming = Join-Path $localRoot 'incoming'
    $download = Join-Path $localRoot 'download'
    New-Item -ItemType Directory -Path $incoming,$download -Force | Out-Null
    $archive = Join-Path $incoming 'source.tar'
    & git archive --format=tar --output=$archive $Commit
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $archive)) { throw 'git archive failed.' }
    $archiveSha = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()

    $projectRows = & ssh $SshTarget 'android-build projects'
    if ($LASTEXITCODE -ne 0) { throw 'Could not query the online server build coordinator.' }
    $projectRow = $projectRows | Where-Object { $_ -like 'ylven|*' } | Select-Object -First 1
    if (-not $projectRow -or $projectRow -notmatch 'run_root=([^|]+)') { throw 'The online server has no registered YLVEN build root.' }
    $remoteRoot = $Matches[1]
    if ($remoteRoot -notmatch '^/[A-Za-z0-9._/-]+$') { throw 'The registered remote build root is unsafe.' }

    $remoteIncoming = "$remoteRoot/incoming/$buildId"
    $remoteWorkspace = "$remoteRoot/workspaces/$buildId"
    $remoteArtifacts = "$remoteRoot/artifacts/$buildId"
    $remoteArchive = "$remoteIncoming/source.tar"
    $remotePrevious = "$remoteIncoming/previous.apk"
    $remoteSigning = "$remoteRoot/signing"
    $remoteGradleCache = "$remoteRoot/gradle-cache"
    $createCommand = "set -eu; mkdir -p '$remoteIncoming' '$remoteWorkspace' '$remoteArtifacts'"
    & ssh $SshTarget $createCommand
    if ($LASTEXITCODE -ne 0) { throw 'Could not create isolated online-server build directories.' }
    $localChunkRoot = Join-Path $incoming 'chunks'
    Copy-FileToOnlineServer -SourcePath $archive -SshTarget $SshTarget -RemotePath $remoteArchive -LocalChunkRoot $localChunkRoot
    Copy-FileToOnlineServer -SourcePath $PreviousApk -SshTarget $SshTarget -RemotePath $remotePrevious -LocalChunkRoot $localChunkRoot

    $remoteCommand = @"
set -euo pipefail
echo '$archiveSha  $remoteArchive' | sha256sum -c -
printf '%s\n' '$archiveSha' > '$remoteWorkspace/source-archive.sha256'
tar -xf '$remoteArchive' -C '$remoteWorkspace'
bash '$remoteWorkspace/scripts/49_RUN_ONLINE_SERVER_BUILD.sh' '$remoteWorkspace' '$remoteArtifacts' '$remoteGradleCache' '$remoteSigning' '$remotePrevious' '$Phase' '$Version' '$Commit'
"@
    & ssh $SshTarget $remoteCommand
    if ($LASTEXITCODE -ne 0) { throw 'Online-server Android build failed. No APK is eligible for device testing.' }

    & scp -r "${SshTarget}:$remoteArtifacts/." $download
    if ($LASTEXITCODE -ne 0) { throw 'Could not download the exact online-server build outputs.' }
    & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/51_VERIFY_SERVER_ANDROID_BUILD.py' --artifact-dir $download --phase $Phase --version $Version --commit $Commit --source-archive-sha256 $archiveSha
    if ($LASTEXITCODE -ne 0) { throw 'Downloaded online-server build verification failed.' }
    Write-Output "SERVER_BUILD_DIR=$download"
} finally {
    Pop-Location
}
