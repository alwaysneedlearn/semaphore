# SINEXCEL: ensure desktop shortcut has Run as administrator.
# If .lnk exists → only set admin bit (preserve TargetPath); repeated updates are safe (idempotent OR).
# If missing → create pointing at SEMAPHORE_SHORTCUT_EXE_PATH.
#
# Env:
#   SEMAPHORE_SHORTCUT_EXE_PATH     — target exe when creating
#   SEMAPHORE_SHORTCUT_PROFILE_USER — DOMAIN\user or short name (from explorer GetOwner)
#   SEMAPHORE_SHORTCUT_DESKTOP      — optional explicit desktop dir
#   SEMAPHORE_SHORTCUT_NAME         — default "BTS - 快捷方式.lnk"
#   SEMAPHORE_EXE_ARGS              — Arguments when creating new .lnk
$ErrorActionPreference = 'Stop'

function Resolve-DesktopDir {
  param([string]$ProfileUser, [string]$Override)
  $override = if ($null -eq $Override) { '' } else { $Override.Trim() }
  if ($override.Length -gt 0) { return $override }

  $profileUser = if ($null -eq $ProfileUser) { '' } else { $ProfileUser.Trim() }
  if ($profileUser.Length -eq 0) {
    throw 'empty_profile_user'
  }
  $short = ($profileUser -split '\\')[-1]
  $desktop = Join-Path $env:SystemDrive "Users\$short\Desktop"
  if (-not (Test-Path -LiteralPath $desktop)) {
    $suffix = '\' + $short + '$'
    $loaded = Get-CimInstance Win32_UserProfile -Filter "Loaded=True" -ErrorAction SilentlyContinue |
      Where-Object { $_.LocalPath -and ($_.LocalPath.EndsWith($suffix)) } |
      Select-Object -First 1
    if ($loaded -and $loaded.LocalPath) {
      $desktop = Join-Path $loaded.LocalPath 'Desktop'
    }
  }
  if (-not (Test-Path -LiteralPath $desktop)) {
    throw "desktop_not_found|user=$profileUser|tried=$desktop"
  }
  return $desktop
}

function Set-RunAsAdminFlag {
  param([string]$ShortcutPath)
  $bytes = [IO.File]::ReadAllBytes($ShortcutPath)
  $bytes[0x15] = $bytes[0x15] -bor 0x20
  [IO.File]::WriteAllBytes($ShortcutPath, $bytes)
}

$exePath = ([string]$env:SEMAPHORE_SHORTCUT_EXE_PATH).Trim()
$linkName = ([string]$env:SEMAPHORE_SHORTCUT_NAME).Trim()
if ($linkName.Length -eq 0) { $linkName = 'BTS - 快捷方式.lnk' }
if (-not $linkName.ToLowerInvariant().EndsWith('.lnk')) {
  $linkName = $linkName + '.lnk'
}

$exeArgs = ([string]$env:SEMAPHORE_EXE_ARGS)
if ($null -eq $exeArgs) { $exeArgs = '' }

try {
  $desktop = Resolve-DesktopDir -ProfileUser ([string]$env:SEMAPHORE_SHORTCUT_PROFILE_USER) -Override ([string]$env:SEMAPHORE_SHORTCUT_DESKTOP)
  $shortcutPath = Join-Path $desktop $linkName

  if (Test-Path -LiteralPath $shortcutPath) {
    Set-RunAsAdminFlag -ShortcutPath $shortcutPath
    Write-Output "SHORTCUT_OK|mode=flag_only|path=$shortcutPath|run_as_admin=1"
    exit 0
  }

  if ($exePath.Length -eq 0) {
    Write-Output "SHORTCUT_ERROR|reason=lnk_missing_and_exe_path_empty|path=$shortcutPath"
    exit 1
  }

  $workDir = Split-Path -Parent $exePath
  $shell = New-Object -ComObject WScript.Shell
  $lnk = $shell.CreateShortcut($shortcutPath)
  $lnk.TargetPath = $exePath
  $lnk.WorkingDirectory = $workDir
  $lnk.Arguments = $exeArgs
  $lnk.WindowStyle = 1
  $lnk.Description = "Launch $exePath"
  $lnk.Save()
  [System.Runtime.InteropServices.Marshal]::ReleaseComObject($lnk) | Out-Null
  [System.Runtime.InteropServices.Marshal]::ReleaseComObject($shell) | Out-Null

  Set-RunAsAdminFlag -ShortcutPath $shortcutPath
  Write-Output "SHORTCUT_OK|mode=create|path=$shortcutPath|exe=$exePath|run_as_admin=1"
  exit 0
} catch {
  Write-Output ("SHORTCUT_ERROR|reason=" + ($_.Exception.Message -replace '[\r\n]+', ' '))
  exit 1
}
