# Resolve install path (zip/APP_DIR) vs runtime path (running EXE or legacy folder e.g. 1).
# Env: SEMAPHORE_EXE_DIR, SEMAPHORE_EXE_NAME, SEMAPHORE_APP_DIR_INSTALL, SEMAPHORE_APP_DIR_TRY,
#      SEMAPHORE_PROCESS_NAME (optional)
$root = [string]$env:SEMAPHORE_EXE_DIR
$exeName = [string]$env:SEMAPHORE_EXE_NAME
$installDir = [string]$env:SEMAPHORE_APP_DIR_INSTALL
$tryRaw = [string]$env:SEMAPHORE_APP_DIR_TRY
$procName = [string]$env:SEMAPHORE_PROCESS_NAME
if ($null -eq $root) { $root = '' }
if ($null -eq $exeName) { $exeName = '' }
$root = $root.Trim().TrimEnd('\')
$exeName = $exeName.Trim()
$installDir = $installDir.Trim().Trim('\')

function Join-ExePath([string]$base, [string]$subDir) {
  if ($subDir -eq '.' -or [string]::IsNullOrWhiteSpace($subDir)) {
    return (Join-Path -Path $base -ChildPath $exeName)
  }
  return (Join-Path -Path (Join-Path -Path $base -ChildPath $subDir.Trim().Trim('\')) -ChildPath $exeName)
}

function Test-ExePath([string]$path) {
  return ($path.Length -gt 0) -and (Test-Path -LiteralPath $path)
}

if ($root.Length -eq 0 -or $exeName.Length -eq 0) {
  Write-Output 'INSTALL_APP_DIR='
  Write-Output 'INSTALL_EXE_PATH='
  Write-Output 'RUNTIME_APP_DIR='
  Write-Output 'RUNTIME_EXE_PATH='
  Write-Output 'APP_DIR_SOURCE=missing_env'
  exit 0
}

# Install target (Expand-Archive layout), e.g. F:\LHBTS\LHBTS.exe
if ($installDir.Length -eq 0) { $installDir = 'LHBTS' }
$installExe = Join-ExePath $root $installDir
Write-Output ("INSTALL_APP_DIR=" + $installDir)
Write-Output ("INSTALL_EXE_PATH=" + $installExe)

# Runtime: prefer running process directory (e.g. F:\1\LHBTS.exe)
$runtimeDir = ''
$runtimeExe = ''
$source = 'not_found'

if ($procName -and $procName.Trim().Length -gt 0) {
  $p = Get-Process -Name $procName.Trim() -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($null -ne $p -and $p.Path -and (Test-ExePath $p.Path)) {
    $runtimeExe = $p.Path
    $runtimeDir = Split-Path -Parent $runtimeExe
    $baseNorm = $root.TrimEnd('\')
    if ($runtimeDir.StartsWith($baseNorm, [System.StringComparison]::OrdinalIgnoreCase)) {
      $rel = $runtimeDir.Substring($baseNorm.Length).TrimStart('\')
      if ($rel.Length -eq 0) { $runtimeDir = '.' } else { $runtimeDir = $rel.Split('\')[0] }
    } else {
      $runtimeDir = Split-Path -Leaf $runtimeDir
    }
    $source = 'running_process'
  }
}

if (-not (Test-ExePath $runtimeExe)) {
  $candidates = New-Object System.Collections.Generic.List[string]
  if ($tryRaw -and $tryRaw.Trim().Length -gt 0) {
    $tryRaw.Trim() -split '[,\s;|]+' | ForEach-Object {
      $t = $_.Trim()
      if ($t.Length -gt 0 -and -not $candidates.Contains($t)) { [void]$candidates.Add($t) }
    }
  }
  foreach ($dir in $candidates) {
    $p = Join-ExePath $root $dir
    if (Test-ExePath $p) {
      $runtimeExe = $p
      $runtimeDir = if ($dir -eq '.' -or [string]::IsNullOrWhiteSpace($dir)) { '.' } else { $dir.Trim().Trim('\') }
      $source = 'candidate_list'
      break
    }
  }
}

if (-not (Test-ExePath $runtimeExe)) {
  $direct = Join-ExePath $root '.'
  if (Test-ExePath $direct) {
    $runtimeExe = $direct
    $runtimeDir = '.'
    $source = 'exe_dir_root'
  }
}

if (-not (Test-ExePath $runtimeExe)) {
  Get-ChildItem -Path $root -Directory -ErrorAction SilentlyContinue | ForEach-Object {
    if (Test-ExePath $runtimeExe) { return }
    $p = Join-ExePath $root $_.Name
    if (Test-ExePath $p) {
      $runtimeExe = $p
      $runtimeDir = $_.Name
      $source = 'auto_scan'
    }
  }
}

if (-not (Test-ExePath $runtimeExe)) {
  if (Test-ExePath $installExe) {
    $runtimeExe = $installExe
    $runtimeDir = $installDir
    $source = 'install_fallback'
  }
}

Write-Output ("RUNTIME_APP_DIR=" + $runtimeDir)
Write-Output ("RUNTIME_EXE_PATH=" + $runtimeExe)
Write-Output ("APP_DIR_SOURCE=" + $source)
