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

# Prefer drive where install path already exists (e.g. F:\LHBTS\LHBTS.exe when D:\ has no folder).
$chosen = ''
$chooseSource = 'drive_root'
if ($probeRel.Length -gt 0) {
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

$resolved = $chosen + ':' + $suffix
Write-Output "EXE_DIR_RESOLVED=$resolved"
Write-Output "EXE_DIR_REQUESTED=$reqLetter"
Write-Output "EXE_DIR_CHOSEN=$chosen"
Write-Output "EXE_DIR_CHOOSE_SOURCE=$chooseSource"
Write-Output ("EXE_DIR_CANDIDATES=" + (($candidates | ForEach-Object { $_ }) -join ','))
