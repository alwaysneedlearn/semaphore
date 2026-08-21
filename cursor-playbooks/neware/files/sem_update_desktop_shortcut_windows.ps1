# Create/overwrite a desktop .lnk pointing at EXE, with Run as administrator.
# Env:
#   SEMAPHORE_SHORTCUT_EXE_PATH  (required) — target exe
#   SEMAPHORE_SHORTCUT_DESKTOP   (optional) — desktop dir; default Public Desktop
#   SEMAPHORE_SHORTCUT_NAME      (optional) — .lnk file name; default <exeBase>.lnk
#   SEMAPHORE_EXE_ARGS           (optional) — shortcut Arguments
$ErrorActionPreference = 'Stop'

$exePath = ([string]$env:SEMAPHORE_SHORTCUT_EXE_PATH).Trim()
if ($exePath.Length -eq 0) {
  Write-Output 'SHORTCUT_ERROR|reason=exe_path_empty'
  exit 1
}

$desktop = ([string]$env:SEMAPHORE_SHORTCUT_DESKTOP).Trim()
if ($desktop.Length -eq 0) {
  $desktop = [Environment]::GetFolderPath('CommonDesktopDirectory')
  if ([string]::IsNullOrWhiteSpace($desktop)) {
    $desktop = Join-Path $env:PUBLIC 'Desktop'
  }
}

$linkName = ([string]$env:SEMAPHORE_SHORTCUT_NAME).Trim()
if ($linkName.Length -eq 0) {
  $linkName = [IO.Path]::GetFileNameWithoutExtension($exePath) + '.lnk'
}
if (-not $linkName.ToLowerInvariant().EndsWith('.lnk')) {
  $linkName = $linkName + '.lnk'
}

$exeArgs = ([string]$env:SEMAPHORE_EXE_ARGS)
if ($null -eq $exeArgs) { $exeArgs = '' }

try {
  if (-not (Test-Path -LiteralPath $desktop)) {
    New-Item -ItemType Directory -Path $desktop -Force | Out-Null
  }

  $shortcutPath = Join-Path $desktop $linkName
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

  # Shell Link Header offset 0x15 bit 0x20 = Run as administrator
  $bytes = [IO.File]::ReadAllBytes($shortcutPath)
  $bytes[0x15] = $bytes[0x15] -bor 0x20
  [IO.File]::WriteAllBytes($shortcutPath, $bytes)

  Write-Output "SHORTCUT_OK|path=$shortcutPath|exe=$exePath|run_as_admin=1"
  exit 0
} catch {
  Write-Output ("SHORTCUT_ERROR|reason=" + ($_.Exception.Message -replace '[\r\n]+', ' '))
  exit 1
}
