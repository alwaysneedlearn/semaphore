# Backup program directory before in-place redeploy extract.
# Env: SINEXCEL_PROGRAM_DIR

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

$progDir = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_PROGRAM_DIR')

try {
  if ([string]::IsNullOrWhiteSpace($progDir)) {
    Write-Output 'BACKUP_ERROR|reason=program_dir_empty'
    exit 1
  }
  if (-not (Test-Path -LiteralPath $progDir)) {
    Write-Output "BACKUP_ERROR|reason=old_program_dir_missing|dir=$progDir"
    exit 1
  }

  $stamp = Get-Date -Format 'yyyyMMdd_HHmmss'
  $leaf = [System.IO.Path]::GetFileName($progDir)
  $parent = [System.IO.Path]::GetDirectoryName($progDir)
  if ([string]::IsNullOrWhiteSpace($parent)) {
    $parent = $progDir
  }
  $backupDir = Join-Path $parent ($leaf + '.bak_' + $stamp)

  if (Test-Path -LiteralPath $backupDir) {
    Write-Output "BACKUP_ERROR|reason=backup_exists|path=$backupDir"
    exit 1
  }

  New-Item -ItemType Directory -Path $backupDir -Force | Out-Null

  $robocopy = Join-Path $env:SystemRoot 'System32\robocopy.exe'
  if (Test-Path -LiteralPath $robocopy) {
    & $robocopy $progDir $backupDir /E /COPY:DAT /DCOPY:DAT /R:2 /W:2 /NFL /NDL /NJH /NJS /NP | Out-Null
    $robocode = $LASTEXITCODE
    if ($robocode -ge 8) {
      Write-Output "BACKUP_ERROR|reason=robocopy_failed|code=$robocode|from=$progDir|to=$backupDir"
      exit 1
    }
    Write-Output "BACKUP_OK|from=$progDir|to=$backupDir|method=robocopy|code=$robocode"
    exit 0
  }

  Copy-Item -LiteralPath $progDir -Destination $backupDir -Recurse -Force
  Write-Output "BACKUP_OK|from=$progDir|to=$backupDir|method=copy_item"
  exit 0
} catch {
  Write-Output ("BACKUP_ERROR|reason=" + $_.Exception.Message + "|dir=$progDir")
  exit 1
}
