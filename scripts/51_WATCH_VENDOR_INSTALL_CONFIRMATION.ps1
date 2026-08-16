param(
    [Parameter(Mandatory=$true)][string]$AdbPath,
    [Parameter(Mandatory=$true)][string]$Serial,
    [Parameter(Mandatory=$true)][string]$MonitorPidPath,
    [Parameter(Mandatory=$true)][string]$EvidenceDir,
    [ValidateRange(1,600)][int]$TimeoutSeconds = 180
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-AdbLenient {
    param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Arguments)
    $savedPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $output = @(& $AdbPath -s $Serial @Arguments 2>&1)
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $savedPreference
    }
    return [pscustomobject]@{
        ExitCode = $exitCode
        Text = @($output | ForEach-Object { $_.ToString() })
    }
}

if (-not (Test-Path -LiteralPath $AdbPath)) { throw 'ADB path does not exist.' }
$AdbPath = (Resolve-Path -LiteralPath $AdbPath).Path
$state = (& $AdbPath -s $Serial get-state 2>$null).Trim()
if ($state -ne 'device') { throw "Device $Serial is not in exact state device." }

New-Item -ItemType Directory -Path $EvidenceDir -Force | Out-Null
$localXml = Join-Path $EvidenceDir 'latest-install-ui.xml'
$transcript = Join-Path $EvidenceDir 'install-confirmations.log'
$readyMarker = Join-Path $EvidenceDir 'watcher-ready.txt'
"$(Get-Date -Format o) READY serial=$Serial" | Set-Content -LiteralPath $readyMarker -Encoding UTF8
$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
$lastSignature = $null
$clickCount = 0
$riskAcknowledgementClicked = $false
$monitorPid = $null

while ($null -eq $monitorPid -and (Get-Date) -lt $deadline) {
    if (Test-Path -LiteralPath $MonitorPidPath) {
        $monitorPidText = (Get-Content -Raw -LiteralPath $MonitorPidPath).Trim()
        if ($monitorPidText -notmatch '^\d+$') { throw "Invalid ADB install process identifier in $MonitorPidPath." }
        $monitorPid = [int]$monitorPidText
    } else {
        Start-Sleep -Milliseconds 50
    }
}
if ($null -eq $monitorPid) { throw 'Install watcher timed out waiting for the ADB install process identifier.' }

