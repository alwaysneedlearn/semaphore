# Resolve LAND installer copy destination on target host.
# Env: LAND_INSTALLER_DEST_OVERRIDE, LAND_INSTALLER_FILENAME, LAND_INSTALLER_USE_DESKTOP,
#      LAND_INSTALL_PROFILE_USER, LAND_INSTALLER_FALLBACK_DIR

$custom = [string]$env:LAND_INSTALLER_DEST_OVERRIDE
if ($null -eq $custom) { $custom = '' }
$custom = $custom.Trim()

$fileName = [string]$env:LAND_INSTALLER_FILENAME
if ($null -eq $fileName) { $fileName = '' }
$fileName = $fileName.Trim()
if ($fileName.Length -eq 0) {
  Write-Output 'DEST_RESOLVE_FAILED|reason=empty_filename'
  exit 1
}

$useDesktop = ([string]$env:LAND_INSTALLER_USE_DESKTOP) -match '^(?i:true|1|yes)$'

if ($custom.Length -gt 0) {
  Write-Output $custom
  exit 0
}

if ($useDesktop) {
  $profileUser = [string]$env:LAND_INSTALL_PROFILE_USER
  if ($null -eq $profileUser) { $profileUser = '' }
  $profileUser = $profileUser.Trim()
  if ($profileUser.Length -eq 0) {
    Write-Output 'DESKTOP_RESOLVE_FAILED|reason=empty_profile_user'
    exit 1
  }
  $short = ($profileUser -split '\\')[-1]
  $desktop = Join-Path $env:SystemDrive "Users\$short\Desktop"
  if (-not (Test-Path -LiteralPath $desktop)) {
    $suffix = '\' + $short + '$'
    $loaded = Get-CimInstance Win32_UserProfile -Filter "Loaded=True" -ErrorAction SilentlyContinue |
      Where-Object { $_.LocalPath -and ($_.LocalPath.EndsWith($suffix)) } |
      Select-Object -First 1
    if ($loaded -and $loaded.LocalPath) {
      $desktop = Join-Path $loaded.LocalPath 'Desktop'
    }
  }
  if (-not (Test-Path -LiteralPath $desktop)) {
    Write-Output "DESKTOP_RESOLVE_FAILED|user=$profileUser|tried=$desktop"
    exit 1
  }
  Write-Output (Join-Path $desktop $fileName)
  exit 0
}

$fallback = [string]$env:LAND_INSTALLER_FALLBACK_DIR
if ($null -eq $fallback) { $fallback = '' }
$fallback = $fallback.Trim()
if ($fallback.Length -eq 0) {
  Write-Output 'DEST_RESOLVE_FAILED|reason=empty_fallback_dir'
  exit 1
}
Write-Output (Join-Path $fallback $fileName)
