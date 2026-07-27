# Show custom text on the MACHINE lock screen (physical display), not the RDP desktop.
# How: render a JPEG and force it as Windows lock-screen image (Personalization policy/CSP).
# ASCII-only script body (WinPS 5.1 / GBK safe). Pass Chinese via -Title / -Text.
#
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\show-lockscreen-banner.ps1 -Title "..." -Text "..."
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\show-lockscreen-banner.ps1 -Title "..." -Text "..." -LockConsole
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\show-lockscreen-banner.ps1 -Clear
#
# Requires Administrator. Lock screen updates after the console is locked (or re-locked).

param(
  [string]$Title = 'Remote session active',
  [string]$Text = 'This PC is being accessed via Remote Desktop. Do not operate locally.',
  [string]$ImagePath = '',
  [switch]$LockConsole,
  [switch]$Clear
)

$ErrorActionPreference = 'Stop'

function Test-IsAdmin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  $p = New-Object Security.Principal.WindowsPrincipal($id)
  return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-IsAdmin)) {
  Write-Output 'ERROR|admin_required'
  throw 'Run elevated (Administrator) PowerShell.'
}

$policyKey = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization'
$cspKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP'
$dataDir = Join-Path $env:ProgramData 'SemaphoreRdp'
if ([string]::IsNullOrWhiteSpace($ImagePath)) {
  $ImagePath = Join-Path $dataDir 'lock-banner.jpg'
}

if ($Clear) {
  if (Test-Path -LiteralPath $policyKey) {
    Remove-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -ErrorAction SilentlyContinue
  }
  if (Test-Path -LiteralPath $cspKey) {
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -ErrorAction SilentlyContinue
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -ErrorAction SilentlyContinue
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -ErrorAction SilentlyContinue
  }
  if (Test-Path -LiteralPath $ImagePath) {
    Remove-Item -LiteralPath $ImagePath -Force -ErrorAction SilentlyContinue
  }
  Write-Output 'LOCKSCREEN_BANNER_CLEARED'
  Write-Output 'HINT|re-lock_console_to_refresh'
  exit 0
}

if (-not (Test-Path -LiteralPath $dataDir)) {
  New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
}

Add-Type -AssemblyName System.Drawing

$width = 1920
$height = 1080
$bmp = New-Object System.Drawing.Bitmap $width, $height
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = 'AntiAlias'
$g.TextRenderingHint = 'ClearTypeGridFit'
$g.Clear([System.Drawing.Color]::FromArgb(255, 20, 20, 20))

# Accent bar
$bar = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(255, 230, 126, 34))
$g.FillRectangle($bar, 0, 0, $width, 160)
$bar.Dispose()

$fontFamily = 'Segoe UI'
foreach ($name in @('Microsoft YaHei UI', 'Microsoft YaHei', 'Segoe UI')) {
  try {
    $probe = New-Object System.Drawing.Font($name, 12)
    $probe.Dispose()
    $fontFamily = $name
    break
  } catch { }
}

$titleFont = New-Object System.Drawing.Font($fontFamily, 48, [System.Drawing.FontStyle]::Bold)
$bodyFont = New-Object System.Drawing.Font($fontFamily, 28, [System.Drawing.FontStyle]::Regular)
$white = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::White)
$gray = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(255, 220, 220, 220))
$sf = New-Object System.Drawing.StringFormat
$sf.Alignment = 'Center'
$sf.LineAlignment = 'Center'

$titleRect = New-Object System.Drawing.RectangleF(40, 20, ($width - 80), 120)
$bodyRect = New-Object System.Drawing.RectangleF(80, 280, ($width - 160), 400)
$g.DrawString($Title, $titleFont, $white, $titleRect, $sf)
$g.DrawString($Text, $bodyFont, $gray, $bodyRect, $sf)

$hintFont = New-Object System.Drawing.Font($fontFamily, 16, [System.Drawing.FontStyle]::Regular)
$hintRect = New-Object System.Drawing.RectangleF(40, ($height - 80), ($width - 80), 40)
$g.DrawString('Semaphore RDP lock-screen banner', $hintFont, $gray, $hintRect, $sf)

$g.Dispose()
$titleFont.Dispose()
$bodyFont.Dispose()
$hintFont.Dispose()
$white.Dispose()
$gray.Dispose()
$sf.Dispose()

$jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq 'image/jpeg' }
$enc = [System.Drawing.Imaging.Encoder]::Quality
$encParams = New-Object System.Drawing.Imaging.EncoderParameters(1)
$encParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter($enc, 90L)
$bmp.Save($ImagePath, $jpegCodec, $encParams)
$encParams.Dispose()
$bmp.Dispose()

if (-not (Test-Path -LiteralPath $policyKey)) {
  New-Item -Path $policyKey -Force | Out-Null
}
New-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -PropertyType String -Value $ImagePath -Force | Out-Null

if (-not (Test-Path -LiteralPath $cspKey)) {
  New-Item -Path $cspKey -Force | Out-Null
}
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -PropertyType String -Value $ImagePath -Force | Out-Null
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -PropertyType String -Value $ImagePath -Force | Out-Null
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -PropertyType DWord -Value 1 -Force | Out-Null

Write-Output "LOCKSCREEN_BANNER_SET|image=$ImagePath"
Write-Output "LOCKSCREEN_BANNER_SET|title=$Title"

function Get-ConsoleSessionId {
  $lines = @(qwinsta 2>$null)
  foreach ($line in $lines) {
    # console session: SESSIONNAME "console" (or "Console")
    if ($line -match '(?i)^\s*console\s+\S+\s+(\d+)\s+') {
      return [int]$Matches[1]
    }
    if ($line -match '(?i)^\s*console\s+(\d+)\s+') {
      return [int]$Matches[1]
    }
  }
  return 0
}

if ($LockConsole) {
  $sid = Get-ConsoleSessionId
  $task = 'SemaphoreRdpLockConsole'
  $tr = 'rundll32.exe user32.dll,LockWorkStation'
  # Interactive task in console session context is best-effort; may fail without logged-on console user.
  schtasks.exe /Delete /TN $task /F 2>$null | Out-Null
  $create = schtasks.exe /Create /TN $task /TR $tr /SC ONCE /ST 00:00 /RL HIGHEST /F 2>&1
  if ($LASTEXITCODE -eq 0) {
    schtasks.exe /Run /TN $task 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    schtasks.exe /Delete /TN $task /F 2>$null | Out-Null
    Write-Output "LOCK_CONSOLE_ATTEMPTED|session=$sid"
  } else {
    Write-Output "LOCK_CONSOLE_SKIPPED|reason=schtasks_failed|$create"
  }
  # Also lock current (RDP) session so operator sees the new lock image when reconnecting.
  try {
    rundll32.exe user32.dll,LockWorkStation
    Write-Output 'LOCK_CURRENT_SESSION_OK'
  } catch {
    Write-Output 'LOCK_CURRENT_SESSION_FAIL'
  }
} else {
  Write-Output 'HINT|lock_the_console_display_or_pass_-LockConsole'
  Write-Output 'HINT|if_console_already_locked_press_a_key_or_WinL_then_look_at_monitor'
}

Write-Output 'DONE'
