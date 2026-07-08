# Patch one config file under program_dir\config (INI upsert or JSON merge).
# Env: SINEXCEL_PROGRAM_DIR, SINEXCEL_CONFIG_REL_DIR (default config),
#      SINEXCEL_CONFIG_FILE_NAME, SINEXCEL_CONFIG_PATCH_JSON

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

function Upsert-IniKey {
  param(
    [System.Collections.Generic.List[string]]$LinesRef,
    [string]$Section,
    [string]$Key,
    [string]$Value
  )
  $secStart = -1
  $secEnd = $LinesRef.Count
  for ($i = 0; $i -lt $LinesRef.Count; $i++) {
    if ($LinesRef[$i].Trim() -eq "[$Section]") {
      $secStart = $i
      for ($j = $i + 1; $j -lt $LinesRef.Count; $j++) {
        if ($LinesRef[$j].Trim() -match '^\[.+\]$') { $secEnd = $j; break }
      }
      break
    }
  }
  $newLine = "$Key=$Value"
  if ($secStart -lt 0) {
    if ($LinesRef.Count -gt 0 -and $LinesRef[$LinesRef.Count - 1].Trim() -ne '') { [void]$LinesRef.Add('') }
    [void]$LinesRef.Add("[$Section]")
    [void]$LinesRef.Add($newLine)
    return
  }
  for ($k = $secStart + 1; $k -lt $secEnd; $k++) {
    if ($LinesRef[$k].Trim() -match "^$([regex]::Escape($Key))=") {
      $LinesRef[$k] = $newLine
      return
    }
  }
  $LinesRef.Insert($secEnd, $newLine)
}

try {
  $programDir = Normalize-WindowsPath (Get-TrimmedEnv 'SINEXCEL_PROGRAM_DIR')
  $relDir = Get-TrimmedEnv 'SINEXCEL_CONFIG_REL_DIR'
  if ([string]::IsNullOrWhiteSpace($relDir)) { $relDir = 'config' }
  $fileName = Get-TrimmedEnv 'SINEXCEL_CONFIG_FILE_NAME'
  $patchJson = Get-TrimmedEnv 'SINEXCEL_CONFIG_PATCH_JSON'

  if ([string]::IsNullOrWhiteSpace($programDir)) {
    Write-Output 'CFG_ERROR|reason=program_dir_empty'
    exit 1
  }
  if ([string]::IsNullOrWhiteSpace($fileName)) {
    Write-Output 'CFG_ERROR|reason=file_name_empty'
    exit 1
  }

  $configDir = Join-Path $programDir $relDir
  if (-not (Test-Path -LiteralPath $configDir)) {
    New-Item -ItemType Directory -Path $configDir -Force | Out-Null
  }
  $targetPath = Join-Path $configDir $fileName

  $patch = $null
  if (-not [string]::IsNullOrWhiteSpace($patchJson)) {
    $patch = $patchJson | ConvertFrom-Json
  }
  $patchHt = @{}
  if ($null -ne $patch) {
    foreach ($p in $patch.PSObject.Properties) {
      $patchHt[$p.Name] = $p.Value
    }
  }
  if ($patchHt.Count -eq 0) {
    Write-Output ("CFG_SKIP|file=" + $fileName + "|reason=empty_patch")
    exit 0
  }

  $ext = [System.IO.Path]::GetExtension($fileName).ToLowerInvariant()
  if ($ext -eq '.json') {
    $root = @{}
    if (Test-Path -LiteralPath $targetPath) {
      $raw = Get-Content -LiteralPath $targetPath -Raw -Encoding UTF8
      if ($raw -and $raw.Trim().Length -gt 0) {
        $parsed = $raw | ConvertFrom-Json
        foreach ($p in $parsed.PSObject.Properties) {
          $root[$p.Name] = $p.Value
        }
      }
    }
    foreach ($k in $patchHt.Keys) {
      $root[$k] = $patchHt[$k]
    }
    ($root | ConvertTo-Json -Depth 32) | Set-Content -LiteralPath $targetPath -Encoding UTF8
    Write-Output ("CFG_OK|file=" + $fileName + "|format=json|keys=" + (($patchHt.Keys | Sort-Object) -join ','))
    exit 0
  }

  $lines = New-Object 'System.Collections.Generic.List[string]'
  if (Test-Path -LiteralPath $targetPath) {
    $lines.AddRange([string[]](Get-Content -LiteralPath $targetPath -Encoding UTF8))
  }
  foreach ($k in $patchHt.Keys) {
    $val = [string]$patchHt[$k]
    if ($k -match '^(.+)\.(.+)$') {
      Upsert-IniKey -LinesRef $lines -Section $Matches[1] -Key $Matches[2] -Value $val
    } else {
      Upsert-IniKey -LinesRef $lines -Section 'DEFAULT' -Key $k -Value $val
    }
  }
  $lines | Set-Content -LiteralPath $targetPath -Encoding UTF8
  Write-Output ("CFG_OK|file=" + $fileName + "|format=ini|keys=" + (($patchHt.Keys | Sort-Object) -join ','))
  exit 0
} catch {
  Write-Output ("CFG_ERROR|reason=" + $_.Exception.Message)
  exit 1
}
