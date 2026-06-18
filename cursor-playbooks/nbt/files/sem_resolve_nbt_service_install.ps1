# Resolve NBT service exe under NBT_SERVICE_BASE_DIR using NBT_SERVICE_NAME / NBT_SERVICE_EXE_NAME.
# Env: NBT_SERVICE_BASE_DIR, NBT_SERVICE_NAME, NBT_SERVICE_EXE_NAME,
#      NBT_SERVICE_SCAN_MAX_DEPTH, NBT_SERVICE_SCAN_LATEST

$ErrorActionPreference = 'Continue'
$base = [string]$env:NBT_SERVICE_BASE_DIR
if ($null -eq $base) { $base = '' }
$base = $base.Trim().TrimEnd('\')
$svcName = [string]$env:NBT_SERVICE_NAME
if ($null -eq $svcName) { $svcName = '' }
$svcName = $svcName.Trim()
$exeName = [string]$env:NBT_SERVICE_EXE_NAME
if ([string]::IsNullOrWhiteSpace($exeName)) {
  $exeName = $svcName
  if (-not $exeName.EndsWith('.exe', [StringComparison]::OrdinalIgnoreCase)) {
    $exeName = $exeName + '.exe'
  }
}
$maxDepth = 3
if (-not [string]::IsNullOrWhiteSpace($env:NBT_SERVICE_SCAN_MAX_DEPTH)) {
  [void][int]::TryParse($env:NBT_SERVICE_SCAN_MAX_DEPTH, [ref]$maxDepth)
  if ($maxDepth -lt 0) { $maxDepth = 0 }
}
$scanLatest = $true
$scanLatestRaw = [string]$env:NBT_SERVICE_SCAN_LATEST
if (-not [string]::IsNullOrWhiteSpace($scanLatestRaw)) {
  $scanLatest = $scanLatestRaw -match '^(?i:true|1|yes)$'
}

function Test-ExeAt([string]$Dir) {
  if ([string]::IsNullOrWhiteSpace($Dir)) { return $null }
  $p = Join-Path $Dir $exeName
  if (Test-Path -LiteralPath $p) { return $p }
  return $null
}

$resolved = $null
$source = 'not_found'

if ($base.Length -gt 0) {
  $resolved = Test-ExeAt $base
  if ($resolved) { $source = 'exe_in_base_dir' }

  if (-not $resolved -and $svcName.Length -gt 0) {
    $sub = Join-Path $base $svcName
    $resolved = Test-ExeAt $sub
    if ($resolved) { $source = 'exe_in_service_name_subdir' }
  }

  if (-not $resolved -and (Test-Path -LiteralPath $base)) {
    $hits = @(
      Get-ChildItem -LiteralPath $base -Filter $exeName -File -Recurse -Depth $maxDepth -ErrorAction SilentlyContinue
    )
    if ($hits.Count -gt 0) {
      if ($scanLatest) {
        $pick = $hits | Sort-Object LastWriteTime -Descending | Select-Object -First 1
      } else {
        $pick = $hits | Select-Object -First 1
      }
      $resolved = $pick.FullName
      $source = 'scan_under_base'
    }
  }
}

if ($resolved) {
  $installDir = Split-Path -Parent $resolved
  Write-Output "SERVICE_EXE_PATH=$resolved"
  Write-Output "SERVICE_INSTALL_DIR=$installDir"
  Write-Output "SERVICE_BASE_DIR=$base"
  Write-Output "SERVICE_RESOLVE_SOURCE=$source"
  exit 0
}

Write-Output 'SERVICE_EXE_PATH='
Write-Output 'SERVICE_INSTALL_DIR='
Write-Output "SERVICE_BASE_DIR=$base"
Write-Output "SERVICE_RESOLVE_SOURCE=$source"
Write-Output "SERVICE_EXE_NOT_FOUND|base=$base|exe=$exeName|depth=$maxDepth"
exit 0
