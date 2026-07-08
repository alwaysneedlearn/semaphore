# Resolve SINEXCEL program directory for redeploy.
# Env: SINEXCEL_PROCESS_NAME, SINEXCEL_FALLBACK_EXE_PATH, SINEXCEL_FALLBACK_EXE_DIR,
#      SINEXCEL_FALLBACK_APP_DIR (zip inner folder name hint, optional),
#      SINEXCEL_FALLBACK_EXE_DIR_PREFERRED, SINEXCEL_EXE_NAME
#
# Redeploy layout (in-place upgrade into current install folder):
#   program_dir   = current install dir (e.g. D:\盛弘软件\电池检测与化成V3.9.2.7-20240307-全包)
#   parent_dir    = parent of program_dir (zip copy destination)
#   extract dest  = program_dir (merge zip contents, overwrite files)

$ErrorActionPreference = 'Continue'

function Get-TrimmedEnv([string]$name) {
  $v = [Environment]::GetEnvironmentVariable($name)
  if ($null -eq $v) { return '' }
  return ([string]$v).Trim()
}

function Normalize-WindowsPath([string]$path) {
  if ([string]::IsNullOrWhiteSpace($path)) { return '' }
  $p = $path.Trim()
  $p = $p -replace '：', ':'
  $p = $p -replace '／', '/'
  $p = $p -replace '＼', '\'
  $p = $p.Replace('/', '\')
  if ($p -match '^([A-Za-z]):([^\\/]|$)') {
    $letter = $Matches[1]
    $rest = $Matches[2]
    if ($rest.Length -eq 0) {
      $p = $letter + ':\'
    } elseif ($rest[0] -ne '\') {
      $p = $letter + ':\' + $rest
    }
  }
  return $p.TrimEnd('\')
}

function Test-AbsoluteWindowsPath([string]$path) {
  $p = Normalize-WindowsPath $path
  if ($p.Length -eq 0) { return $false }
  return $p -match '^[A-Za-z]:\\'
}

function Get-DirectoryParent([string]$dir) {
  $dir = Normalize-WindowsPath $dir
  if (-not (Test-AbsoluteWindowsPath $dir)) { return '' }
  try {
    $parent = [System.IO.Path]::GetDirectoryName($dir)
    if ([string]::IsNullOrWhiteSpace($parent)) {
      return (Normalize-WindowsPath $dir).Substring(0, 3)
    }
    return Normalize-WindowsPath $parent
  } catch {
    return ''
  }
}

function Resolve-InstallFromExeFile([string]$exeFile) {
  $exeFile = Normalize-WindowsPath $exeFile
  if (-not (Test-AbsoluteWindowsPath $exeFile)) { return $null }
  $installDir = Get-DirectoryParent $exeFile
  if ($installDir.Length -eq 0) { return $null }
  $baseDir = Get-DirectoryParent $installDir
  if ($baseDir.Length -eq 0) { return $null }
  return @{
    ExePath = $exeFile
    InstallDir = $installDir
    BaseDir = $baseDir
  }
}

function Resolve-InstallFromDir([string]$installDir) {
  $installDir = Normalize-WindowsPath $installDir
  if (-not (Test-AbsoluteWindowsPath $installDir)) { return $null }
  $baseDir = Get-DirectoryParent $installDir
  if ($baseDir.Length -eq 0) { return $null }
  return @{
    InstallDir = $installDir
    BaseDir = $baseDir
  }
}

function Resolve-InstallFromBase([string]$baseDir) {
  $baseDir = Normalize-WindowsPath $baseDir
  if (-not (Test-AbsoluteWindowsPath $baseDir)) { return $null }
  return @{
    BaseDir = $baseDir
  }
}

function Test-ExePathExists([string]$exeFile) {
  $exeFile = Normalize-WindowsPath $exeFile
  if (-not (Test-AbsoluteWindowsPath $exeFile)) { return $false }
  try {
    return Test-Path -LiteralPath $exeFile
  } catch {
    return $false
  }
}

$procName = Get-TrimmedEnv 'SINEXCEL_PROCESS_NAME'
$procName = $procName -replace '(?i)\.exe$', ''
$fallbackExePath = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_PATH')
$fallbackExeDir = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_DIR')
$targetAppDir = Get-TrimmedEnv 'SINEXCEL_FALLBACK_APP_DIR'
$fallbackPreferred = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_FALLBACK_EXE_DIR_PREFERRED')
$exeName = Get-TrimmedEnv 'SINEXCEL_EXE_NAME'

$resolved = $null
$source = 'not_found'

if ($procName.Length -gt 0) {
  $filters = @(
    "Name='$procName.exe'",
    "Name='$procName'"
  )
  foreach ($flt in $filters) {
    $p = Get-CimInstance Win32_Process -Filter $flt -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $p) { continue }
    $candidate = Normalize-WindowsPath ([string]$p.ExecutablePath)
    if ($candidate.Length -eq 0) { continue }
    $hit = Resolve-InstallFromExeFile $candidate
    if ($null -ne $hit) {
      $resolved = $hit
      $source = 'process'
      break
    }
  }
}

if ($null -eq $resolved -and (Test-AbsoluteWindowsPath $fallbackExePath)) {
  $hit = Resolve-InstallFromExeFile $fallbackExePath
  if ($null -ne $hit) {
    $resolved = $hit
    $source = if (Test-ExePathExists $fallbackExePath) { 'scan' } else { 'scan_path_only' }
  }
}

if ($null -eq $resolved -and (Test-AbsoluteWindowsPath $fallbackExeDir)) {
  $hit = Resolve-InstallFromDir $fallbackExeDir
  if ($null -ne $hit) {
    $resolved = $hit
    $source = 'install_dir'
  }
}

if ($null -eq $resolved -and (Test-AbsoluteWindowsPath $fallbackPreferred)) {
  $hit = Resolve-InstallFromBase $fallbackPreferred
  if ($null -ne $hit) {
    $resolved = $hit
    $source = 'preferred_base'
  }
}

$programDir = ''
$parentDir = ''
$exePath = ''

if ($null -ne $resolved) {
  if ($resolved.InstallDir) {
    $programDir = Normalize-WindowsPath $resolved.InstallDir
    $parentDir = Normalize-WindowsPath $resolved.BaseDir
  } else {
    $parentDir = Normalize-WindowsPath $resolved.BaseDir
    if ($targetAppDir.Length -gt 0 -and $targetAppDir -ne '.') {
      $programDir = Normalize-WindowsPath (Join-Path $parentDir $targetAppDir)
    } else {
      $programDir = $parentDir
    }
  }

  if ($exeName.Length -gt 0) {
    $exePath = Normalize-WindowsPath (Join-Path $programDir $exeName)
  } elseif ($resolved.ExePath) {
    $exePath = Normalize-WindowsPath $resolved.ExePath
  }
}

if ($programDir.Length -eq 0 -or $parentDir.Length -eq 0 -or -not (Test-AbsoluteWindowsPath $programDir) -or -not (Test-AbsoluteWindowsPath $parentDir)) {
  Write-Output 'PROGRAM_DIR_RESOLVED='
  Write-Output 'PROGRAM_PARENT_RESOLVED='
  Write-Output 'EXE_PATH_RESOLVED='
  Write-Output "PROGRAM_DIR_SOURCE=$source"
  Write-Output ("PROGRAM_RESOLVE_ERROR|reason=invalid_paths|program_dir=$programDir|parent_dir=$parentDir|fallback_exe_path=$fallbackExePath|fallback_exe_dir=$fallbackExeDir|preferred=$fallbackPreferred|target_app_dir=$targetAppDir")
  exit 0
}

Write-Output "PROGRAM_DIR_RESOLVED=$programDir"
Write-Output "PROGRAM_PARENT_RESOLVED=$parentDir"
Write-Output "EXE_PATH_RESOLVED=$exePath"
Write-Output "PROGRAM_DIR_SOURCE=$source"
Write-Output "PROGRAM_CURRENT_INSTALL_DIR=$($resolved.InstallDir)"
exit 0
