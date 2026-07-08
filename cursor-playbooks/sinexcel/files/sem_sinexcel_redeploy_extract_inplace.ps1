# Extract upgrade zip into an existing program directory (in-place overwrite).
# Env: SINEXCEL_ZIP_PATH, SINEXCEL_PROGRAM_DIR, SINEXCEL_ZIP_INNER_NAME (optional top-level folder in zip)

$ErrorActionPreference = 'Stop'

function Get-TrimmedEnv([string]$name) {
  $v = [Environment]::GetEnvironmentVariable($name)
  if ($null -eq $v) { return '' }
  return ([string]$v).Trim()
}

function Normalize-WindowsPath([string]$path) {
  if ([string]::IsNullOrWhiteSpace($path)) { return '' }
  $p = $path.Trim().Replace('/', '\')
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

$zipPath = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_ZIP_PATH')
$programDir = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_PROGRAM_DIR')
$innerName = Get-TrimmedEnv 'SINEXCEL_ZIP_INNER_NAME'

if (-not (Test-Path -LiteralPath $zipPath)) {
  Write-Output "EXTRACT_ERROR|reason=zip_missing|path=$zipPath"
  exit 1
}
if ([string]::IsNullOrWhiteSpace($programDir)) {
  Write-Output "EXTRACT_ERROR|reason=program_dir_empty"
  exit 1
}
if (-not (Test-Path -LiteralPath $programDir)) {
  New-Item -ItemType Directory -Path $programDir -Force | Out-Null
}

$tempDir = Join-Path $env:TEMP ('sem_sinexcel_redeploy_' + [Guid]::NewGuid().ToString('N'))
try {
  New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
  Expand-Archive -LiteralPath $zipPath -DestinationPath $tempDir -Force

  $topItems = @(Get-ChildItem -LiteralPath $tempDir -Force)
  $sourceDir = $tempDir

  if ($topItems.Count -eq 1 -and $topItems[0].PSIsContainer) {
    $onlyDir = $topItems[0]
    if ($innerName.Length -eq 0 -or ($onlyDir.Name -ieq $innerName)) {
      $sourceDir = $onlyDir.FullName
    }
  }

  Copy-Item -LiteralPath (Join-Path $sourceDir '*') -Destination $programDir -Recurse -Force
  Write-Output ("EXTRACT_OK|zip=$zipPath|dest=$programDir|mode=inplace_merge|source=$sourceDir")
  exit 0
} catch {
  Write-Output ("EXTRACT_ERROR|reason=" + $_.Exception.Message)
  exit 1
} finally {
  if (Test-Path -LiteralPath $tempDir) {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
  }
}
