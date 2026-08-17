# NEWARE health: process has Established TCP to Kafka broker ports → working.
# Env:
#   KAFKA_PROCESS_NAME (default NWReport_DBWB) — Get-Process name without .exe
#   KAFKA_REMOTE_PORTS (default 9092,9093,9094) — comma-separated remote ports
#   KAFKA_TCP_TIMEOUT_SEC (default 12) — cap for netstat.exe
# Stdout terminal lines:
#   KAFKA_OK|Established|count=N|ports=...
#   KAFKA_FAIL|no_process
#   KAFKA_FAIL|timeout|netstat|...
#   KAFKA_FAIL|no_established|seen=...|conn_count=N
#
# Do NOT use unfiltered Get-NetTCPConnection: it walks every TCP socket via CIM
# and can hang for hours on busy or stuck Windows hosts (linear Ansible then
# blocks the whole patrol on that task).

$ErrorActionPreference = 'Continue'

$timeoutSec = 12
$rawTimeout = [string]$env:KAFKA_TCP_TIMEOUT_SEC
if ($rawTimeout -match '^\d+$') {
  $timeoutSec = [int]$rawTimeout
  if ($timeoutSec -lt 3) { $timeoutSec = 3 }
  if ($timeoutSec -gt 45) { $timeoutSec = 45 }
}

$procName = [string]$env:KAFKA_PROCESS_NAME
if ([string]::IsNullOrWhiteSpace($procName)) { $procName = 'NWReport_DBWB' }
$procName = $procName.Trim() -replace '(?i)\.exe$', ''

$portsRaw = [string]$env:KAFKA_REMOTE_PORTS
if ([string]::IsNullOrWhiteSpace($portsRaw)) { $portsRaw = '9092,9093,9094' }
$ports = @(
  $portsRaw.Split(',') |
    ForEach-Object { $_.Trim() } |
    Where-Object { $_ -match '^\d+$' } |
    ForEach-Object { [int]$_ }
)
if ($ports.Count -eq 0) { $ports = @(9092, 9093, 9094) }

$pids = @(Get-Process -Name $procName -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Id)
if ($pids.Count -eq 0) {
  Write-Output "KAFKA_FAIL|no_process|name=$procName"
  exit 0
}

function Get-NetstatAnoText {
  param([int]$WaitMs)
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = "$env:SystemRoot\System32\netstat.exe"
  $psi.Arguments = '-ano'
  $psi.UseShellExecute = $false
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.CreateNoWindow = $true
  $p = New-Object System.Diagnostics.Process
  $p.StartInfo = $psi
  try {
    [void]$p.Start()
    $outTask = $p.StandardOutput.ReadToEndAsync()
    $errTask = $p.StandardError.ReadToEndAsync()
    if (-not $p.WaitForExit($WaitMs)) {
      try { $p.Kill() } catch { }
      return $null
    }
    [void][System.Threading.Tasks.Task]::WaitAll(@($outTask, $errTask), 3000)
    return [string]$outTask.Result
  } catch {
    return $null
  } finally {
    try { $p.Dispose() } catch { }
  }
}

$netstatOut = Get-NetstatAnoText -WaitMs ($timeoutSec * 1000)
if ($null -eq $netstatOut) {
  Write-Output ("KAFKA_FAIL|timeout|netstat|sec=" + $timeoutSec + "|pids=" + ($pids -join ',') + "|name=" + $procName)
  exit 0
}

$matchedPorts = New-Object System.Collections.Generic.List[int]
$seenStates = New-Object 'System.Collections.Generic.HashSet[string]'
$connCount = 0
foreach ($line in ($netstatOut -split "`r?`n")) {
  $t = $line.Trim()
  if ($t -notmatch '^(TCP|TCPv6)\s+') { continue }
  $parts = $t -split '\s+'
  if ($parts.Length -lt 5) { continue }
  $remote = $parts[2]
  $state = $parts[3]
  $opid = 0
  try { $opid = [int]$parts[$parts.Length - 1] } catch { continue }
  if ($pids -notcontains $opid) { continue }
  $rport = 0
  if ($remote -match ':(\d+)$') {
    $rport = [int]$Matches[1]
  } else {
    continue
  }
  if ($ports -notcontains $rport) { continue }
  $connCount++
  [void]$seenStates.Add([string]$state)
  if ($state -eq 'ESTABLISHED' -or $state -eq '已建立') {
    $matchedPorts.Add($rport)
  }
}

if ($matchedPorts.Count -gt 0) {
  $portList = ($matchedPorts | Select-Object -Unique) -join ','
  Write-Output ("KAFKA_OK|Established|count=" + $matchedPorts.Count + "|ports=" + $portList + "|pids=" + ($pids -join ',') + "|name=" + $procName)
  exit 0
}

$seen = '(none)'
if ($seenStates.Count -gt 0) {
  $seen = (@($seenStates) | Sort-Object) -join ','
}
Write-Output ("KAFKA_FAIL|no_established|seen=" + $seen + "|conn_count=" + $connCount + "|pids=" + ($pids -join ',') + "|name=" + $procName + "|want_ports=" + ($ports -join ','))
exit 0
