# Reads SEMAPHORE_EXE_DIR_PREFERRED + optional SEMAPHORE_EXE_DIR_FALLBACK_DRIVES;
# prints EXE_DIR_RESOLVED / REQUESTED / CHOSEN / CANDIDATES.
$preferred = [string]$env:SEMAPHORE_EXE_DIR_PREFERRED
if ($null -eq $preferred) { $preferred = '' }
$preferred = $preferred.Trim()

$defaultSuffix = '\Program Files\DeviceApp'
$reqLetter = 'D'
$suffix = $defaultSuffix

if ($preferred -match '^([A-Za-z])\s*:?\s*(.*)$') {
  $reqLetter = $matches[1].ToUpperInvariant()
  $rest = $matches[2].Trim()
  if ($rest.Length -gt 0) {
    $suffix = $(if ($rest.StartsWith('\')) { $rest } else { '\' + $rest })
  } else {
    # EXE_DIR=D: or F: — install at drive root (e.g. D:\LHBTS\), not \Program Files\DeviceApp
    $suffix = '\'
  }
} elseif ($preferred.Length -gt 0) {
  $suffix = $(if ($preferred.StartsWith('\')) { $preferred } else { '\' + $preferred })
  $reqLetter = 'D'
}

function Test-DriveRootExists([string]$letter) {
  $root = ($letter.TrimEnd(':') + ':\')
  return Test-Path -LiteralPath $root
}

function Normalize-DriveToken([string]$token) {
  if ($null -eq $token) { return '' }
  $t = $token.Trim().ToUpperInvariant()
  if ($t -match '^([A-Z])(:)?$') { return $matches[1] }
  return ''
}

$fallbackRaw = [string]$env:SEMAPHORE_EXE_DIR_FALLBACK_DRIVES
if ($null -eq $fallbackRaw) { $fallbackRaw = '' }
$fallbackRaw = $fallbackRaw.Trim()

$candidates = New-Object System.Collections.Generic.List[string]
if ($fallbackRaw.Length -gt 0) {
  $parts = $fallbackRaw -split '[,\s;|]+' | Where-Object { $_ -and $_.Trim().Length -gt 0 }
  foreach ($p in $parts) {
    $d = Normalize-DriveToken $p
    if ($d.Length -gt 0 -and -not $candidates.Contains($d)) {
      [void]$candidates.Add($d)
    }
  }
}

if ($candidates.Count -eq 0) {
  [void]$candidates.Add($reqLetter)
  if (-not $candidates.Contains('E')) { [void]$candidates.Add('E') }
  if (-not $candidates.Contains('C')) { [void]$candidates.Add('C') }
} elseif (-not $candidates.Contains($reqLetter)) {
  [void]$candidates.Insert(0, $reqLetter)
} else {
  $idx = $candidates.IndexOf($reqLetter)
  if ($idx -gt 0) {
    [void]$candidates.RemoveAt($idx)
    [void]$candidates.Insert(0, $reqLetter)
  }
}

$probeRel = [string]$env:SEMAPHORE_EXE_PROBE_RELATIVE
if ($null -eq $probeRel) { $probeRel = '' }
$probeRel = $probeRel.Trim().TrimStart('\')

$scanLatest = $false
$scanLatestRaw = [string]$env:SEMAPHORE_EXE_SCAN_LATEST
if (-not [string]::IsNullOrWhiteSpace($scanLatestRaw)) {
  $scanLatest = $scanLatestRaw -match '^(?i:true|1|yes)$'
}
$scanName = [string]$env:SEMAPHORE_EXE_SCAN_NAME
if ($null -eq $scanName) { $scanName = '' }
$scanName = $scanName.Trim()
if ($scanName.Length -eq 0 -and $probeRel.Length -gt 0) {
  $scanName = Split-Path -Leaf $probeRel
}
$scanMaxDepth = 2
$scanDepthRaw = [string]$env:SEMAPHORE_EXE_SCAN_MAX_DEPTH
if (-not [string]::IsNullOrWhiteSpace($scanDepthRaw)) {
  [int]::TryParse($scanDepthRaw, [ref]$scanMaxDepth) | Out-Null
}
if ($scanMaxDepth -lt 0) { $scanMaxDepth = 0 }
if ($scanMaxDepth -gt 5) { $scanMaxDepth = 5 }

$skipDirNames = @(
  '$Recycle.Bin', 'System Volume Information', 'Recovery', 'Windows',
  'ProgramData', 'PerfLogs', 'Config.Msi'
)

function Search-NewestExeUnder {
  param(
    [string]$RootDir,
    [string]$ExeFileName,
    [int]$MaxDirDepth,
    [int]$CurrentDepth
  )
  $best = $null
  if (-not (Test-Path -LiteralPath $RootDir)) { return $null }
  try {
    $files = @(Get-ChildItem -LiteralPath $RootDir -File -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -ieq $ExeFileName })
    foreach ($f in $files) {
      if ($null -eq $best -or $f.LastWriteTime -gt $best.LastWriteTime) {
        $best = $f
      }
    }
    if ($CurrentDepth -ge $MaxDirDepth) { return $best }
    $dirs = @(Get-ChildItem -LiteralPath $RootDir -Directory -ErrorAction SilentlyContinue)
    foreach ($d in $dirs) {
      if ($skipDirNames -contains $d.Name) { continue }
      $found = Search-NewestExeUnder -RootDir $d.FullName -ExeFileName $ExeFileName `
        -MaxDirDepth $MaxDirDepth -CurrentDepth ($CurrentDepth + 1)
      if ($null -ne $found -and ($null -eq $best -or $found.LastWriteTime -gt $best.LastWriteTime)) {
        $best = $found
      }
    }
  } catch {
    # ignore permission errors on shallow scan
  }
  return $best
}

function Find-NewestExeAcrossDrives {
  param(
    [System.Collections.Generic.List[string]]$DriveLetters,
    [string]$ExeFileName,
    [int]$MaxDirDepth
  )
  $globalBest = $null
  $globalDrive = ''
  foreach ($L in $DriveLetters) {
    if (-not (Test-DriveRootExists $L)) { continue }
    $root = $L.TrimEnd(':') + ':\'
    $localBest = Search-NewestExeUnder -RootDir $root -ExeFileName $ExeFileName `
      -MaxDirDepth $MaxDirDepth -CurrentDepth 0
    if ($null -eq $localBest) { continue }
    Write-Output ("EXE_SCAN_DRIVE|$L|newest=$($localBest.FullName)|mtime=$($localBest.LastWriteTime.ToString('o'))")
    if ($null -eq $globalBest -or $localBest.LastWriteTime -gt $globalBest.LastWriteTime) {
      $globalBest = $localBest
      $globalDrive = $L.TrimEnd(':').ToUpperInvariant()
    }
  }
  return @{ File = $globalBest; Drive = $globalDrive }
}

$resolvedExePath = ''
$chosen = ''
$chooseSource = 'drive_root'

if ($scanLatest -and $scanName.Length -gt 0) {
  $scanResult = Find-NewestExeAcrossDrives -DriveLetters $candidates -ExeFileName $scanName -MaxDirDepth $scanMaxDepth
  if ($null -ne $scanResult.File) {
    $resolvedExePath = $scanResult.File.FullName
    $chosen = $scanResult.Drive
    $chooseSource = 'scan_latest'
    Write-Output ("EXE_PATH_RESOLVED=$resolvedExePath")
    Write-Output ("EXE_SCAN_WINNER|drive=$chosen|mtime=$($scanResult.File.LastWriteTime.ToString('o'))")
  } else {
    Write-Output "EXE_SCAN_MISS|name=$scanName|max_depth=$scanMaxDepth"
  }
}

# Prefer drive where install path already exists (e.g. F:\LHBTS\LHBTS.exe when D:\ has no folder).
if ($chosen.Length -eq 0) {
  $chooseSource = 'drive_root'
}
if ($chosen.Length -eq 0 -and $probeRel.Length -gt 0) {
  foreach ($L in $candidates) {
    if (-not (Test-DriveRootExists $L)) { continue }
    $root = $L + ':' + $suffix
    $probePath = Join-Path -Path $root.TrimEnd('\') -ChildPath $probeRel
    if (Test-Path -LiteralPath $probePath) {
      $chosen = $L.TrimEnd(':').ToUpperInvariant()
      $chooseSource = 'probe_exe'
      break
    }
  }
}

if ($chosen.Length -eq 0 -and $probeRel.Length -gt 0) {
  $dirRel = $probeRel
  if ($probeRel.Contains('\')) {
    $dirRel = $probeRel.Substring(0, $probeRel.LastIndexOf('\'))
  }
  foreach ($L in $candidates) {
    if (-not (Test-DriveRootExists $L)) { continue }
    $root = $L + ':' + $suffix
    $dirPath = Join-Path -Path $root.TrimEnd('\') -ChildPath $dirRel
    if ((Test-Path -LiteralPath $dirPath) -and (Get-Item -LiteralPath $dirPath).PSIsContainer) {
      $chosen = $L.TrimEnd(':').ToUpperInvariant()
      $chooseSource = 'probe_dir'
      break
    }
  }
}

if ($chosen.Length -eq 0) {
  foreach ($L in $candidates) {
    if (Test-DriveRootExists $L) {
      $chosen = $L.TrimEnd(':').ToUpperInvariant()
      $chooseSource = 'drive_root'
      break
    }
  }
}

if ($chosen.Length -eq 0) {
  $chosen = 'C'
  $chooseSource = 'fallback_c'
}

if ($chooseSource -eq 'scan_latest' -and $resolvedExePath.Length -gt 0) {
  $resolved = [System.IO.Path]::GetDirectoryName($resolvedExePath)
  if ([string]::IsNullOrWhiteSpace($resolved)) {
    $resolved = $chosen + ':' + $suffix
  }
} else {
  $resolved = $chosen + ':' + $suffix
}
Write-Output "EXE_DIR_RESOLVED=$resolved"
Write-Output "EXE_DIR_REQUESTED=$reqLetter"
Write-Output "EXE_DIR_CHOSEN=$chosen"
Write-Output "EXE_DIR_CHOOSE_SOURCE=$chooseSource"
Write-Output ("EXE_DIR_CANDIDATES=" + (($candidates | ForEach-Object { $_ }) -join ','))
