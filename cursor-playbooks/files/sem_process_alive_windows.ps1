# Shared helpers for Semaphore Windows playbooks (PS 5.1+).
# Env: PROCESS_ALIVE_MIN_HANDLES (default 1), PROCESS_ALIVE_MIN_WS_KB (default 512)

function Get-SemaphoreProcessAliveThresholds {
  $minHandles = 1
  if (-not [string]::IsNullOrWhiteSpace($env:PROCESS_ALIVE_MIN_HANDLES)) {
    $minHandles = [int]$env:PROCESS_ALIVE_MIN_HANDLES
  }
  $minWsKb = 512
  if (-not [string]::IsNullOrWhiteSpace($env:PROCESS_ALIVE_MIN_WS_KB)) {
    $minWsKb = [int]$env:PROCESS_ALIVE_MIN_WS_KB
  }
  $minWsBytes = [long]$minWsKb * 1024L
  return @{ MinHandles = $minHandles; MinWsBytes = $minWsBytes; MinWsKb = $minWsKb }
}

function Get-SemaphoreAliveProcess {
  param(
    [Parameter(Mandatory = $true)][string]$ProcessName,
    [int]$MinHandles = 1,
    [long]$MinWsBytes = 524288
  )
  Get-Process -Name $ProcessName -ErrorAction SilentlyContinue |
    Where-Object { $_.Handles -gt $MinHandles -and $_.WorkingSet64 -ge $MinWsBytes } |
    Select-Object -First 1
}

function Get-SemaphoreStaleProcess {
  param(
    [Parameter(Mandatory = $true)][string]$ProcessName,
    [int]$MinHandles = 1,
    [long]$MinWsBytes = 524288
  )
  $alive = Get-SemaphoreAliveProcess -ProcessName $ProcessName -MinHandles $MinHandles -MinWsBytes $MinWsBytes
  if ($alive) { return $null }
  Get-Process -Name $ProcessName -ErrorAction SilentlyContinue | Select-Object -First 1
}
