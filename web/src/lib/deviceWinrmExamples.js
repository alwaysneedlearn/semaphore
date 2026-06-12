const DEVICE_WINRM_EXAMPLE_GROUPS = [
  {
    key: 'process',
    labelKey: 'deviceWinrmExamplesProcess',
    commands: [
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
