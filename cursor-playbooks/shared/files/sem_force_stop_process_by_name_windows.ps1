# Force Stop-Process then verify with alive thresholds (same as NEWARE stop verify).
# Env: VERIFY_PROCESS_NAME / STOP_VERIFY_PROCESS_NAME / PROCESS_NAME
#      PROCESS_ALIVE_MIN_HANDLES, PROCESS_ALIVE_MIN_WS_KB

$ErrorActionPreference = 'Continue'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. (Join-Path $scriptDir 'sem_process_alive_windows.ps1')

$name = [string]$env:VERIFY_PROCESS_NAME
if ([string]::IsNullOrWhiteSpace($name)) { $name = [string]$env:STOP_VERIFY_PROCESS_NAME }
if ([string]::IsNullOrWhiteSpace($name)) { $name = [string]$env:PROCESS_NAME }
$name = $name -replace '(?i)\.exe$', ''
if ([string]::IsNullOrWhiteSpace($name)) { $name = 'LHBTS' }

$t = Get-SemaphoreProcessAliveThresholds

$before = @(Get-Process -Name $name -ErrorAction SilentlyContinue)
if ($before.Count -eq 0) {
  Write-Output 'NOT_RUNNING|reason=force_fallback'
  exit 0
}

$before | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$alive = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
if ($alive) {
  Write-Output "STILL_RUNNING|reason=force_fallback_failed|PID:$($alive.Id)|Handles:$($alive.Handles)|WS_KB:$([math]::Round($alive.WorkingSet64/1KB,0))"
  exit 1
}

$stale = Get-SemaphoreStaleProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
if ($stale) {
  Write-Output "FORCE_STOPPED|reason=winrm_fallback|STALE_PID:$($stale.Id)"
  exit 0
}

Write-Output 'FORCE_STOPPED|reason=winrm_fallback'