while ((Get-Process -Id $monitorPid -ErrorAction SilentlyContinue) -and (Get-Date) -lt $deadline) {
    $stateResult = Invoke-AdbLenient get-state
    if ($stateResult.ExitCode -ne 0 -or (($stateResult.Text -join '').Trim()) -ne 'device') {
        throw "Device $Serial stopped being available in exact state device."
    }

    $dump = Invoke-AdbLenient shell uiautomator dump /sdcard/Download/ylven-install-ui.xml
    if ($dump.ExitCode -eq 0) {
        $pull = Invoke-AdbLenient pull /sdcard/Download/ylven-install-ui.xml $localXml
        if ($pull.ExitCode -eq 0 -and (Test-Path -LiteralPath $localXml)) {
            try {
                [xml]$document = Get-Content -Raw -LiteralPath $localXml
                $nodes = @($document.SelectNodes('//node') | Where-Object {
                    $_.package -match '(?i)(securitycenter|packageinstaller|permissionmanager|installer|vivo|bbk)' -and
                    $_.package -notmatch '(?i)(launcher|systemui)'
                })
                if ($nodes.Count -gt 0) {
                    $labels = @($nodes | ForEach-Object {
                        (@($_.text, $_.GetAttribute('content-desc')) | Where-Object { $_ }) -join '/'
                    } | Where-Object { $_ } | Sort-Object -Unique)
                    $signature = $labels -join ' | '
                    if ($signature -and $signature -ne $lastSignature) {
                        "$(Get-Date -Format o) UI $signature" | Add-Content -LiteralPath $transcript -Encoding UTF8
                        $lastSignature = $signature
                    }

                    $riskAcknowledgement = $nodes | Where-Object {
                        $label = (($_.text + ' ' + $_.GetAttribute('content-desc')).Trim())
                        $_.enabled -eq 'true' -and $_.clickable -eq 'true' -and
                        $label -match '已了解应用的风险检测结果'
                    } | Select-Object -First 1

                    if (-not $riskAcknowledgementClicked -and $riskAcknowledgement -and
                        $riskAcknowledgement.bounds -match '^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$') {
                        $x = [int](([int]$Matches[1] + [int]$Matches[3]) / 2)
                        $y = [int](([int]$Matches[2] + [int]$Matches[4]) / 2)
                        $clickCount++
                        $remoteShot = "/sdcard/Download/ylven-install-confirm-$clickCount.png"
                        $localShot = Join-Path $EvidenceDir "install-confirm-$clickCount.png"
                        Invoke-AdbLenient shell screencap '-p' $remoteShot | Out-Null
                        Invoke-AdbLenient pull $remoteShot $localShot | Out-Null
                        "$(Get-Date -Format o) CLICK text=$($riskAcknowledgement.text) bounds=$($riskAcknowledgement.bounds) screenshot=$localShot" |
                            Add-Content -LiteralPath $transcript -Encoding UTF8
                        Invoke-AdbLenient shell input tap $x $y | Out-Null
                        $riskAcknowledgementClicked = $true
                        Start-Sleep -Milliseconds 900
                        continue
                    }

                    $positiveLabel = $nodes | Where-Object {
                        $label = (($_.text + ' ' + $_.GetAttribute('content-desc')).Trim())
                        $label -notmatch '取消|拒绝|禁止|不允许|退出' -and
                        $label -match '^(继续安装|允许本次安装|仍然安装|安装|允许|继续|确认|确定|仅打开一次)(\s*[（(]?\d+\s*[）)]?)?$'
                    } | Select-Object -First 1

                    # vivo renders the affirmative label in a non-clickable child while
                    # android:id/button1 is the clickable parent. Require the approved
                    # label before using that parent so unrelated installer dialogs remain untouched.
                    $positive = $null
                    if ($positiveLabel) {
                        $positive = $nodes | Where-Object {
                            $_.'resource-id' -eq 'android:id/button1' -and
                            $_.enabled -eq 'true' -and $_.clickable -eq 'true'
                        } | Select-Object -First 1
                    }
                    if (-not $positive) {
                        $positive = $positiveLabel | Where-Object {
                            $_.enabled -eq 'true' -and $_.clickable -eq 'true'
                        } | Select-Object -First 1
                    }

                    if ($positive -and $positive.bounds -match '^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$') {
                        $x = [int](([int]$Matches[1] + [int]$Matches[3]) / 2)
                        $y = [int](([int]$Matches[2] + [int]$Matches[4]) / 2)
                        $clickCount++
                        $remoteShot = "/sdcard/Download/ylven-install-confirm-$clickCount.png"
                        $localShot = Join-Path $EvidenceDir "install-confirm-$clickCount.png"
                        Invoke-AdbLenient shell screencap '-p' $remoteShot | Out-Null
                        Invoke-AdbLenient pull $remoteShot $localShot | Out-Null
                        "$(Get-Date -Format o) CLICK text=$($positive.text) bounds=$($positive.bounds) screenshot=$localShot" |
                            Add-Content -LiteralPath $transcript -Encoding UTF8
                        Invoke-AdbLenient shell input tap $x $y | Out-Null
                        $riskAcknowledgementClicked = $false
                        Start-Sleep -Milliseconds 900
                    }
                }
            } catch {
                "$(Get-Date -Format o) RETRY $($_.Exception.Message)" | Add-Content -LiteralPath $transcript -Encoding UTF8
            }
        }
    }
    Start-Sleep -Milliseconds 250
}

if (Get-Process -Id $monitorPid -ErrorAction SilentlyContinue) {
    throw "Install watcher timed out while ADB install process $monitorPid was still running."
}

"$(Get-Date -Format o) WATCHER_EXITED reason=adb_install_process_exited pid=$monitorPid clicks=$clickCount" |
    Add-Content -LiteralPath $transcript -Encoding UTF8
