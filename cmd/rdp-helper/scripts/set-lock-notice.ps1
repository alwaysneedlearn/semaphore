# Set custom lock/logon notice on a Windows host (LegalNotice).
# Shows at interactive logon AND when unlocking (console or RDP session lock screen).
# This is NOT driven by mstsc — deploy on the *remote* host (GPO / Intune / WinRM once).
#
# Usage (elevated PowerShell on the remote):
#   .\set-lock-notice.ps1 -Title "远程协助中" -Text "本机正在被远程桌面连接，请勿本地操作。"
#   .\set-lock-notice.ps1 -Clear
#
# Env fallbacks: RDP_LOCK_NOTICE_TITLE, RDP_LOCK_NOTICE_TEXT

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
if ([string]::IsNullOrWhiteSpace($Title)) { $Title = '远程协助中' }
if ([string]::IsNullOrWhiteSpace($Text)) {
  $Text = '本机正在被远程桌面连接。请勿在本地操作，等待远程结束后再使用。'
}

if (-not (Test-Path -LiteralPath $key)) {
  New-Item -Path $key -Force | Out-Null
}
New-ItemProperty -LiteralPath $key -Name LegalNoticeCaption -PropertyType String -Value $Title -Force | Out-Null
New-ItemProperty -LiteralPath $key -Name LegalNoticeText -PropertyType String -Value $Text -Force | Out-Null

Write-Output "LOCK_NOTICE_SET|title=$Title"
Write-Output "LOCK_NOTICE_SET|text=$Text"
Write-Output 'HINT|takes_effect_on_next_lock_or_logon'
