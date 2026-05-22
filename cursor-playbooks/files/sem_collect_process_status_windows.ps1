$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'sem_process_alive_windows.ps1')

$name = [string]$env:EXE_NAME
if ([string]::IsNullOrWhiteSpace($name)) { $name = 'uu' }

$t = Get-SemaphoreProcessAliveThresholds
$alive = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes

if ($alive) {
  $uptime = ''
  try {
    $uptime = ([datetime]::Now - $alive.StartTime).ToString('d\d\ h\h\ m\m')
  } catch {
    $uptime = 'n/a'
  }
  $memMb = [math]::Round($alive.WorkingSet64 / 1MB, 2)
  Write-Output "RUNNING|PID:$($alive.Id)|Memory:${memMb}MB|Handles:$($alive.Handles)|Started:$($alive.StartTime)|Uptime:$uptime"
  exit 0
}

$stale = Get-SemaphoreStaleProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
if ($stale) {
  $wsKb = [math]::Round($stale.WorkingSet64 / 1KB, 0)
  Write-Output "NOT_RUNNING|STALE_PID:$($stale.Id)|Handles:$($stale.Handles)|WS_KB:$wsKb"
  exit 0
}

Write-Output 'NOT_RUNNING'
