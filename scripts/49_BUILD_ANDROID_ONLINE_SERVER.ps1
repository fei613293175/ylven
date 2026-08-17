param(
    [Parameter(Mandatory=$true)][ValidatePattern('^P\d{2}$')][string]$Phase,
    [string]$Version,
    [string]$SshTarget,
    [string]$PreviousApk
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$SshConnectionOptions = @('-o','BatchMode=yes','-o','ConnectTimeout=15','-o','ServerAliveInterval=15','-o','ServerAliveCountMax=4')

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
    $maxChunkAttempts = 5
    New-Item -ItemType Directory -Path $chunkDirectory -Force | Out-Null

    $prepareCommand = "set -eu; mkdir -p '$remoteChunkDirectory'"
    & ssh @SshConnectionOptions $SshTarget $prepareCommand
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
            $uploaded = $false
            for ($attempt = 1; $attempt -le $maxChunkAttempts -and -not $uploaded; $attempt++) {
                & scp -o BatchMode=yes -o ConnectTimeout=15 -o ServerAliveInterval=15 -o ServerAliveCountMax=4 -o IPQoS=throughput $localPart "${SshTarget}:$remoteChunkDirectory/$partName"
                $uploaded = $LASTEXITCODE -eq 0
                if (-not $uploaded -and $attempt -lt $maxChunkAttempts) {
                    Write-Warning "Chunk $partName upload interrupted; retrying ($attempt/$maxChunkAttempts)."
                    Start-Sleep -Seconds 2
                }
            }
            if (-not $uploaded) { throw "Could not upload chunk $partName for $RemotePath after $maxChunkAttempts attempts." }
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
    & ssh @SshConnectionOptions $SshTarget ($assembleCommand -replace "`r`n", "`n")
    if ($LASTEXITCODE -ne 0) { throw "Server-side reconstruction or SHA-256 verification failed for $RemotePath." }
}

