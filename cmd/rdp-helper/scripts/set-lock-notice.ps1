# Set custom lock/logon notice (LegalNotice). ASCII-only source for Windows PowerShell 5.1.
# Pass Chinese via -Title / -Text.
#
#   .\set-lock-notice.ps1 -Title "..." -Text "..."
#   .\set-lock-notice.ps1 -Clear

param(
  [string]$Title = '',
  [string]$Text = '',
  [switch]$Clear
)

$ErrorActionPreference = 'Stop'
$key = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System'

if ($Clear) {
  if (Test-Path -LiteralPath $key) {
    Remove-ItemProperty -LiteralPath $key -Name LegalNoticeCaption -ErrorAction SilentlyContinue
    Remove-ItemProperty -LiteralPath $key -Name LegalNoticeText -ErrorAction SilentlyContinue
  }
  Write-Output 'LOCK_NOTICE_CLEARED'
  exit 0
}

if ([string]::IsNullOrWhiteSpace($Title)) {
  $Title = [string]$env:RDP_LOCK_NOTICE_TITLE
}
if ([string]::IsNullOrWhiteSpace($Text)) {
  $Text = [string]$env:RDP_LOCK_NOTICE_TEXT
}
if ([string]::IsNullOrWhiteSpace($Title)) { $Title = 'Remote assistance' }
if ([string]::IsNullOrWhiteSpace($Text)) {
  $Text = 'This PC may be accessed via Remote Desktop. Confirm before local use.'
}

if (-not (Test-Path -LiteralPath $key)) {
  New-Item -Path $key -Force | Out-Null
}
New-ItemProperty -LiteralPath $key -Name LegalNoticeCaption -PropertyType String -Value $Title -Force | Out-Null
New-ItemProperty -LiteralPath $key -Name LegalNoticeText -PropertyType String -Value $Text -Force | Out-Null

Write-Output "LOCK_NOTICE_SET|title=$Title"
Write-Output "LOCK_NOTICE_SET|text=$Text"
Write-Output 'HINT|takes_effect_on_next_lock_or_logon'
