# Session-aware LegalNotice. ASCII-only source for Windows PowerShell 5.1.
# Pass Chinese via -Title / -Text.
#
#   .\watch-rdp-session-notice.ps1 -Title "..." -Text "..." -Once
#   .\watch-rdp-session-notice.ps1 -Title "..." -Text "..." -IntervalSeconds 30

param(
  [string]$Title = 'Remote session active',
  [string]$Text = 'This PC is being accessed via Remote Desktop. Do not operate locally.',
  [int]$IntervalSeconds = 30,
  [switch]$Once
)

$ErrorActionPreference = 'Continue'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$setNotice = Join-Path $scriptDir 'set-lock-notice.ps1'

function Test-ActiveRdpSession {
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
Write-Output "WATCH_RDP_NOTICE interval=${IntervalSeconds}s"
while ($true) {
  Sync-Notice
  Start-Sleep -Seconds $IntervalSeconds
}
