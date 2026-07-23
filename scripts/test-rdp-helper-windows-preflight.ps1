#Requires -Version 5.1
<#
.SYNOPSIS
  Semaphore RDP Helper — preflight checks on a non-admin Windows PC.

.DESCRIPTION
  Verifies operations the Helper needs WITHOUT elevation, and intentionally
  probes a few admin-only actions so you can see what would fail.

  Run in a normal (non-elevated) PowerShell:

    powershell -NoProfile -ExecutionPolicy Bypass -File .\test-rdp-helper-windows-preflight.ps1

  Optional cleanup of protocol test keys is automatic unless -KeepTestKeys.

.NOTES
  Does not require administrator. Safe to run: uses temp ports, HKCU only for
  real Helper path, and rolls back test protocol registration.
#>
[CmdletBinding()]
param(
    [switch]$KeepTestKeys,
    [switch]$SkipSshListen,
    [switch]$SkipMstscLaunch
)

$ErrorActionPreference = 'Continue'
$ProtocolScheme = 'semaphore-rdp-preflight'
$Results = New-Object System.Collections.Generic.List[object]

function Add-Result {
    param(
        [string]$Name,
        [ValidateSet('PASS', 'FAIL', 'WARN', 'INFO', 'SKIP')]$Status,
        [string]$Detail,
        [bool]$NeedsAdminIfFail = $false
    )
    $Results.Add([pscustomobject]@{
            Name              = $Name
            Status            = $Status
            NeedsAdminIfFail  = $NeedsAdminIfFail
            Detail            = $Detail
        }) | Out-Null
    $color = switch ($Status) {
        'PASS' { 'Green' }
        'FAIL' { 'Red' }
        'WARN' { 'Yellow' }
        'SKIP' { 'DarkGray' }
        default { 'Cyan' }
    }
    Write-Host ("[{0}] {1}: {2}" -f $Status, $Name, $Detail) -ForegroundColor $color
}

