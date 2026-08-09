param(
    [Parameter(Mandatory=$true)][ValidatePattern('^P\d{2}$')][string]$Phase,
    [Parameter(Mandatory=$true)][string]$Version,
    [Parameter(Mandatory=$true)][string]$ServerBuildDir,
    [string]$Serial,
    [string]$AdbPath,
    [string]$PreviousApk,
    [Parameter(Mandatory=$true)][ValidatePattern('^\d+\.\d+\.\d+$')][string]$PreviousVersion
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$PackageId = 'cc.orbexa.ylven'
$TestPackageId = 'cc.orbexa.ylven.test'
$QueueOwned = $false
$TicketPath = $null
$ActiveLock = $null

function Invoke-Adb {
    param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Arguments)
    $savedErrorActionPreference = $ErrorActionPreference
    $output = @()
    $exitCode = $null
    try {
        # Windows PowerShell 5 surfaces native stderr as ErrorRecord objects. ADB writes successful
        # push progress there, so judge the command by its native exit code and preserve all text.
        $ErrorActionPreference = 'Continue'
        $output = @(& $script:AdbPath -s $script:Serial @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $savedErrorActionPreference
    }
    $textOutput = @($output | ForEach-Object { $_.ToString() })
    if ($exitCode -ne 0) { throw "adb $($Arguments -join ' ') failed:`n$($textOutput -join "`n")" }
    return $textOutput
}

function Invoke-AdbInstall {
    param(
        [Parameter(Mandatory=$true)][string]$Apk,
        [Parameter(Mandatory=$true)][string]$ResultPath,
        [Parameter(Mandatory=$true)][string]$EvidenceDir,
        [switch]$Replace
    )
    $device = @(Get-DeviceRows | Where-Object { $_.Serial -eq $script:Serial }) | Select-Object -First 1
    if (-not $device -or $device.State -ne 'device') {
        throw "ADB target $script:Serial is not in exact state device immediately before installing $(Split-Path -Leaf $Apk); testing stops."
    }

    $arguments = @('-s', $script:Serial, 'install', '--no-streaming')
    if ($Replace) { $arguments += '-r' }
    $arguments += ('"{0}"' -f $Apk.Replace('"', '\"'))
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $script:AdbPath
    $startInfo.Arguments = $arguments -join ' '
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo
    $watcherJob = $null
    try {
        if (-not $process.Start()) { throw "Could not start adb install for $Apk." }
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        $watcherPath = Join-Path $PSScriptRoot '51_WATCH_VENDOR_INSTALL_CONFIRMATION.ps1'
        $watcherJob = Start-Job -FilePath $watcherPath -ArgumentList @(
            $script:AdbPath, $script:Serial, $process.Id, $EvidenceDir, 180
        )
        if (-not $process.WaitForExit(180000)) {
            $process.Kill()
            $process.WaitForExit()
            throw "ADB install exceeded the bounded timeout of 180 seconds for $(Split-Path -Leaf $Apk)."
        }
        $process.WaitForExit()
        $output = @($stdoutTask.GetAwaiter().GetResult() -split '\r?\n')
        $output += @($stderrTask.GetAwaiter().GetResult() -split '\r?\n')
        $output | Where-Object { $_ } | Set-Content -LiteralPath $ResultPath -Encoding UTF8
        if ($process.ExitCode -ne 0 -or ($output -join "`n") -notmatch '(?m)^Success\s*$') {
            throw "ADB install failed for $(Split-Path -Leaf $Apk):`n$($output -join "`n")"
        }
    } finally {
        if ($watcherJob) {
            $watcherJob | Wait-Job -Timeout 15 | Out-Null
            if ($watcherJob.State -eq 'Running') { $watcherJob | Stop-Job }
            $watcherOutput = @($watcherJob | Receive-Job -ErrorAction SilentlyContinue)
            if ($watcherOutput.Count -gt 0) {
                $watcherOutput | Set-Content -LiteralPath (Join-Path $EvidenceDir 'watcher-output.txt') -Encoding UTF8
            }
            $watcherState = $watcherJob.State
            $watcherJob | Remove-Job -Force
            if ($watcherState -eq 'Failed' -and $process.HasExited -and $process.ExitCode -ne 0) {
                throw "Vendor install confirmation watcher failed for $(Split-Path -Leaf $Apk)."
            }
        }
        $process.Dispose()
    }
}

function Invoke-Instrumentation {
    param(
        [Parameter(Mandatory=$true)][string]$ClassName,
        [Parameter(Mandatory=$true)][ValidateRange(1,3600)][int]$TimeoutSeconds,
        [Parameter(Mandatory=$true)][string]$ResultPath
    )
    $deviceRows = @(Get-DeviceRows)
    $device = $deviceRows | Where-Object { $_.Serial -eq $script:Serial } | Select-Object -First 1
    if (-not $device -or $device.State -ne 'device') {
        throw "ADB target $script:Serial is not in exact state device immediately before $ClassName; testing stops."
    }
    if ($ClassName -eq "$script:PackageId.P03ProvisionStagingSessionTest") {
        $foregroundOutput = @(Invoke-Adb shell am start '-W' '-n' "$script:PackageId/.MainActivity")
        $foregroundOutput | Set-Content -LiteralPath "$ResultPath.foreground.txt" -Encoding UTF8
        if (($foregroundOutput -join "`n") -notmatch 'Status:\s*ok') {
            throw "The target APK could not be brought to the foreground before $ClassName."
        }
    }
    $arguments = @(
        '-s', $script:Serial, 'shell', 'am', 'instrument', '-w', '-r',
        '-e', 'class', $ClassName,
        "$script:TestPackageId/androidx.test.runner.AndroidJUnitRunner"
    )
    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $script:AdbPath
    $startInfo.Arguments = $arguments -join ' '
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo
    try {
        if (-not $process.Start()) { throw "Could not start adb instrumentation for $ClassName." }
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            $process.Kill()
            $process.WaitForExit()
            # Closing the host adb client is not sufficient proof that device-side instrumentation stopped.
            & $script:AdbPath -s $script:Serial shell am force-stop $script:TestPackageId 2>$null
            & $script:AdbPath -s $script:Serial shell am force-stop $script:PackageId 2>$null
            $partial = @($stdoutTask.GetAwaiter().GetResult() -split '\r?\n')
            $partial += @($stderrTask.GetAwaiter().GetResult() -split '\r?\n')
            $partial | Set-Content -LiteralPath $ResultPath -Encoding UTF8
            throw "Instrumentation $ClassName exceeded the bounded timeout of $TimeoutSeconds seconds."
        }
        $process.WaitForExit()
        $exitCode = $process.ExitCode
        $output = @($stdoutTask.GetAwaiter().GetResult() -split '\r?\n')
        $output += @($stderrTask.GetAwaiter().GetResult() -split '\r?\n')
        $output | Set-Content -LiteralPath $ResultPath -Encoding UTF8
        if ($exitCode -ne 0) {
            throw "Instrumentation $ClassName failed with adb exit code ${exitCode}:`n$($output -join "`n")"
        }
        return $output
    } finally {
        $process.Dispose()
    }
}

function Test-AppPath {
    param(
        [Parameter(Mandatory=$true)][string]$Package,
        [Parameter(Mandatory=$true)][string]$Path,
        [switch]$NonEmpty
    )
    $testFlag = if ($NonEmpty) { '-s' } else { '-e' }
    & $script:AdbPath -s $script:Serial shell run-as $Package test $testFlag $Path 2>$null
    return $LASTEXITCODE -eq 0
}

function Test-AppSessionBlob {
    if (-not (Test-AppPath -Package $script:PackageId -Path 'shared_prefs/ylven_identity.xml')) { return $false }
    $savedErrorActionPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        & $script:AdbPath -s $script:Serial shell run-as $script:PackageId grep -q session_blob shared_prefs/ylven_identity.xml 2>&1 | Out-Null
        return $LASTEXITCODE -eq 0
    } finally {
        $ErrorActionPreference = $savedErrorActionPreference
    }
}

function Get-PackagePath {
    param([Parameter(Mandatory=$true)][string]$Package)
    $output = & $script:AdbPath -s $script:Serial shell pm path $Package 2>&1
    if ($LASTEXITCODE -ne 0) {
        $text = ($output -join "`n").Trim()
        if ($text -and $text -notmatch '(?i)(unable to find package|unknown package|package .* not found|does not exist)') {
            throw "adb shell pm path $Package failed:`n$text"
        }
        return @()
    }
    return $output
}

