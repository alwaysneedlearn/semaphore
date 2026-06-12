# Verify process is not effectively running (zombie PIDs with low Handles/WS count as stopped).
# Env: VERIFY_PROCESS_NAME or STOP_VERIFY_PROCESS_NAME or PROCESS_NAME
#      PROCESS_ALIVE_MIN_HANDLES (default 1), PROCESS_ALIVE_MIN_WS_KB (default 512)

$ErrorActionPreference = 'Continue'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$aliveHelper = Join-Path $scriptDir 'sem_process_alive_windows.ps1'
if (-not (Test-Path -LiteralPath $aliveHelper)) {
  $aliveHelper = 'C:\Windows\Temp\sem_process_alive_windows.ps1'
}
if (-not (Test-Path -LiteralPath $aliveHelper)) {
  Write-Output 'VERIFY_HELPER_MISSING|path=sem_process_alive_windows.ps1'
  exit 1
}
. $aliveHelper

$name = [string]$env:VERIFY_PROCESS_NAME
if ([string]::IsNullOrWhiteSpace($name)) { $name = [string]$env:STOP_VERIFY_PROCESS_NAME }
if ([string]::IsNullOrWhiteSpace($name)) { $name = [string]$env:PROCESS_NAME }
$name = $name -replace '(?i)\.exe$', ''
if ([string]::IsNullOrWhiteSpace($name)) { $name = 'LHBTS' }

$t = Get-SemaphoreProcessAliveThresholds
Start-Sleep -Seconds 1

$alive = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
if ($alive) {
  Write-Output "STILL_RUNNING|PID:$($alive.Id)|Handles:$($alive.Handles)|WS_KB:$([math]::Round($alive.WorkingSet64/1KB,0))"
  exit 0
}

$stale = Get-SemaphoreStaleProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
if ($stale) {
  Write-Output "NOT_RUNNING|STALE_PID:$($stale.Id)|Handles:$($stale.Handles)|WS_KB:$([math]::Round($stale.WorkingSet64/1KB,0))"
  exit 0
}

Write-Output 'NOT_RUNNING'
