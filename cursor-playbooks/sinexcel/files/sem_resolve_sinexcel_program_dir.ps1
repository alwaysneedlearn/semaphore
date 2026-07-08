# Resolve SINEXCEL program directory for redeploy (process → scanned exe → install paths).
# Env: SINEXCEL_PROCESS_NAME, SINEXCEL_FALLBACK_EXE_PATH, SINEXCEL_FALLBACK_EXE_DIR,
#      SINEXCEL_FALLBACK_APP_DIR, SINEXCEL_FALLBACK_EXE_DIR_PREFERRED, SINEXCEL_EXE_NAME

$ErrorActionPreference = 'Continue'

function Get-TrimmedEnv([string]$name) {
  $v = [Environment]::GetEnvironmentVariable($name)
  if ($null -eq $v) { return '' }
  return ([string]$v).Trim()
}

function Test-AbsoluteWindowsPath([string]$path) {
  if ([string]::IsNullOrWhiteSpace($path)) { return $false }
  return $path -match '^[A-Za-z]:\\'
}

function Normalize-Dir([string]$path) {
  if ([string]::IsNullOrWhiteSpace($path)) { return '' }
  return $path.Trim().TrimEnd('\')
}

function Get-ParentOrDriveRoot([string]$dir) {
  $dir = Normalize-Dir $dir
  if (-not (Test-AbsoluteWindowsPath $dir)) { return '' }
  $parent = Split-Path -LiteralPath $dir -Parent
  if ([string]::IsNullOrWhiteSpace($parent)) {
    $drive = $dir.Substring(0, 2)
    return $drive + '\'
  }
  return Normalize-Dir $parent
}

function Build-ProgramDirFromBase([string]$baseDir, [string]$appDir) {
  $baseDir = Normalize-Dir $baseDir
  if (-not (Test-AbsoluteWindowsPath $baseDir)) { return '' }
  $appDir = $appDir.Trim()
  if ($appDir.Length -eq 0 -or $appDir -eq '.') {
    return $baseDir
  }
  return Normalize-Dir (Join-Path $baseDir $appDir)
}

$procName = Get-TrimmedEnv 'SINEXCEL_PROCESS_NAME'
$procName = $procName -replace '(?i)\.exe$', ''
$fallbackExePath = Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_PATH'
$fallbackExeDir = Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_DIR'
$fallbackAppDir = Get-TrimmedEnv 'SINEXCEL_FALLBACK_APP_DIR'
$fallbackPreferred = Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_DIR_PREFERRED'
$exeName = Get-TrimmedEnv 'SINEXCEL_EXE_NAME'

$exePath = ''
$programDir = ''
$source = 'not_found'

if ($procName.Length -gt 0) {
  $filters = @(
    "Name='$procName.exe'",
    "Name='$procName'"
  )
  foreach ($flt in $filters) {
    $p = Get-CimInstance Win32_Process -Filter $flt -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $p) { continue }
    $candidate = [string]$p.ExecutablePath
    $candidate = $candidate.Trim()
    if ($candidate.Length -gt 0 -and (Test-Path -LiteralPath $candidate)) {
      $exePath = $candidate
      $programDir = Normalize-Dir (Split-Path -LiteralPath $exePath -Parent)
      $source = 'process'
      break
    }
  }
}

if ($source -ne 'process') {
  if ((Test-AbsoluteWindowsPath $fallbackExePath) -and (Test-Path -LiteralPath $fallbackExePath)) {
    $exePath = $fallbackExePath
    $programDir = Normalize-Dir (Split-Path -LiteralPath $exePath -Parent)
    $source = 'scan'
  } elseif ((Test-AbsoluteWindowsPath $fallbackExeDir)) {
    $programDir = Build-ProgramDirFromBase $fallbackExeDir $fallbackAppDir
    if ($programDir.Length -gt 0) {
      if ($exeName.Length -gt 0) {
        $exePath = Join-Path $programDir $exeName
      }
      $source = 'install_paths'
    }
  } elseif ((Test-AbsoluteWindowsPath $fallbackPreferred)) {
    $programDir = Build-ProgramDirFromBase $fallbackPreferred $fallbackAppDir
    if ($programDir.Length -gt 0) {
      if ($exeName.Length -gt 0) {
        $exePath = Join-Path $programDir $exeName
      }
      $source = 'preferred_install_paths'
    }
  }
}

if ($programDir.Length -gt 0 -and -not (Test-AbsoluteWindowsPath $programDir)) {
  $programDir = ''
  $exePath = ''
  $source = 'not_found'
}

$parentDir = ''
if ($programDir.Length -gt 0) {
  $parentDir = Get-ParentOrDriveRoot $programDir
}

if ($programDir.Length -eq 0 -or $parentDir.Length -eq 0 -or -not (Test-AbsoluteWindowsPath $parentDir)) {
  Write-Output 'PROGRAM_DIR_RESOLVED='
  Write-Output 'PROGRAM_PARENT_RESOLVED='
  Write-Output 'EXE_PATH_RESOLVED='
  Write-Output "PROGRAM_DIR_SOURCE=$source"
  Write-Output ("PROGRAM_RESOLVE_ERROR|reason=invalid_paths|program_dir=$programDir|parent_dir=$parentDir|fallback_exe_path=$fallbackExePath|fallback_exe_dir=$fallbackExeDir|preferred=$fallbackPreferred")
  exit 0
}

Write-Output "PROGRAM_DIR_RESOLVED=$programDir"
Write-Output "PROGRAM_PARENT_RESOLVED=$parentDir"
Write-Output "EXE_PATH_RESOLVED=$exePath"
Write-Output "PROGRAM_DIR_SOURCE=$source"
exit 0
