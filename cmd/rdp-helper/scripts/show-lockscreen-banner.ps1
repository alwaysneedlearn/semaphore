# Lock-screen notice for the PHYSICAL display.
# By default OVERLAYS text on a copy of the current lock wallpaper (does not replace with a solid image).
# Still temporarily forces LockScreenImage policy to that copy; -Clear restores previous policy values.
# ASCII-only script (WinPS 5.1 safe). Pass Chinese via -Title / -Text.
#
#   .\show-lockscreen-banner.ps1 -Title "..." -Text "..."
#   .\show-lockscreen-banner.ps1 -Title "..." -Text "..." -LockConsole
#   .\show-lockscreen-banner.ps1 -SolidBackground          # old look: dark image, ignore wallpaper
#   .\show-lockscreen-banner.ps1 -Clear                    # remove force + restore backup

param(
  [string]$Title = 'Remote session active',
  [string]$Text = 'This PC is being accessed via Remote Desktop. Do not operate locally.',
  [string]$ImagePath = '',
  [switch]$SolidBackground,
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
$backupPath = Join-Path $dataDir 'lockscreen-policy-backup.xml'
if ([string]::IsNullOrWhiteSpace($ImagePath)) {
  $ImagePath = Join-Path $dataDir 'lock-banner.jpg'
}

function Get-RegString([string]$Key, [string]$Name) {
  if (-not (Test-Path -LiteralPath $Key)) { return $null }
  try {
    $v = (Get-ItemProperty -LiteralPath $Key -Name $Name -ErrorAction Stop).$Name
    if ([string]::IsNullOrWhiteSpace([string]$v)) { return $null }
    return [string]$v
  } catch { return $null }
}

function Get-RegDword([string]$Key, [string]$Name) {
  if (-not (Test-Path -LiteralPath $Key)) { return $null }
  try {
    return [int](Get-ItemProperty -LiteralPath $Key -Name $Name -ErrorAction Stop).$Name
  } catch { return $null }
}

function Save-PolicyBackup {
  if (-not (Test-Path -LiteralPath $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
  }
  # Only backup once per "session" of banner use (do not overwrite with our own image path).
  if (Test-Path -LiteralPath $backupPath) { return }
  $obj = New-Object PSObject -Property @{
    PolicyLockScreenImage = (Get-RegString $policyKey 'LockScreenImage')
    CspPath               = (Get-RegString $cspKey 'LockScreenImagePath')
    CspUrl                = (Get-RegString $cspKey 'LockScreenImageUrl')
    CspStatus             = (Get-RegDword $cspKey 'LockScreenImageStatus')
    SavedAt               = (Get-Date -Format o)
  }
  $obj | Export-Clixml -LiteralPath $backupPath
  Write-Output "LOCKSCREEN_BACKUP_SAVED|$backupPath"
}

function Restore-PolicyBackup {
  if (-not (Test-Path -LiteralPath $backupPath)) {
    if (Test-Path -LiteralPath $policyKey) {
      Remove-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath $cspKey) {
      Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -ErrorAction SilentlyContinue
      Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -ErrorAction SilentlyContinue
      Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -ErrorAction SilentlyContinue
    }
    Write-Output 'LOCKSCREEN_BANNER_CLEARED|no_backup'
    return
  }
  $b = Import-Clixml -LiteralPath $backupPath
  if (-not (Test-Path -LiteralPath $policyKey)) { New-Item -Path $policyKey -Force | Out-Null }
  if (-not (Test-Path -LiteralPath $cspKey)) { New-Item -Path $cspKey -Force | Out-Null }

  if ($null -ne $b.PolicyLockScreenImage -and $b.PolicyLockScreenImage -ne '') {
    New-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -PropertyType String -Value $b.PolicyLockScreenImage -Force | Out-Null
  } else {
    Remove-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -ErrorAction SilentlyContinue
  }

  if ($null -ne $b.CspPath -and $b.CspPath -ne '') {
    New-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -PropertyType String -Value $b.CspPath -Force | Out-Null
  } else {
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -ErrorAction SilentlyContinue
  }
  if ($null -ne $b.CspUrl -and $b.CspUrl -ne '') {
    New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -PropertyType String -Value $b.CspUrl -Force | Out-Null
  } else {
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -ErrorAction SilentlyContinue
  }
  if ($null -ne $b.CspStatus) {
    New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -PropertyType DWord -Value ([int]$b.CspStatus) -Force | Out-Null
  } else {
    Remove-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -ErrorAction SilentlyContinue
  }

  Remove-Item -LiteralPath $backupPath -Force -ErrorAction SilentlyContinue
  Write-Output 'LOCKSCREEN_BANNER_CLEARED|restored_backup'
}

function Find-BaseLockImage {
  $candidates = @()
  $p1 = Get-RegString $policyKey 'LockScreenImage'
  $p2 = Get-RegString $cspKey 'LockScreenImagePath'
  foreach ($p in @($p1, $p2)) {
    if ($p -and ($p -ne $ImagePath) -and (Test-Path -LiteralPath $p)) { $candidates += $p }
  }
  if (Test-Path -LiteralPath $backupPath) {
    try {
      $b = Import-Clixml -LiteralPath $backupPath
      foreach ($p in @($b.PolicyLockScreenImage, $b.CspPath)) {
        if ($p -and ($p -ne $ImagePath) -and (Test-Path -LiteralPath $p)) { $candidates += $p }
      }
    } catch { }
  }
  $web = Join-Path $env:SystemRoot 'Web\Screen'
  if (Test-Path -LiteralPath $web) {
    $candidates += @(Get-ChildItem -LiteralPath $web -Include *.jpg,*.jpeg,*.png -File -ErrorAction SilentlyContinue |
      Sort-Object Length -Descending | Select-Object -ExpandProperty FullName)
  }
  foreach ($c in $candidates) {
    if ($c -and (Test-Path -LiteralPath $c)) { return $c }
  }
  return $null
}

if ($Clear) {
  Restore-PolicyBackup
  if (Test-Path -LiteralPath $ImagePath) {
    Remove-Item -LiteralPath $ImagePath -Force -ErrorAction SilentlyContinue
  }
  Write-Output 'HINT|re-lock_console_to_refresh'
  exit 0
}

if (-not (Test-Path -LiteralPath $dataDir)) {
  New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
}

Save-PolicyBackup

Add-Type -AssemblyName System.Drawing

$width = 1920
$height = 1080
$bmp = $null
$g = $null
$basePath = $null
$mode = 'solid'

if (-not $SolidBackground) {
  $basePath = Find-BaseLockImage
  if ($basePath) {
    try {
      $src = [System.Drawing.Image]::FromFile($basePath)
      $bmp = New-Object System.Drawing.Bitmap $width, $height
      $g = [System.Drawing.Graphics]::FromImage($bmp)
      $g.InterpolationMode = 'HighQualityBicubic'
      $g.DrawImage($src, 0, 0, $width, $height)
      $src.Dispose()
      $mode = "overlay|$basePath"
      Write-Output "LOCKSCREEN_BASE|$basePath"
    } catch {
      $bmp = $null
      $g = $null
    }
  }
}

if ($null -eq $bmp) {
  $bmp = New-Object System.Drawing.Bitmap $width, $height
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.Clear([System.Drawing.Color]::FromArgb(255, 20, 20, 20))
  $mode = 'solid'
  Write-Output 'LOCKSCREEN_BASE|none_using_solid'
}

$g.SmoothingMode = 'AntiAlias'
$g.TextRenderingHint = 'ClearTypeGridFit'

# Semi-transparent top bar so original wallpaper remains visible around/below it.
$bar = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(210, 230, 126, 34))
$g.FillRectangle($bar, 0, 0, $width, 200)
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

$titleFont = New-Object System.Drawing.Font($fontFamily, 42, [System.Drawing.FontStyle]::Bold)
$bodyFont = New-Object System.Drawing.Font($fontFamily, 22, [System.Drawing.FontStyle]::Regular)
$white = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::White)
$sf = New-Object System.Drawing.StringFormat
$sf.Alignment = 'Center'
$sf.LineAlignment = 'Center'