function Test-IsAdministrator {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

Write-Host ''
Write-Host '=== Semaphore RDP Helper Windows Preflight ===' -ForegroundColor Cyan
Write-Host ("User: {0}" -f $env:USERNAME)
Write-Host ("Computer: {0}" -f $env:COMPUTERNAME)
Write-Host ("Elevated admin: {0}" -f (Test-IsAdministrator))
Write-Host ''

# --- 1. Current elevation (INFO) ---
if (Test-IsAdministrator) {
    Add-Result 'Elevation' 'WARN' 'This shell IS elevated. Re-run as a normal user to match Helper PCs.' $false
} else {
    Add-Result 'Elevation' 'PASS' 'Not elevated (matches target Helper PCs).' $false
}

# --- 2. User-writable install dir (AppData) ---
$helperDir = Join-Path $env:LOCALAPPDATA 'SemaphoreRdpHelper'
$probeFile = Join-Path $helperDir '_preflight_write_test.txt'
try {
    New-Item -ItemType Directory -Path $helperDir -Force | Out-Null
    'ok' | Set-Content -LiteralPath $probeFile -Encoding UTF8
    if (Test-Path -LiteralPath $probeFile) {
        Add-Result 'UserInstallDir' 'PASS' "Can write $helperDir" $false
    } else {
        Add-Result 'UserInstallDir' 'FAIL' "Write appeared to succeed but file missing: $probeFile" $false
    }
} catch {
    Add-Result 'UserInstallDir' 'FAIL' $_.Exception.Message $false
} finally {
    Remove-Item -LiteralPath $probeFile -Force -ErrorAction SilentlyContinue
}

# --- 3. Program Files write (expect FAIL without admin) ---
$pfProbe = Join-Path ${env:ProgramFiles} 'SemaphoreRdpHelper_preflight_probe.txt'
try {
    'should-fail' | Set-Content -LiteralPath $pfProbe -Encoding UTF8 -ErrorAction Stop
    Remove-Item -LiteralPath $pfProbe -Force -ErrorAction SilentlyContinue
    Add-Result 'ProgramFilesWrite' 'WARN' 'Write to Program Files succeeded (unexpected on locked-down PCs).' $true
} catch {
    Add-Result 'ProgramFilesWrite' 'PASS' 'Cannot write Program Files without admin (expected). Use AppData/portable.' $true
}

# --- 4. HKCU protocol registration (Helper path — no admin) ---
$hkcuKey = "HKCU:\Software\Classes\$ProtocolScheme"
$hkcuCmd = "HKCU:\Software\Classes\$ProtocolScheme\shell\open\command"
try {
    New-Item -Path $hkcuKey -Force | Out-Null
    New-ItemProperty -Path $hkcuKey -Name 'URL Protocol' -Value '' -PropertyType String -Force | Out-Null
    New-Item -Path $hkcuCmd -Force | Out-Null
    $cmdValue = '"' + (Join-Path $env:LOCALAPPDATA 'SemaphoreRdpHelper\semaphore-rdp-helper.exe') + '" "%1"'
    Set-ItemProperty -Path $hkcuCmd -Name '(default)' -Value $cmdValue -Force
    $readBack = (Get-ItemProperty -Path $hkcuCmd -Name '(default)').'(default)'
    if ($readBack -eq $cmdValue) {
        Add-Result 'HKCU_Protocol' 'PASS' ("Registered {0}:// under HKCU (no admin)." -f $ProtocolScheme) $false
    } else {
        Add-Result 'HKCU_Protocol' 'FAIL' 'HKCU write/read mismatch.' $false
    }
} catch {
    Add-Result 'HKCU_Protocol' 'FAIL' $_.Exception.Message $false
} finally {
    if (-not $KeepTestKeys) {
        Remove-Item -Path $hkcuKey -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- 5. HKLM protocol registration (expect FAIL without admin) ---
$hklmKey = "HKLM:\Software\Classes\$ProtocolScheme"
try {
    New-Item -Path $hklmKey -Force -ErrorAction Stop | Out-Null
    Remove-Item -Path $hklmKey -Recurse -Force -ErrorAction SilentlyContinue
    Add-Result 'HKLM_Protocol' 'WARN' 'HKLM write succeeded (running elevated or loose policy).' $true
} catch {
    Add-Result 'HKLM_Protocol' 'PASS' 'Cannot write HKLM without admin (expected). Helper must use HKCU only.' $true
}

# --- 6. OpenSSH client ---
$ssh = Get-Command ssh -ErrorAction SilentlyContinue
if ($ssh) {
    $ver = & ssh -V 2>&1 | Out-String
    Add-Result 'OpenSSH' 'PASS' ("ssh found: {0} ({1})" -f $ssh.Source, ($ver.Trim())) $false
} else {
    Add-Result 'OpenSSH' 'FAIL' 'ssh.exe not on PATH. Enable Optional Feature "OpenSSH Client" (may need admin once).' $true
}

# --- 7. ssh -L style localhost bind (TcpListener on high port) ---
if ($SkipSshListen) {
    Add-Result 'LocalPortBind' 'SKIP' 'Skipped by -SkipSshListen' $false
} else {
    $listener = $null
    $port = Get-Random -Minimum 18000 -Maximum 28000
    try {
        $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $port)
        $listener.Start()
        Add-Result 'LocalPortBind' 'PASS' "Can bind 127.0.0.1:$port (ssh -L style)." $false
    } catch {
        Add-Result 'LocalPortBind' 'FAIL' $_.Exception.Message $false
    } finally {
        if ($listener) { $listener.Stop() }
    }
}

# --- 8. ControlMaster path dir writable ---
$sshDir = Join-Path $env:USERPROFILE '.ssh'
$cmProbe = Join-Path $sshDir 'cm-preflight-probe'
try {
    if (-not (Test-Path -LiteralPath $sshDir)) {
        New-Item -ItemType Directory -Path $sshDir -Force | Out-Null
    }
    'x' | Set-Content -LiteralPath $cmProbe -Encoding ASCII
    Add-Result 'SshConfigDir' 'PASS' "Can write $sshDir (ControlPath)." $false
} catch {
    Add-Result 'SshConfigDir' 'FAIL' $_.Exception.Message $false
} finally {
    Remove-Item -LiteralPath $cmProbe -Force -ErrorAction SilentlyContinue
}

# --- 9. mstsc present ---
$mstsc = Join-Path $env:SystemRoot 'System32\mstsc.exe'
if (Test-Path -LiteralPath $mstsc) {
    Add-Result 'mstsc' 'PASS' "Found $mstsc" $false
} else {
    Add-Result 'mstsc' 'FAIL' 'mstsc.exe not found (RDP client missing / policy removed).' $false
}

# --- 10. Optional: start mstsc then kill (does not need admin) ---
if ($SkipMstscLaunch) {
    Add-Result 'mstscLaunch' 'SKIP' 'Skipped by -SkipMstscLaunch' $false
} elseif (-not (Test-Path -LiteralPath $mstsc)) {
    Add-Result 'mstscLaunch' 'SKIP' 'mstsc missing' $false
} else {
    try {
        $p = Start-Process -FilePath $mstsc -PassThru -WindowStyle Minimized
        Start-Sleep -Milliseconds 800
        if ($p -and -not $p.HasExited) {
            Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
            Add-Result 'mstscLaunch' 'PASS' 'Started and stopped mstsc.exe as current user.' $false
        } else {
            Add-Result 'mstscLaunch' 'WARN' 'mstsc exited immediately (policy or UI). Still may work interactively.' $false
        }
    } catch {
        Add-Result 'mstscLaunch' 'FAIL' $_.Exception.Message $false
    }
}

# --- 11. Custom protocol URL launch (optional smoke; uses cleaned key unless KeepTestKeys) ---
# Re-register briefly and try Start-Process on URL, then clean.
try {
    New-Item -Path $hkcuKey -Force | Out-Null
    New-ItemProperty -Path $hkcuKey -Name 'URL Protocol' -Value '' -PropertyType String -Force | Out-Null
    New-Item -Path $hkcuCmd -Force | Out-Null
    # Use cmd /c echo so we do not need a real helper binary
    $echoCmd = 'cmd.exe /c echo preflight-ok> "%TEMP%\semaphore-rdp-preflight-url.txt" & exit 0'
    Set-ItemProperty -Path $hkcuCmd -Name '(default)' -Value $echoCmd -Force
    $marker = Join-Path $env:TEMP 'semaphore-rdp-preflight-url.txt'
    Remove-Item -LiteralPath $marker -Force -ErrorAction SilentlyContinue
    Start-Process "${ProtocolScheme}://connect?token=preflight" -ErrorAction Stop
    Start-Sleep -Milliseconds 1200
    if (Test-Path -LiteralPath $marker) {
        Add-Result 'ProtocolURLLaunch' 'PASS' 'OS launched HKCU URL protocol handler.' $false
        Remove-Item -LiteralPath $marker -Force -ErrorAction SilentlyContinue
    } else {
        Add-Result 'ProtocolURLLaunch' 'WARN' 'URL start issued but marker file missing (browser/policy may block).' $false
    }
} catch {
    Add-Result 'ProtocolURLLaunch' 'WARN' $_.Exception.Message $false
} finally {
    if (-not $KeepTestKeys) {
        Remove-Item -Path $hkcuKey -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- Summary ---
Write-Host ''
Write-Host '=== Summary ===' -ForegroundColor Cyan
$Results | Format-Table -AutoSize Name, Status, NeedsAdminIfFail, Detail

$mustPass = @(
    'UserInstallDir',
    'HKCU_Protocol',
    'LocalPortBind',
    'SshConfigDir',
    'mstsc'
)
$blocking = $Results | Where-Object { $_.Name -in $mustPass -and $_.Status -eq 'FAIL' }
$opensshFail = $Results | Where-Object { $_.Name -eq 'OpenSSH' -and $_.Status -eq 'FAIL' }

Write-Host ''
if ($blocking) {
    Write-Host 'BLOCKING for Helper MVP (fix without admin if possible):' -ForegroundColor Red
    $blocking | ForEach-Object { Write-Host ("  - {0}: {1}" -f $_.Name, $_.Detail) -ForegroundColor Red }
} else {
    Write-Host 'Core no-admin checks PASSED (install dir, HKCU protocol, localhost bind, .ssh, mstsc).' -ForegroundColor Green
}

if ($opensshFail) {
    Write-Host ''
    Write-Host 'OpenSSH Client missing: enabling the Windows optional feature often needs admin ONCE.' -ForegroundColor Yellow
    Write-Host '  Settings → Apps → Optional features → Add → OpenSSH Client' -ForegroundColor Yellow
    Write-Host '  Or (admin): Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0' -ForegroundColor Yellow
}

Write-Host ''
Write-Host 'Expected without admin: ProgramFilesWrite + HKLM_Protocol = PASS (meaning those admin paths are blocked).' -ForegroundColor DarkGray
Write-Host 'Helper design: AppData + HKCU only; never Program Files / HKLM.' -ForegroundColor DarkGray
Write-Host ''

if ($blocking -or $opensshFail) { exit 1 }
exit 0
