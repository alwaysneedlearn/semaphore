$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'sem_process_alive_windows.ps1')

$name = [string]$env:EXE_NAME
if ([string]::IsNullOrWhiteSpace($name)) { $name = 'uu' }

$t = Get-SemaphoreProcessAliveThresholds
$proc = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes

if (-not $proc) {
  $stale = Get-SemaphoreStaleProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
  if ($stale) {
    Write-Output "NOT_RUNNING|STALE_PID:$($stale.Id)|Handles:$($stale.Handles)"
  } else {
    Write-Output 'NOT_RUNNING'
  }
  exit 0
}

try {
  Stop-Process -Id $proc.Id -Force -ErrorAction Stop
  Start-Sleep -Seconds 2
  $verify = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
  if (-not $verify) {
    Write-Output 'STOPPED'
    exit 0
  }
  taskkill /F /T /PID $proc.Id 2>$null
  Start-Sleep -Seconds 2
  $verify2 = Get-SemaphoreAliveProcess -ProcessName $name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
  if (-not $verify2) {
    Write-Output 'STOPPED_BY_TASKKILL'
    exit 0
  }
  Write-Output 'STOP_FAILED'
} catch {
  Write-Output "STOP_ERROR:$($_.Exception.Message)"
}
