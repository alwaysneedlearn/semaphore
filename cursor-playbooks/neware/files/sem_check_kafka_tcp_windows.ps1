# NEWARE health: process has Established TCP to Kafka broker ports → working.
# Env:
#   KAFKA_PROCESS_NAME (default NWReport_DBWB) — Get-Process name without .exe
#   KAFKA_REMOTE_PORTS (default 9092,9093,9094) — comma-separated remote ports
# Stdout terminal lines:
#   KAFKA_OK|Established|count=N|ports=...
#   KAFKA_FAIL|no_process
#   KAFKA_FAIL|no_established|seen=...|conn_count=N

$ErrorActionPreference = 'Continue'

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

$conns = @(Get-NetTCPConnection -ErrorAction SilentlyContinue | Where-Object {
  ($pids -contains $_.OwningProcess) -and ($ports -contains [int]$_.RemotePort)
})

$established = @($conns | Where-Object { [string]$_.State -eq 'Established' })
if ($established.Count -gt 0) {
  $portList = @($established | Select-Object -ExpandProperty RemotePort -Unique) -join ','
  Write-Output "KAFKA_OK|Established|count=$($established.Count)|ports=$portList|pids=$($pids -join ',')|name=$procName"
  exit 0
}

$seen = @($conns | ForEach-Object { [string]$_.State } | Select-Object -Unique) -join ','
if ([string]::IsNullOrWhiteSpace($seen)) { $seen = '(none)' }
Write-Output "KAFKA_FAIL|no_established|seen=$seen|conn_count=$($conns.Count)|pids=$($pids -join ',')|name=$procName|want_ports=$($ports -join ',')"
exit 0
