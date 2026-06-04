# Probe install folder under EXE_DIR for EXE_NAME.
# Env: SEMAPHORE_EXE_DIR, SEMAPHORE_EXE_NAME, SEMAPHORE_APP_DIR_TRY (comma-separated, optional)
$root = [string]$env:SEMAPHORE_EXE_DIR
$exeName = [string]$env:SEMAPHORE_EXE_NAME
$tryRaw = [string]$env:SEMAPHORE_APP_DIR_TRY
if ($null -eq $root) { $root = '' }
if ($null -eq $exeName) { $exeName = '' }
$root = $root.Trim().TrimEnd('\')
$exeName = $exeName.Trim()
if ($root.Length -eq 0 -or $exeName.Length -eq 0) {
  Write-Output 'APP_DIR_RESOLVED='
  Write-Output 'EXE_PATH_RESOLVED='
  Write-Output 'APP_DIR_SOURCE=missing_env'
  exit 0
}

function Test-ExeUnder([string]$base, [string]$subDir) {
  if ($subDir -eq '.' -or [string]::IsNullOrWhiteSpace($subDir)) {
    $p = Join-Path -Path $base -ChildPath $exeName
  } else {
    $p = Join-Path -Path $base -ChildPath ($subDir.Trim().Trim('\'))
    $p = Join-Path -Path $p -ChildPath $exeName
  }
  return @{ Path = $p; Exists = (Test-Path -LiteralPath $p) }
}

$candidates = New-Object System.Collections.Generic.List[string]
if ($tryRaw -and $tryRaw.Trim().Length -gt 0) {
  $tryRaw.Trim() -split '[,\s;|]+' | ForEach-Object {
    $t = $_.Trim()
    if ($t.Length -gt 0 -and -not $candidates.Contains($t)) {
      [void]$candidates.Add($t)
    }
  }
}

foreach ($dir in $candidates) {
  $hit = Test-ExeUnder $root $dir
  if ($hit.Exists) {
    $resolvedDir = if ($dir -eq '.' -or [string]::IsNullOrWhiteSpace($dir)) { '.' } else { $dir.Trim().Trim('\') }
    Write-Output ("APP_DIR_RESOLVED=" + $resolvedDir)
    Write-Output ("EXE_PATH_RESOLVED=" + $hit.Path)
    Write-Output 'APP_DIR_SOURCE=candidate_list'
    exit 0
  }
}

$direct = Test-ExeUnder $root '.'
if ($direct.Exists) {
  Write-Output 'APP_DIR_RESOLVED=.'
  Write-Output ("EXE_PATH_RESOLVED=" + $direct.Path)
  Write-Output 'APP_DIR_SOURCE=exe_dir_root'
  exit 0
}

Get-ChildItem -Path $root -Directory -ErrorAction SilentlyContinue | ForEach-Object {
  $hit = Test-ExeUnder $root $_.Name
  if ($hit.Exists) {
    Write-Output ("APP_DIR_RESOLVED=" + $_.Name)
    Write-Output ("EXE_PATH_RESOLVED=" + $hit.Path)
    Write-Output 'APP_DIR_SOURCE=auto_scan'
    exit 0
  }
}

Write-Output 'APP_DIR_RESOLVED='
Write-Output 'EXE_PATH_RESOLVED='
Write-Output 'APP_DIR_SOURCE=not_found'
