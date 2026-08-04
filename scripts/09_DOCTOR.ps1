param([string]$ProjectRoot=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path,[switch]$SkipSpecKit)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$errors=[Collections.Generic.List[string]]::new(); $warnings=[Collections.Generic.List[string]]::new()
$required=@('PROJECT_CONTEXT.yaml','CURRENT_PHASE.yaml','CURRENT_WORK_PACKET.yaml','contracts\state-transition-contract.yaml','contracts\execution-environment.yaml','contracts\repository-policy.yaml','status\WORK_PACKET_HISTORY.jsonl','status\RELEASE_LEDGER.jsonl','scripts\37_PROJECT_STATE.py','scripts\38_VALIDATE_STATE_CONTINUITY.py','scripts\40_PUBLIC_REPOSITORY_SAFETY.py','scripts\42_RUN_PYTHON.ps1')
foreach($r in $required){if(-not(Test-Path(Join-Path $ProjectRoot $r))){$errors.Add("Missing $r")}}
if(-not(Get-Command git -ErrorAction SilentlyContinue)){$errors.Add('Git is required on the lightweight local client')}
if(-not(Get-Command uv -ErrorAction SilentlyContinue)){$errors.Add('uv is required; system Python is not required')}
if(-not $SkipSpecKit -and -not(Get-Command specify -ErrorAction SilentlyContinue)){$errors.Add('Spec Kit is not installed')}
foreach($tool in @('java','go','gradle','adb','docker','node')){if(-not(Get-Command $tool -ErrorAction SilentlyContinue)){$warnings.Add("$tool absent locally: expected under THIN_CONTROL_CLIENT; CI/server owns this workload")}}
if($errors.Count-eq 0){
 & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/38_VALIDATE_STATE_CONTINUITY.py'
 & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/40_PUBLIC_REPOSITORY_SAFETY.py'
 & (Join-Path $PSScriptRoot '42_RUN_PYTHON.ps1') -Script 'scripts/13_VALIDATE_PACKAGE.py' --repository-mode
}
foreach($x in $warnings){Write-Warning $x}
if($errors.Count){foreach($x in $errors){Write-Error $x}; throw "Doctor failed with $($errors.Count) errors"}
Write-Host "Doctor passed. Local heavy-tool warnings: $($warnings.Count); they are not blockers."