function Copy-DirectoryFromOnlineServer {
    param(
        [Parameter(Mandatory=$true)][string]$SshTarget,
        [Parameter(Mandatory=$true)][string]$RemoteDirectory,
        [Parameter(Mandatory=$true)][string]$LocalDirectory,
        [Parameter(Mandatory=$true)][string]$LocalChunkRoot
    )

    # The server closes long SCP streams; transfer one compressed directory archive in bounded chunks.
    $remoteDirectory = $RemoteDirectory.TrimEnd('/')
    $remoteSeparator = $remoteDirectory.LastIndexOf('/')
    if ($remoteSeparator -le 0 -or $remoteSeparator -eq $remoteDirectory.Length - 1) { throw "Unsafe online-server artifact path: $RemoteDirectory" }
    $remoteArchive = "$remoteDirectory.download.tar.gz"
    $remoteParts = "$remoteArchive.parts"
    # These are Linux paths for SSH; do not use Windows Split-Path, which changes '/' to '\'.
    $remoteParent = $remoteDirectory.Substring(0, $remoteSeparator)
    $remoteName = $remoteDirectory.Substring($remoteSeparator + 1)
    $localChunkDirectory = Join-Path $LocalChunkRoot 'artifact-download'
    $localArchive = Join-Path $LocalChunkRoot 'artifacts.tar.gz'
    $chunkSize = 4MB
    $maxChunkAttempts = 5

    New-Item -ItemType Directory -Path $localChunkDirectory,$LocalDirectory -Force | Out-Null
    $prepareCommand = @"
set -eu
rm -f '$remoteArchive'
rm -rf '$remoteParts'
mkdir -p '$remoteParts'
tar -czf '$remoteArchive' -C '$remoteParent' '$remoteName'
split -b $chunkSize -d -a 6 '$remoteArchive' '$remoteParts/part-'
sha256sum '$remoteArchive' | cut -d ' ' -f 1
"@
    $prepareOutput = @(& ssh @SshConnectionOptions $SshTarget ($prepareCommand -replace "`r`n", "`n"))
    if ($LASTEXITCODE -ne 0) { throw "Could not prepare the chunked online-server artifact download:`n$($prepareOutput -join "`n")" }
    $remoteArchiveHash = (($prepareOutput | Select-Object -Last 1) -as [string]).Trim().ToLowerInvariant()
    if ($remoteArchiveHash -notmatch '^[0-9a-f]{64}$') { throw 'Online-server artifact archive did not return a SHA-256.' }

    $remotePartsCommand = "set -eu; find '$remoteParts' -maxdepth 1 -type f -name 'part-*' -printf '%f\n' | sort"
    $remotePartNames = @(& ssh @SshConnectionOptions $SshTarget $remotePartsCommand | Where-Object { $_ -match '^part-[0-9]{6}$' })
    if ($LASTEXITCODE -ne 0 -or $remotePartNames.Count -eq 0) { throw 'The online-server artifact archive has no downloadable chunks.' }

    foreach ($partName in $remotePartNames) {
        $localPart = Join-Path $localChunkDirectory $partName
        $downloaded = $false
        for ($attempt = 1; $attempt -le $maxChunkAttempts -and -not $downloaded; $attempt++) {
            & scp -o BatchMode=yes -o ConnectTimeout=15 -o ServerAliveInterval=15 -o ServerAliveCountMax=4 -o IPQoS=throughput "${SshTarget}:$remoteParts/$partName" $localPart
            $downloaded = $LASTEXITCODE -eq 0
            if (-not $downloaded -and $attempt -lt $maxChunkAttempts) {
                Write-Warning "Artifact chunk $partName download interrupted; retrying ($attempt/$maxChunkAttempts)."
                Start-Sleep -Seconds 2
            }
        }
        if (-not $downloaded) { throw "Could not download artifact chunk $partName after $maxChunkAttempts attempts." }
    }

    $archiveOutput = [System.IO.File]::Open($localArchive, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
    try {
        foreach ($partName in $remotePartNames) {
            $partPath = Join-Path $localChunkDirectory $partName
            $partInput = [System.IO.File]::OpenRead($partPath)
            try { $partInput.CopyTo($archiveOutput) } finally { $partInput.Dispose() }
        }
    } finally { $archiveOutput.Dispose() }
    $localArchiveHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $localArchive).Hash.ToLowerInvariant()
    if ($localArchiveHash -ne $remoteArchiveHash) { throw "Downloaded artifact archive SHA-256 mismatch: expected $remoteArchiveHash, got $localArchiveHash." }

    & tar -xzf $localArchive -C $LocalDirectory
    if ($LASTEXITCODE -ne 0) { throw 'Could not extract the exact online-server artifact archive.' }
    $extractedRoot = Join-Path $LocalDirectory $remoteName
    if (-not (Test-Path -LiteralPath $extractedRoot -PathType Container)) { throw 'Extracted online-server artifact root is missing.' }
    Get-ChildItem -LiteralPath $extractedRoot -Force | Move-Item -Destination $LocalDirectory -Force
    Remove-Item -LiteralPath $extractedRoot -Force -Recurse
    [ordered]@{ remote_archive=$remoteArchive; remote_archive_sha256=$remoteArchiveHash; local_archive_sha256=$localArchiveHash; chunk_count=$remotePartNames.Count } |
        ConvertTo-Json | Set-Content -LiteralPath (Join-Path (Split-Path -Path $LocalDirectory -Parent) '线上产物归档下载校验证明.json') -Encoding UTF8
}

function Remove-StaleLocalServerBuilds {
    param(
        [Parameter(Mandatory=$true)][string]$BuildRoot,
        [int]$KeepCompletedBuilds = 1
    )

    if ($KeepCompletedBuilds -lt 0) { throw 'KeepCompletedBuilds must not be negative.' }
    if (-not (Test-Path -LiteralPath $BuildRoot -PathType Container)) { return }
    $resolvedRoot = (Resolve-Path -LiteralPath $BuildRoot).Path.TrimEnd('\') + '\'
    $candidates = Get-ChildItem -LiteralPath $resolvedRoot -Directory -ErrorAction Stop |
        Where-Object { $_.Name -match '^p\d{2}-' }
    foreach ($candidate in $candidates) {
        $resolvedCandidate = (Resolve-Path -LiteralPath $candidate.FullName).Path
        if (-not $resolvedCandidate.StartsWith($resolvedRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove a build outside ${resolvedRoot}: $resolvedCandidate"
        }
        # A partial upload or a scheduler-rejected build has no verified source
        # record and must never consume one of the two retained artifact slots.
        if (-not (Test-Path -LiteralPath (Join-Path $resolvedCandidate 'download\服务器构建来源证明.json') -PathType Leaf)) {
            Remove-Item -LiteralPath $resolvedCandidate -Recurse -Force
        }
    }
    Get-ChildItem -LiteralPath $resolvedRoot -Directory -ErrorAction Stop |
        Where-Object { $_.Name -match '^p\d{2}-' -and (Test-Path -LiteralPath (Join-Path $_.FullName 'download\服务器构建来源证明.json') -PathType Leaf) } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -Skip $KeepCompletedBuilds |
        ForEach-Object {
            $resolvedCandidate = (Resolve-Path -LiteralPath $_.FullName).Path
            if (-not $resolvedCandidate.StartsWith($resolvedRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
                throw "Refusing to remove a build outside ${resolvedRoot}: $resolvedCandidate"
            }
            Remove-Item -LiteralPath $resolvedCandidate -Recurse -Force
        }
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
    $localBuildHistory = Join-Path $Root '.ylven-local\server-builds'
    # Keep one verified predecessor before starting; the candidate created below
    # is the second and final retained local server-build artifact.
    Remove-StaleLocalServerBuilds -BuildRoot $localBuildHistory -KeepCompletedBuilds 1
    $localRoot = Join-Path $localBuildHistory $buildId
    $incoming = Join-Path $localRoot 'incoming'
    $download = Join-Path $localRoot 'download'
    New-Item -ItemType Directory -Path $incoming,$download -Force | Out-Null
    $archive = Join-Path $incoming 'source.tar.gz'
    # Disable platform defaults; committed attributes define the deployed bytes for
    # append-only migrations, including the legacy CRLF migration set.
    & git -c core.autocrlf=false -c core.eol=lf archive --format=tar.gz --output=$archive $Commit
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $archive)) { throw 'git archive failed.' }
    $archiveSha = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()

    $projectRows = & ssh @SshConnectionOptions $SshTarget 'android-build projects'
    if ($LASTEXITCODE -ne 0) { throw 'Could not query the online server build coordinator.' }
    $projectRow = $projectRows | Where-Object { $_ -like 'ylven|*' } | Select-Object -First 1
    if (-not $projectRow -or $projectRow -notmatch 'run_root=([^|]+)') { throw 'The online server has no registered YLVEN build root.' }
    $remoteRoot = $Matches[1]
    if ($remoteRoot -notmatch '^/[A-Za-z0-9._/-]+$') { throw 'The registered remote build root is unsafe.' }

    $remoteIncoming = "$remoteRoot/incoming/$buildId"
    $remoteWorkspace = "$remoteRoot/workspaces/$buildId"
    $remoteArtifacts = "$remoteRoot/artifacts/$buildId"
    $remoteArchive = "$remoteIncoming/source.tar.gz"
    $remotePrevious = "$remoteIncoming/previous.apk"
    $remoteSigning = "$remoteRoot/signing"
    $remoteGradleCache = "$remoteRoot/gradle-cache"
    $createCommand = "set -eu; mkdir -p '$remoteIncoming' '$remoteWorkspace' '$remoteArtifacts'"
    & ssh @SshConnectionOptions $SshTarget $createCommand
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
    & ssh @SshConnectionOptions $SshTarget ($remoteCommand -replace "`r`n", "`n")
    if ($LASTEXITCODE -ne 0) { throw 'Online-server Android build failed. No APK is eligible for device testing.' }

    Copy-DirectoryFromOnlineServer -SshTarget $SshTarget -RemoteDirectory $remoteArtifacts -LocalDirectory $download -LocalChunkRoot (Join-Path $localRoot 'download-chunks')
    & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/51_VERIFY_SERVER_ANDROID_BUILD.py' --artifact-dir $download --phase $Phase --version $Version --commit $Commit --source-archive-sha256 $archiveSha
    if ($LASTEXITCODE -ne 0) { throw 'Downloaded online-server build verification failed.' }
    Write-Output "SERVER_BUILD_DIR=$download"
} finally {
    Pop-Location
}