$titleRect = New-Object System.Drawing.RectangleF(40, 24, ($width - 80), 70)
$bodyRect = New-Object System.Drawing.RectangleF(60, 100, ($width - 120), 80)
$g.DrawString($Title, $titleFont, $white, $titleRect, $sf)
$g.DrawString($Text, $bodyFont, $white, $bodyRect, $sf)

$g.Dispose()
$titleFont.Dispose()
$bodyFont.Dispose()
$white.Dispose()
$sf.Dispose()

$jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq 'image/jpeg' }
$enc = [System.Drawing.Imaging.Encoder]::Quality
$encParams = New-Object System.Drawing.Imaging.EncoderParameters(1)
$encParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter($enc, 90L)
$bmp.Save($ImagePath, $jpegCodec, $encParams)
$encParams.Dispose()
$bmp.Dispose()

if (-not (Test-Path -LiteralPath $policyKey)) { New-Item -Path $policyKey -Force | Out-Null }
New-ItemProperty -LiteralPath $policyKey -Name LockScreenImage -PropertyType String -Value $ImagePath -Force | Out-Null

if (-not (Test-Path -LiteralPath $cspKey)) { New-Item -Path $cspKey -Force | Out-Null }
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImagePath -PropertyType String -Value $ImagePath -Force | Out-Null
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageUrl -PropertyType String -Value $ImagePath -Force | Out-Null
New-ItemProperty -LiteralPath $cspKey -Name LockScreenImageStatus -PropertyType DWord -Value 1 -Force | Out-Null

Write-Output "LOCKSCREEN_BANNER_SET|image=$ImagePath|mode=$mode"
Write-Output "LOCKSCREEN_BANNER_SET|title=$Title"

if ($LockConsole) {
  $task = 'SemaphoreRdpLockConsole'
  $tr = 'rundll32.exe user32.dll,LockWorkStation'
  schtasks.exe /Delete /TN $task /F 2>$null | Out-Null
  $create = schtasks.exe /Create /TN $task /TR $tr /SC ONCE /ST 00:00 /RL HIGHEST /F 2>&1
  if ($LASTEXITCODE -eq 0) {
    schtasks.exe /Run /TN $task 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    schtasks.exe /Delete /TN $task /F 2>$null | Out-Null
    Write-Output 'LOCK_CONSOLE_ATTEMPTED'
  } else {
    Write-Output "LOCK_CONSOLE_SKIPPED|$create"
  }
  try {
    rundll32.exe user32.dll,LockWorkStation
    Write-Output 'LOCK_CURRENT_SESSION_OK'
  } catch {
    Write-Output 'LOCK_CURRENT_SESSION_FAIL'
  }
} else {
  Write-Output 'HINT|lock_console_or_pass_-LockConsole'
}

Write-Output 'DONE'
