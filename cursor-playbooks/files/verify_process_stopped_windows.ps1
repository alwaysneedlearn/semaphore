$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'process_alive_windows.ps1')

$name = [string]$env:EXE_NAME
if ([string]::IsNullOrWhiteSpace($name)) { $name = 'uu' }

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
