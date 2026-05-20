# Reads SEMAPHORE_EXE_DIR_PREFERRED; prints EXE_DIR_RESOLVED / REQUESTED / CHOSEN
$preferred = [string]$env:SEMAPHORE_EXE_DIR_PREFERRED
if ($null -eq $preferred) { $preferred = '' }
$preferred = $preferred.Trim()
$defaultSuffix = '\Program Files\NEWARE'
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

$candidates = New-Object System.Collections.Generic.List[string]
[void]$candidates.Add($reqLetter)
if (-not $candidates.Contains('E')) { [void]$candidates.Add('E') }
if (-not $candidates.Contains('C')) { [void]$candidates.Add('C') }

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
