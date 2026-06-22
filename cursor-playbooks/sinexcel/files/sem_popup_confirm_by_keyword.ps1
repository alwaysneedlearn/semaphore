# Confirm a desktop popup by title and/or child control text (content keyword).
param(
  [string]$PopupKeywordArg = '',
  [int]$PopupWaitSecondsArg = -1,
  [string]$LogFileArg = '',
  [string]$ProcessNameArg = '',
  [string]$MatchModeArg = ''
)

function Write-PopupLine {
  param([string]$Line)
  Write-Output $Line
  if (-not [string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
    Add-Content -LiteralPath $script:PopupLogPath -Value $Line -Encoding UTF8 -ErrorAction SilentlyContinue
  }
}

$script:PopupLogPath = $LogFileArg
if ([string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
  $script:PopupLogPath = [string]$env:POPUP_LOG_FILE
}
if (-not [string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
  $logDir = Split-Path -Parent $script:PopupLogPath
  if ($logDir -and -not (Test-Path -LiteralPath $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
  }
  Set-Content -LiteralPath $script:PopupLogPath -Value '' -Encoding UTF8 -ErrorAction SilentlyContinue
}

$waitSec = $PopupWaitSecondsArg
if ($waitSec -lt 0) {
  $waitSec = 2
  if (-not [string]::IsNullOrWhiteSpace($env:POPUP_WAIT_SECONDS)) {
    [int]::TryParse($env:POPUP_WAIT_SECONDS, [ref]$waitSec) | Out-Null
  }
}
if ($waitSec -lt 0) { $waitSec = 0 }

$keyword = $PopupKeywordArg
if ([string]::IsNullOrWhiteSpace($keyword)) {
  $keyword = [string]$env:POPUP_KEYWORD
}
if ([string]::IsNullOrWhiteSpace($keyword)) {
  Write-PopupLine 'POPUP_SKIP|reason=empty_keyword'
  exit 0
}

$procName = $ProcessNameArg
if ([string]::IsNullOrWhiteSpace($procName)) {
  $procName = [string]$env:POPUP_PROCESS_NAME
}

$matchMode = $MatchModeArg
if ([string]::IsNullOrWhiteSpace($matchMode)) {
  $matchMode = [string]$env:POPUP_MATCH_MODE
}
if ([string]::IsNullOrWhiteSpace($matchMode)) {
  $matchMode = 'title_or_content'
}

Start-Sleep -Seconds $waitSec

$helperPath = Join-Path $PSScriptRoot 'sem_win32_popup_helper.ps1'
if (-not (Test-Path -LiteralPath $helperPath)) {
  $helperPath = 'C:\Windows\Temp\sem_win32_popup_helper.ps1'
}
if (-not (Test-Path -LiteralPath $helperPath)) {
  Write-PopupLine "POPUP_HELPER_MISSING|path=$helperPath"
  exit 0
}
. $helperPath

Write-PopupLine "POPUP_SCAN|keyword=$keyword|match_mode=$matchMode|process=$procName"
$matched = Invoke-SemaphorePopupConfirm -Keyword $keyword -ProcessName $procName -MatchMode $matchMode
if ($matched.Count -gt 0) {
  foreach ($t in $matched) {
    Write-PopupLine "POPUP_CONFIRMED|keyword=$keyword|match=$t"
  }
} else {
  Write-PopupLine "POPUP_NOT_FOUND|keyword=$keyword|match_mode=$matchMode|hint=no_title_popup_try_content_keyword_or_check_process"
}

exit 0