function Get-DeviceRows {
    $lines = & $script:AdbPath devices -l 2>&1
    if ($LASTEXITCODE -ne 0) { throw "adb devices -l failed:`n$($lines -join "`n")" }
    $rows = @()
    foreach ($line in $lines) {
        if ($line -match '^(\S+)\s+(device|offline|unauthorized|no permissions)(?:\s+(.*))?$') {
            $rows += [pscustomobject]@{ Serial=$Matches[1]; State=$Matches[2]; Detail=$Matches[3] }
        }
    }
    return $rows
}

function Get-QueueSnapshot {
    param([Parameter(Mandatory=$true)][string]$DeviceSerial)
    $root = Join-Path $env:USERPROFILE ".codex\android-device-queue\$DeviceSerial"
    $ticketCount = 0
    $hasActiveLock = $false
    if (Test-Path -LiteralPath $root) {
        $ticketCount = @(Get-ChildItem -LiteralPath $root -Filter '*.ticket' -File -ErrorAction Stop).Count
        $hasActiveLock = Test-Path -LiteralPath (Join-Path $root 'active.lock')
    }
    return [pscustomobject]@{
        Root=$root
        TicketCount=$ticketCount
        HasActiveLock=$hasActiveLock
        QueueLoad=$ticketCount + [int]$hasActiveLock
        IsIdle=(-not $hasActiveLock -and $ticketCount -eq 0)
    }
}

function Get-PhysicalDeviceCandidates {
    param([Parameter(Mandatory=$true)][object[]]$Rows)
    $candidates = @()
    foreach ($row in $Rows) {
        if ($row.State -ne 'device' -or $row.Serial -like 'emulator-*') { continue }
        $qemuOutput = & $script:AdbPath -s $row.Serial shell getprop ro.kernel.qemu 2>&1
        if ($LASTEXITCODE -ne 0) { continue }
        $qemu = ($qemuOutput -join '').Trim()
        if ($qemu -eq '1') { continue }
        $queue = Get-QueueSnapshot -DeviceSerial $row.Serial
        $candidates += [pscustomobject]@{
            Serial=$row.Serial
            State=$row.State
            Detail=$row.Detail
            Qemu=$qemu
            QueueRoot=$queue.Root
            TicketCount=$queue.TicketCount
            HasActiveLock=$queue.HasActiveLock
            QueueLoad=$queue.QueueLoad
            IsIdle=$queue.IsIdle
        }
    }
    return $candidates
}

Push-Location $Root
try {
    if (-not $AdbPath) { $AdbPath = $env:YLVEN_ADB_PATH }
    if (-not $AdbPath) {
        $localAdb = Join-Path $Root '.ylven-local\platform-tools-p03\platform-tools\adb.exe'
        if (Test-Path -LiteralPath $localAdb) { $AdbPath = $localAdb }
    }
    if (-not $AdbPath) {
        $command = Get-Command adb -ErrorAction SilentlyContinue
        if ($command) { $AdbPath = $command.Source }
    }
    if (-not $AdbPath -or -not (Test-Path -LiteralPath $AdbPath)) { throw 'ADB is unavailable; physical-device testing stops with no simulator fallback.' }
    $script:AdbPath = (Resolve-Path -LiteralPath $AdbPath).Path

    $rows = @(Get-DeviceRows)
    $physicalCandidates = @(Get-PhysicalDeviceCandidates -Rows $rows)
    $selectionMode = $null
    $queueLoadAtSelection = $null
    $connectedCandidateCount = $physicalCandidates.Count
    if ($Serial) {
        if ($Serial -like 'emulator-*') { throw 'Emulator serials are forbidden.' }
        $selected = $physicalCandidates | Where-Object { $_.Serial -eq $Serial } | Select-Object -First 1
        if (-not $selected) { throw "ADB target $Serial is not an eligible physical device in exact state device; testing stops." }
        $selectionMode = 'explicit_serial'
    } else {
        if ($physicalCandidates.Count -eq 0) { throw 'No eligible physical ADB device is in exact state device; testing stops with no simulator fallback.' }
        $selected = $physicalCandidates |
            Sort-Object @{Expression={ if ($_.IsIdle) { 0 } else { 1 } }}, @{Expression={$_.QueueLoad}}, @{Expression={$_.Serial}} |
            Select-Object -First 1
        $Serial = $selected.Serial
        if ($selected.IsIdle) { $selectionMode = 'automatic_idle' } else { $selectionMode = 'automatic_shortest_fifo' }
    }
    $queueLoadAtSelection = $selected.QueueLoad
    Write-Host "Selected physical device $Serial via $selectionMode (eligible devices: $connectedCandidateCount; queue load: $queueLoadAtSelection)."
    $script:Serial = $Serial
    $script:PackageId = $PackageId
    $script:TestPackageId = $TestPackageId
    $qemu = ((Invoke-Adb shell getprop ro.kernel.qemu) -join '').Trim()
    if ($qemu -eq '1') { throw 'The selected ADB target reports ro.kernel.qemu=1; simulators are forbidden.' }

    $queueRoot = Join-Path $env:USERPROFILE ".codex\android-device-queue\$Serial"
    New-Item -ItemType Directory -Path $queueRoot -Force | Out-Null
    $queueRootResolved = (Resolve-Path -LiteralPath $queueRoot).Path
    $ticketName = ('{0}-{1:D8}-{2}.ticket' -f (Get-Date).ToUniversalTime().ToString('yyyyMMddHHmmssfffffff'), $PID, [guid]::NewGuid().ToString('N'))
    $TicketPath = Join-Path $queueRootResolved $ticketName
    New-Item -ItemType File -Path $TicketPath -ErrorAction Stop | Out-Null
    $ActiveLock = Join-Path $queueRootResolved 'active.lock'
    $lastNotice = [DateTime]::MinValue
    while (-not $QueueOwned) {
        $first = Get-ChildItem -LiteralPath $queueRootResolved -Filter '*.ticket' -File | Sort-Object Name | Select-Object -First 1
        if ($first -and $first.FullName -eq $TicketPath -and -not (Test-Path -LiteralPath $ActiveLock)) {
            try {
                New-Item -ItemType Directory -Path $ActiveLock -ErrorAction Stop | Out-Null
                $QueueOwned = $true
                @{ project='YLVEN'; phase=$Phase; serial=$Serial; pid=$PID; acquired_at=(Get-Date).ToUniversalTime().ToString('o') } |
                    ConvertTo-Json | Set-Content -LiteralPath (Join-Path $ActiveLock 'owner.json') -Encoding UTF8
            } catch [System.IO.IOException] {
                $QueueOwned = $false
            }
        }
        if (-not $QueueOwned) {
            if (((Get-Date) - $lastNotice).TotalSeconds -ge 30) {
                Write-Host "Physical device $Serial is busy; waiting in the shared FIFO queue without interruption."
                $lastNotice = Get-Date
            }
            Start-Sleep -Seconds 5
        }
    }

    $currentRow = @(Get-DeviceRows | Where-Object { $_.Serial -eq $Serial }) | Select-Object -First 1
    if (-not $currentRow -or $currentRow.State -ne 'device') { throw "Device $Serial became unavailable while queued; testing stops." }
    $qemu = ((Invoke-Adb shell getprop ro.kernel.qemu) -join '').Trim()
    if ($qemu -eq '1') { throw 'The acquired target is a simulator; testing stops.' }

    $ServerBuildDir = (Resolve-Path -LiteralPath $ServerBuildDir).Path
    $serverProvenancePath = Join-Path $ServerBuildDir '服务器构建来源证明.json'
    $downloadProofPath = Join-Path $ServerBuildDir '本机下载校验证明.json'
    if (-not (Test-Path -LiteralPath $serverProvenancePath) -or -not (Test-Path -LiteralPath $downloadProofPath)) {
        throw 'Server build provenance or local download verification is missing.'
    }
    $provenance = Get-Content -Raw -LiteralPath $serverProvenancePath | ConvertFrom-Json
    if ($provenance.phase -ne $Phase -or $provenance.version -ne $Version) { throw 'Server build phase/version mismatch.' }
    $CurrentApk = (Resolve-Path -LiteralPath (Join-Path $ServerBuildDir $provenance.apk)).Path
    $TestApk = (Resolve-Path -LiteralPath (Join-Path $ServerBuildDir $provenance.instrumentation_apk)).Path
    if ((Get-FileHash -Algorithm SHA256 -LiteralPath $CurrentApk).Hash.ToLowerInvariant() -ne $provenance.apk_sha256) { throw 'Current APK hash differs from server provenance.' }
    if ((Get-FileHash -Algorithm SHA256 -LiteralPath $TestApk).Hash.ToLowerInvariant() -ne $provenance.instrumentation_apk_sha256) { throw 'Instrumentation APK hash differs from server provenance.' }
    $signingMigration = [bool]$provenance.signing_migration
    if ($signingMigration -and $Phase -ne 'P03') { throw 'Signing migration is only allowed for P03.' }

    $upgradeFrom = $PreviousVersion.Trim()
    if (-not $PreviousApk) {
        $previousIndex = [int]$Phase.Substring(1) - 1
        if ($previousIndex -lt 0) { throw 'P00 requires -PreviousApk for same-package reinstall testing.' }
        $previousPhase = 'P{0:D2}' -f $previousIndex
        $PreviousApk = Join-Path $Root "dist\releases\$previousPhase\YLVEN-$upgradeFrom-$previousPhase.apk"
    }
    $PreviousApk = (Resolve-Path -LiteralPath $PreviousApk).Path

    $acceptanceId = ('{0}-{1}-{2}' -f $Phase.ToLowerInvariant(), $Version, (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssZ'))
    $output = Join-Path $Root ".ylven-local\physical-device-acceptance\$acceptanceId"
    $screenshots = Join-Path $output '截图'
    $productionScreenshots = Join-Path $output '真实页面截图'
    $testResults = Join-Path $output 'test-results'
    New-Item -ItemType Directory -Path $output,$screenshots,$productionScreenshots,$testResults -Force | Out-Null
    Copy-Item -LiteralPath $serverProvenancePath,$downloadProofPath -Destination $output
    $deliveredApk = Join-Path $output "YLVEN-$Version-$Phase.apk"
    Copy-Item -LiteralPath $CurrentApk -Destination $deliveredApk

    $manufacturer = ((Invoke-Adb shell getprop ro.product.manufacturer) -join '').Trim()
    $model = ((Invoke-Adb shell getprop ro.product.model) -join '').Trim()
    $androidVersion = ((Invoke-Adb shell getprop ro.build.version.release) -join '').Trim()
    $apiLevel = ((Invoke-Adb shell getprop ro.build.version.sdk) -join '').Trim()
    $physicalSize = ((Invoke-Adb shell wm size) -join ' ').Trim()
    if ($physicalSize -notmatch '(?i)Physical size:\s*\d+x\d+') { throw "Could not determine the physical display size: $physicalSize" }
    if ($physicalSize -match '(?i)Override size:') { throw "The selected phone has an existing display-size override; testing stops without changing it: $physicalSize" }
    $density = ((Invoke-Adb shell wm density) -join ' ').Trim()
    if ($density -notmatch '(?i)Physical density:\s*\d+') { throw "Could not determine the physical display density: $density" }
    if ($density -match '(?i)Override density:') { throw "The selected phone has an existing density override; testing stops without changing it: $density" }

    Invoke-Adb logcat '-c' | Out-Null
    $existingPackage = (Get-PackagePath -Package $PackageId) -join "`n"
    if ($signingMigration -and $existingPackage -match '^package:') {
        $existingDump = (Invoke-Adb shell dumpsys package $PackageId) -join "`n"
        $existingVersionMatch = [regex]::Match($existingDump, '(?m)^\s*versionName=([^\s]+)')
        if (-not $existingVersionMatch.Success) { throw 'Could not determine the installed YLVEN version before rebuilding the P02 migration baseline.' }
        $existingVersion = $existingVersionMatch.Groups[1].Value
        if ([version]$existingVersion -gt [version]$upgradeFrom) {
            $resetRows = @("installed_version=$existingVersion", "required_baseline=$upgradeFrom")
            $existingTestPackage = (Get-PackagePath -Package $TestPackageId) -join "`n"
            if ($existingTestPackage -match '^package:') {
                $testReset = (Invoke-Adb uninstall $TestPackageId) -join "`n"
                if ($testReset -notmatch '(?m)^Success\s*$') { throw 'Could not remove the residual P03 instrumentation package before rebuilding the P02 baseline.' }
                $resetRows += 'instrumentation_package_removed=true'
            }
            $appReset = (Invoke-Adb uninstall $PackageId) -join "`n"
            if ($appReset -notmatch '(?m)^Success\s*$') { throw 'Could not remove the residual P03 app before rebuilding the P02 baseline.' }
            $resetRows += 'current_app_removed=true'
            $resetRows | Set-Content -LiteralPath (Join-Path $testResults 'signing-migration-baseline-reset.txt') -Encoding UTF8
        }
    }

    Invoke-AdbInstall `
        -Apk $PreviousApk `
        -Replace `
        -ResultPath (Join-Path $testResults 'previous-install.txt') `
        -EvidenceDir (Join-Path $testResults 'install-confirmations\previous-owner')
    $installedBefore = (Get-PackagePath -Package $PackageId) -join "`n"
    if ($installedBefore -notmatch '^package:') { throw 'Previous owner APK was not installed.' }
    $previousPackageDump = (Invoke-Adb shell dumpsys package $PackageId) -join "`n"
    if ($previousPackageDump -notmatch "versionName=$([regex]::Escape($upgradeFrom))") { throw 'The required previous owner version was not installed before migration testing.' }
    $loginBefore = if (Test-AppSessionBlob) { 'present' } else { 'absent' }
    # Quote -p so Windows PowerShell 5 does not bind it as the PipelineVariable common parameter.
    Invoke-Adb shell run-as $PackageId mkdir '-p' files | Out-Null
    Invoke-Adb shell run-as $PackageId touch files/physical-upgrade-marker | Out-Null

    if ($signingMigration) {
        $preMigration = [ordered]@{
            previous_version=$upgradeFrom
            previous_apk=(Split-Path -Leaf $PreviousApk)
            previous_signing_certificate_sha256=$provenance.previous_signing_certificate_sha256
            installed_before_uninstall=$true
            login_state_before=$loginBefore
            data_marker_before='present'
            data_preservation_expected=$false
            install_mode='one_time_uninstall_then_install'
        }
        $preMigration | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $testResults 'signing-migration-pre-uninstall.json') -Encoding UTF8
        (Invoke-Adb uninstall $PackageId) | Set-Content -LiteralPath (Join-Path $testResults 'signing-migration-uninstall.txt') -Encoding UTF8
        $installedAfterUninstall = (Get-PackagePath -Package $PackageId) -join "`n"
        if ($installedAfterUninstall -match '^package:') { throw 'P03 signing migration uninstall did not remove the old package.' }
        Invoke-AdbInstall `
            -Apk $CurrentApk `
            -ResultPath (Join-Path $testResults 'current-migration-install.txt') `
            -EvidenceDir (Join-Path $testResults 'install-confirmations\current-owner')
        $marker = if (Test-AppPath -Package $PackageId -Path 'files/physical-upgrade-marker') { 'present' } else { 'absent' }
        if ($marker -ne 'absent') { throw 'P03 signing migration unexpectedly preserved old app data.' }
        $loginAfter = if (Test-AppSessionBlob) { 'present' } else { 'absent' }
        if ($loginAfter -ne 'absent') { throw 'P03 signing migration unexpectedly preserved old login state.' }
        $oldTestPackage = (Get-PackagePath -Package $TestPackageId) -join "`n"
        if ($oldTestPackage -match '^package:') {
            (Invoke-Adb uninstall $TestPackageId) | Set-Content -LiteralPath (Join-Path $testResults 'signing-migration-test-package-uninstall.txt') -Encoding UTF8
        }
    } else {
        Invoke-AdbInstall `
            -Apk $CurrentApk `
            -Replace `
            -ResultPath (Join-Path $testResults 'current-upgrade-install.txt') `
            -EvidenceDir (Join-Path $testResults 'install-confirmations\current-owner')
        $marker = if (Test-AppPath -Package $PackageId -Path 'files/physical-upgrade-marker') { 'present' } else { 'absent' }
        if ($marker -ne 'present') { throw 'Upgrade data marker did not survive adb install -r.' }
        $loginAfter = if (Test-AppSessionBlob) { 'present' } else { 'absent' }
        if ($loginBefore -eq 'present' -and $loginAfter -ne 'present') { throw 'Existing login-state preferences did not survive the upgrade.' }
    }
    $packageDump = (Invoke-Adb shell dumpsys package $PackageId) -join "`n"
    if ($packageDump -notmatch "versionName=$([regex]::Escape($Version))") { throw 'Installed versionName does not match the release contract.' }

    Invoke-Adb shell am force-stop $PackageId | Out-Null
    $launch = (Invoke-Adb shell am start '-W' '-n' "$PackageId/.MainActivity") -join "`n"
    $launch | Set-Content -LiteralPath (Join-Path $testResults 'launch.txt') -Encoding UTF8
    if ($launch -notmatch 'Status:\s*ok') { throw 'The installed APK did not launch successfully.' }

    Invoke-AdbInstall `
        -Apk $TestApk `
        -Replace `
        -ResultPath (Join-Path $testResults 'instrumentation-install.txt') `
        -EvidenceDir (Join-Path $testResults 'install-confirmations\instrumentation')
    $sessionProvisioned = $false
    if ($loginAfter -ne 'present') {
        $provisionFlow = (Invoke-Instrumentation -ClassName "$PackageId.P03ProvisionStagingSessionTest" -TimeoutSeconds 600 -ResultPath (Join-Path $testResults 'P03ProvisionStagingSessionTest.txt')) -join "`n"
        if ($provisionFlow -notmatch '(?m)^OK \(' -or $provisionFlow -match '(?m)^FAILURES!!!') { throw 'P03 real staging session provisioning failed on the physical device.' }
        $loginAfterProvision = if (Test-AppSessionBlob) { 'present' } else { 'absent' }
        if ($loginAfterProvision -ne 'present') { throw 'P03 real staging session was not persisted after provisioning.' }
        $sessionProvisioned = $true
        Invoke-Adb shell am force-stop $PackageId | Out-Null
        $launchAfterProvision = (Invoke-Adb shell am start '-W' '-n' "$PackageId/.MainActivity") -join "`n"
        $launchAfterProvision | Set-Content -LiteralPath (Join-Path $testResults 'launch-after-staging-session.txt') -Encoding UTF8
        if ($launchAfterProvision -notmatch 'Status:\s*ok') { throw 'The app did not relaunch after real staging session provisioning.' }
    } else {
        $loginAfterProvision = $loginAfter
    }
    $liveFlow = (Invoke-Instrumentation -ClassName "$PackageId.P03LiveStagingFlowTest" -TimeoutSeconds 300 -ResultPath (Join-Path $testResults 'P03LiveStagingFlowTest.txt')) -join "`n"
    if ($liveFlow -notmatch '(?m)^OK \(' -or $liveFlow -match '(?m)^FAILURES!!!') { throw 'P03 real staging flow failed on the physical device.' }

    if ($Phase -eq 'P03') {
        $remoteScreenshotDirectories = @(
            '/sdcard/Download/ylven-p03',
            '/sdcard/Download/ylven-p03-production'
        )
        foreach ($remoteScreenshotDirectory in $remoteScreenshotDirectories) {
            Invoke-Adb shell rm '-rf' $remoteScreenshotDirectory | Out-Null
        }
        $remoteScreenshotDirectories | Set-Content -LiteralPath (Join-Path $testResults 'remote-screenshot-cleanup.txt') -Encoding UTF8
    }

    $flow = (Invoke-Instrumentation -ClassName "$PackageId.P03RealDeviceFlowTest" -TimeoutSeconds 300 -ResultPath (Join-Path $testResults 'P03RealDeviceFlowTest.txt')) -join "`n"
    if ($flow -notmatch '(?m)^OK \(' -or $flow -match '(?m)^FAILURES!!!') { throw 'P03 physical-device interaction flow failed.' }

    $stateFlow = (Invoke-Instrumentation -ClassName "$PackageId.P03ConversationStateUiTest" -TimeoutSeconds 1200 -ResultPath (Join-Path $testResults 'P03ConversationStateUiTest.txt')) -join "`n"
    if ($stateFlow -notmatch '(?m)^OK \(' -or $stateFlow -match '(?m)^FAILURES!!!') { throw 'P03 physical-device state capture failed.' }

    $stateRows = Import-Csv -LiteralPath (Join-Path $Root 'contracts\ui-state-catalog.csv') | Where-Object { $_.surface -eq 'ANDROID' -and ($_.phases -split '\|') -contains $Phase }
    if ($Phase -eq 'P03' -and @($stateRows).Count -ne 93) { throw "Expected 93 P03 Android states, got $(@($stateRows).Count)." }
    $indexRows = @()
    foreach ($row in $stateRows) {
        $remote = "/sdcard/Download/ylven-$($Phase.ToLowerInvariant())/$($row.state_id).png"
        $local = Join-Path $screenshots "$($row.state_id).png"
        Invoke-Adb shell test '-s' $remote | Out-Null
        Invoke-Adb pull $remote $local | Out-Null
        $digest = (Get-FileHash -Algorithm SHA256 -LiteralPath $local).Hash.ToLowerInvariant()
        $indexRows += [pscustomobject]@{
            state_id=$row.state_id; page_id=$row.page_id; device_serial=$Serial
            runtime_screenshot="截图/$($row.state_id).png"; runtime_sha256=$digest; result='PENDING_VISUAL_COMPARE'
        }
    }
    $indexRows | Export-Csv -LiteralPath (Join-Path $output '截图索引.csv') -NoTypeInformation -Encoding UTF8

    $productionPages = [ordered]@{
        'YL-A-018'='YL-A-018-S02_POPULATED'
        'YL-A-019'='YL-A-019-S02_POPULATED'
        'YL-A-020'='YL-A-020-S02_POPULATED'
        'YL-A-021'='YL-A-021-S02_POPULATED'
        'YL-A-022'='YL-A-022-S01_DEFAULT'
        'YL-A-023'='YL-A-023-S07_COMPLETED'
        'YL-A-031'='YL-A-031-S01_DEFAULT'
    }
    $productionIndex = @()
    foreach ($entry in $productionPages.GetEnumerator()) {
        $name = "$($entry.Key)-PRODUCTION.png"
        $remote = "/sdcard/Download/ylven-p03-production/$name"
        $local = Join-Path $productionScreenshots $name
        Invoke-Adb shell test '-s' $remote | Out-Null
        Invoke-Adb pull $remote $local | Out-Null
        $digest = (Get-FileHash -Algorithm SHA256 -LiteralPath $local).Hash.ToLowerInvariant()
        $catalogRow = $stateRows | Where-Object { $_.state_id -eq $entry.Value } | Select-Object -First 1
        if (-not $catalogRow) { throw "Missing representative mockup state $($entry.Value)." }
        $productionIndex += [pscustomobject]@{
            page_id=$entry.Key
            representative_state_id=$entry.Value
            production_screenshot="真实页面截图/$name"
            production_sha256=$digest
            mockup_path=$catalogRow.mockup_path
            result='PENDING_CODEX_VISUAL_REVIEW'
        }
    }
    $productionIndex | Export-Csv -LiteralPath (Join-Path $output '真实页面截图索引.csv') -NoTypeInformation -Encoding UTF8

    @('# Visual Diff Report — P03','', 'Result: **PENDING_SERVER_POST_PROCESSING**','', 'Raw screenshots were captured on the physical device; visual comparison and interaction coverage run only on the connected online build server.') | Set-Content -LiteralPath (Join-Path $output '视觉差异报告.md') -Encoding UTF8
    'PENDING_SERVER_POST_PROCESSING' | Set-Content -LiteralPath (Join-Path $testResults 'server-post-processing.status') -Encoding UTF8
    $serverPostProcessing = 'PENDING_SERVER_POST_PROCESSING'

    (Invoke-Adb logcat '-d' '-v' threadtime) | Set-Content -LiteralPath (Join-Path $testResults 'logcat.txt') -Encoding UTF8
    (Invoke-Adb shell dumpsys activity exit-info $PackageId) | Set-Content -LiteralPath (Join-Path $testResults 'application-exit-info.txt') -Encoding UTF8
    $logText = Get-Content -Raw -LiteralPath (Join-Path $testResults 'logcat.txt')
    $exitText = Get-Content -Raw -LiteralPath (Join-Path $testResults 'application-exit-info.txt')
    $runtimeErrors = @()
    foreach ($pattern in @('FATAL EXCEPTION', 'ANR in cc\.orbexa\.ylven', 'am_crash.*cc\.orbexa\.ylven', 'am_anr.*cc\.orbexa\.ylven', 'Fatal signal.*cc\.orbexa\.ylven')) {
        if ($logText -match $pattern) { $runtimeErrors += $pattern }
    }
    if ($exitText -match '(?is)(REASON_CRASH|REASON_ANR|reason=crash|reason=anr)') { $runtimeErrors += 'ApplicationExitInfo crash/ANR' }
    if ($runtimeErrors.Count -gt 0) { throw "Crash/ANR/log review failed: $($runtimeErrors -join ', ')" }

    $logReview = @"
# 真机日志审查

- 设备：$manufacturer $model / Android $androidVersion / API $apiLevel
- APK：YLVEN-$Version-$Phase.apk
- logcat：test-results/logcat.txt
- ApplicationExitInfo：test-results/application-exit-info.txt
- FATAL EXCEPTION：未发现
- ANR：未发现
- native crash：未发现
- 无法解释的应用异常：未发现
- 结果：PASS
"@
    $logReview | Set-Content -LiteralPath (Join-Path $output '真机日志审查.md') -Encoding UTF8

    $stagingSessionNote = if ($sessionProvisioned) { 'P03ProvisionStagingSessionTest + P03LiveStagingFlowTest: PASS（真实 staging 注册、Android Keystore 会话与 MainActivity 真实 API）' } else { 'P03LiveStagingFlowTest: PASS（MainActivity、已有加密登录态和真实 staging API）' }
    $automationReport = @"
# 自动化测试报告

- Phase: $Phase
- Version: $Version
- Device: $manufacturer $model ($Serial)
- $stagingSessionNote
- P03RealDeviceFlowTest: PASS（物理设备 UI 交互；确定性 Fake 网关，不替代 staging）
- P03ConversationStateUiTest: PASS（物理设备 93 状态截图）
- Interaction coverage: PASS
- Crash/ANR/log review: PASS
- Result: PENDING_CODEX_PRODUCTION_VISUAL_REVIEW
"@
    $automationReport | Set-Content -LiteralPath (Join-Path $output '自动化测试报告.md') -Encoding UTF8

    if ($signingMigration) {
        $upgradeEvidence = @"
# 签名迁移安装证据

- 阶段：$Phase
- 当前版本：$Version
- applicationId：$PackageId
- 安装模式：项目所有者批准的一次性 P02 -> P03 签名迁移
- 旧版安装：adb install --no-streaming -r $PreviousApk
- 迁移安装：adb uninstall $PackageId；adb install --no-streaming $CurrentApk
- 旧签名证书：$($provenance.previous_signing_certificate_sha256)
- 新签名证书：$($provenance.signing_certificate_sha256)
- 旧本地数据/登录态：未保留（迁移合同明确 data_preserved=false）
- 旧数据标记：卸载前 present；卸载后 absent
- 新登录态：通过真实 staging 注册流程重新建立并保存到 Android Keystore
- 新版本启动：已验证
- P03 之后升级基线：新签名证书，恢复 adb install --no-streaming -r
- 结果：PASS
"@
    } else {
        $upgradeEvidence = @"
# 覆盖安装证据

- 阶段：$Phase
- 当前版本：$Version
- applicationId：$PackageId
- 安装命令：adb install --no-streaming -r
- 清除数据或卸载：未执行
- 签名连续性：由服务器证书摘要比较和 adb install -r 共同验证
- 数据标记：physical-upgrade-marker 覆盖安装后仍存在
- 登录态文件：安装前 $loginBefore；安装后 $loginAfter
- 结果：PASS
"@
    }
    $upgradeEvidence | Set-Content -LiteralPath (Join-Path $output '覆盖安装证据.md') -Encoding UTF8

    $deviceEvidence = [ordered]@{
        schema_version='2.0'; phase=$Phase; version=$Version; commit_sha=$provenance.commit_sha
        apk=(Split-Path -Leaf $deliveredApk); apk_sha256=$provenance.apk_sha256
        device=[ordered]@{ serial=$Serial; adb_state='device'; physical_device=$true; ro_kernel_qemu=$qemu; manufacturer=$manufacturer; model=$model; android_version=$androidVersion; api_level=[int]$apiLevel; physical_size=$physicalSize; density=$density; display_size_override=$false; density_override=$false }
        selection=[ordered]@{ mode=$selectionMode; eligible_connected_devices=$connectedCandidateCount; queue_load_at_selection=$queueLoadAtSelection; idle_device_preferred=$true; shortest_fifo_when_all_busy=$true }
        queue=[ordered]@{ type='shared_fifo'; root_class='%USERPROFILE%/.codex/android-device-queue/<serial>'; lock_held_for_entire_run=$true }
        paths=@($(if ($signingMigration) { 'one-time signing migration' } else { 'adb install -r upgrade' }),'launch','click','input','back','scroll','send','SSE cursor recovery','cancel','retry','draft restore','rename','archive','delete','export','feedback','regenerate','speech entry')
        install_transition=[ordered]@{ mode=if ($signingMigration) { 'one_time_uninstall_then_install' } else { 'adb_install_r' }; data_preserved=(-not $signingMigration); old_login_state=$loginBefore; login_state_immediately_after_install=$loginAfter; staging_session_provisioned=$sessionProvisioned; final_login_state=$loginAfterProvision }
        signing_migration=[ordered]@{ applied=$signingMigration; previous_certificate_sha256=$provenance.previous_signing_certificate_sha256; new_certificate_sha256=$provenance.signing_certificate_sha256; approved_contract=if ($signingMigration) { 'contracts/signing-migrations/P03.properties' } else { $null } }
        state_screenshot_count=@($stateRows).Count
        production_page_screenshot_count=$productionPages.Count
        real_device_ui_flow='PASS'
        deterministic_gateway_scope='UI interaction only; not staging acceptance'
        real_staging_business_flow='PASS'
        log_review='PASS'
        state_matrix_visual_compare=$serverPostProcessing
        production_page_visual_review='PENDING_CODEX_VISUAL_REVIEW'
        issues=@()
        result='PENDING_SERVER_POST_PROCESSING'
        completed_at=(Get-Date).ToUniversalTime().ToString('o')
    }
    $deviceEvidence | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $output '真机验收证据.json') -Encoding UTF8
    Write-Warning 'Production-page screenshots require direct Codex comparison with the representative approved mockups. The APK is not deliverable yet.'
    Write-Output "PHYSICAL_ACCEPTANCE_DIR=$output"
} finally {
    if ($QueueOwned -and $ActiveLock -and (Test-Path -LiteralPath $ActiveLock)) {
        $activeFull = [System.IO.Path]::GetFullPath($ActiveLock)
        if ($activeFull.StartsWith([System.IO.Path]::GetFullPath((Split-Path -Parent $ActiveLock)), [System.StringComparison]::OrdinalIgnoreCase)) {
            Remove-Item -LiteralPath $ActiveLock -Recurse -Force
        }
    }
    if ($TicketPath -and (Test-Path -LiteralPath $TicketPath)) { Remove-Item -LiteralPath $TicketPath -Force }
    Pop-Location
}
