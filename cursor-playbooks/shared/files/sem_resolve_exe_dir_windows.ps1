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
}

$chosen = 'C'
foreach ($L in $candidates) {
  if (Test-DriveRootExists $L) {
    $chosen = $L.TrimEnd(':').ToUpperInvariant()
    break
  }
}

$resolved = $chosen + ':' + $suffix
Write-Output "EXE_DIR_RESOLVED=$resolved"
Write-Output "EXE_DIR_REQUESTED=$reqLetter"
Write-Output "EXE_DIR_CHOSEN=$chosen"
Write-Output ("EXE_DIR_CANDIDATES=" + (($candidates | ForEach-Object { $_ }) -join ','))
