# Session-aware lock notice: while an active RDP (rdp-tcp) session exists, set LegalNotice;
# when no RDP session remains, clear it. Run elevated as a scheduled task (e.g. every 1 min)
# or as a long-running loop on the *remote* Windows host.
#
#   .\watch-rdp-session-notice.ps1 -Title "正在被远程" -Text "操作员远程桌面连接中…" -Once
#   .\watch-rdp-session-notice.ps1 -Title "正在被远程" -Text "…" -IntervalSeconds 30

param(
  [string]$Title = '正在被远程',
  [string]$Text = '本机正在被远程桌面连接，请勿本地操作。',
  [int]$IntervalSeconds = 30,
  [switch]$Once
)

$ErrorActionPreference = 'Continue'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$setNotice = Join-Path $scriptDir 'set-lock-notice.ps1'

function Test-ActiveRdpSession {
  # qwinsta: look for SESSIONNAME like rdp-tcp#N with State Active/Conn
  $lines = @(qwinsta 2>$null)
  foreach ($line in $lines) {
    if ($line -match '(?i)rdp-tcp#\S+\s+\S+\s+\d+\s+(Active|Conn)') {
      return $true
    }
    if ($line -match '(?i)^\s*rdp-tcp#\S+.*\sActive\b') {
      return $true
    }
  }
  return $false
}

function Sync-Notice {
  $active = Test-ActiveRdpSession
  if ($active) {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $setNotice -Title $Title -Text $Text | Out-Null
    Write-Output "RDP_SESSION=active NOTICE=set $(Get-Date -Format o)"
  } else {
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $setNotice -Clear | Out-Null
    Write-Output "RDP_SESSION=none NOTICE=cleared $(Get-Date -Format o)"
  }
}

if ($Once) {
  Sync-Notice
  exit 0
}

if ($IntervalSeconds -lt 5) { $IntervalSeconds = 5 }
Write-Output "WATCH_RDP_NOTICE interval=${IntervalSeconds}s title=$Title"
while ($true) {
  Sync-Notice
  Start-Sleep -Seconds $IntervalSeconds
}
