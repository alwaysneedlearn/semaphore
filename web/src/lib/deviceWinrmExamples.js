const DEVICE_WINRM_TOP5_CMD = [
  'Get-CimInstance Win32_PerfFormattedData_PerfProc_Process | Out-Null',
  'Start-Sleep -Seconds 1',
  "Write-Output '=== CPU Top 5 ==='",
  'Get-CimInstance Win32_PerfFormattedData_PerfProc_Process |',
  "  Where-Object { $_.Name -notin @('_Total', 'Idle') } |",
  '  Sort-Object PercentProcessorTime -Descending |',
  '  Select-Object -First 5 Name,',
  "    @{N='PID';E={$_.IDProcess}},",
  "    @{N='CPU_%';E={$_.PercentProcessorTime}},",
  "    @{N='Mem_MB';E={[math]::Round($_.WorkingSetPrivate/1MB,1)}} |",
  '  Format-Table -AutoSize | Out-String',
  "Write-Output '=== Memory Top 5 ==='",
  'Get-Process |',
  '  Sort-Object WorkingSet64 -Descending |',
  '  Select-Object -First 5 Name, Id,',
  "    @{N='CPU_s';E={if ($null -eq $_.CPU) { 0 } else { [math]::Round($_.CPU, 1) }}},",
  "    @{N='Mem_MB';E={[math]::Round($_.WorkingSet64/1MB,1)}} |",
  '  Format-Table -AutoSize | Out-String',
].join('\n');

const DEVICE_WINRM_EXAMPLE_GROUPS = [
  {
    key: 'top5',
    labelKey: 'deviceWinrmExamplesTop5',
    commands: [DEVICE_WINRM_TOP5_CMD],
  },
  {
    key: 'process',
    labelKey: 'deviceWinrmExamplesProcess',
    commands: [
      DEVICE_WINRM_TOP5_CMD,
      "Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 Name, Id, @{N='CPU_s';E={if ($null -eq $_.CPU) { 0 } else { [math]::Round($_.CPU, 1) }}}, @{N='Mem_MB';E={[math]::Round($_.WorkingSet64/1MB,1)}}",
      "Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 5 Name, Id, @{N='Mem_MB';E={[math]::Round($_.WorkingSet64/1MB,1)}}",
      "Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 15 Name, Id, @{N='WS_MB';E={[math]::Round($_.WorkingSet64/1MB,1)}}",
      "Get-Process -Name 'sinexcel_agent' -ErrorAction SilentlyContinue | Format-List *",
    ],
  },
  {
    key: 'network',
    labelKey: 'deviceWinrmExamplesNetwork',
    commands: [
      'Get-NetTCPConnection -State Listen | Sort-Object LocalPort | Select-Object LocalAddress, LocalPort, OwningProcess',
      "netstat -ano | Select-String ':3389'",
      'netstat -ano | findstr LISTENING',
    ],
  },
  {
    key: 'filesystem',
    labelKey: 'deviceWinrmExamplesFilesystem',
    commands: [
      "Get-ChildItem 'C:\\Program Files' -Directory | Select-Object Name, LastWriteTime",
      "Test-Path 'D:\\Program Files\\NEWARE'",
    ],
  },
  {
    key: 'service',
    labelKey: 'deviceWinrmExamplesService',
    commands: [
      "Get-Service | Where-Object { $_.Status -eq 'Running' } | Select-Object Name, DisplayName, Status",
    ],
  },
  {
    key: 'disk',
    labelKey: 'deviceWinrmExamplesDisk',
    commands: [
      "Get-PSDrive -PSProvider FileSystem | Select-Object Name, @{N='Used_GB';E={[math]::Round($_.Used/1GB,2)}}, @{N='Free_GB';E={[math]::Round($_.Free/1GB,2)}}",
    ],
  },
];

export default DEVICE_WINRM_EXAMPLE_GROUPS;
