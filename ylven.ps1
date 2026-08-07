param(
  [Parameter(Position=0)]
  [ValidateSet('bootstrap','doctor','resume','state','start-packet','close-packet','defer-packet','block','release','close-release','inventory','py')]
  [string]$Command='resume',
  [string]$Phase,
  [string]$Packet,
  [string]$Reason,
  [string]$SshTarget,
  [string]$Serial,
  [string]$Script,
  [Parameter(ValueFromRemainingArguments=$true)][string[]]$RemainingArguments,
  [switch]$SkipInitialPush,
  [switch]$NoPush,
  [switch]$LegacyException
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$Root=(Resolve-Path $PSScriptRoot).Path
$PythonRunner=Join-Path $Root 'scripts\42_RUN_PYTHON.ps1'

function Invoke-RepositoryPython {
  param([Parameter(Mandatory=$true)][string]$RepositoryScript,[string[]]$Arguments=@())
  & $PythonRunner -Script $RepositoryScript @Arguments
}

switch($Command){
  'bootstrap' {
    & (Join-Path $Root 'scripts\00_BOOTSTRAP_REPOSITORY.ps1') -ProjectRoot $Root -SkipInitialPush:$SkipInitialPush
  }
  'doctor' {
    & (Join-Path $Root 'scripts\09_DOCTOR.ps1') -ProjectRoot $Root
  }
  'resume' {
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments @('resume')
  }
  'state' {
    Invoke-RepositoryPython -RepositoryScript 'scripts/38_VALIDATE_STATE_CONTINUITY.py'
  }
  'start-packet' {
    $args=@('start-packet')
    if($Packet){$args+=@('--packet',$Packet)}
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments $args
  }
  'close-packet' {
    $args=@('close-packet')
    if($Packet){$args+=@('--packet',$Packet)}
    if($Reason){$args+=@('--note',$Reason)}
    if($NoPush){$args+='--no-push'}
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments $args
  }
  'defer-packet' {
    if(-not $Reason){throw '-Reason is required'}
    $args=@('defer-packet','--reason',$Reason)
    if($Packet){$args+=@('--packet',$Packet)}
    if($NoPush){$args+='--no-push'}
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments $args
  }
  'block' {
    if(-not $Reason){throw '-Reason is required'}
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments @('block','--reason',$Reason)
  }
  'release' {
    if(-not $Phase){throw '-Phase is required'}
    $releaseArgs=@{Phase=$Phase}
    if($SshTarget){$releaseArgs.SshTarget=$SshTarget}
    if($Serial){$releaseArgs.Serial=$Serial}
    & (Join-Path $Root 'scripts\05_RELEASE_PHASE.ps1') @releaseArgs
  }
  'close-release' {
    $args=@('close-release')
    if($Phase){$args+=@('--phase',$Phase)}
    if($NoPush){$args+='--no-push'}
    if($LegacyException){$args+='--legacy-exception'}
    Invoke-RepositoryPython -RepositoryScript 'scripts/37_PROJECT_STATE.py' -Arguments $args
  }
  'inventory' {
    & (Join-Path $Root 'scripts\39_REMOTE_SERVER_INVENTORY.ps1') -SshTarget $SshTarget
  }
  'py' {
    if(-not $Script){throw '-Script is required for the py command'}
    Invoke-RepositoryPython -RepositoryScript $Script -Arguments $RemainingArguments
  }
}
