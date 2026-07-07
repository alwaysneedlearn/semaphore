# Run LAND GUI installer worker in logged-in desktop user's interactive session.
# Writes a JSON config file (UTF-8) and schedules worker with a single -ConfigFileArg.

$profileUserHint = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUserHint) { $profileUserHint = '' }
$profileUserHint = $profileUserHint.Trim()

$installerPath = [string]$env:LAND_INSTALLER_EXE_PATH
if ($null -eq $installerPath) { $installerPath = '' }
$installerPath = $installerPath.Trim()

function Get-EnvOrDefault {
  param([string]$Name, [string]$Default)
  $v = [string][Environment]::GetEnvironmentVariable($Name)
  if ($null -eq $v) { return $Default }
  $v = $v.Trim()
  if ($v.Length -eq 0) { return $Default }
  return $v
}

$dlg1Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_TITLE' -Default 'LHBTS 安装'
$dlg1Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_BUTTON' -Default '升级'
$dlg2Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_TITLE' -Default '提示'
$dlg2Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_BUTTON' -Default '确定'
$dlg3Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_TITLE' -Default 'LHBTS 安装'
$dlg3Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_BUTTON' -Default '确定'
$stepTimeout = Get-EnvOrDefault -Name 'LAND_INSTALL_STEP_TIMEOUT_SECONDS' -Default '90'
$dlg1CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_COORD_X_PCT' -Default '32'
$dlg1CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_COORD_Y_PCT' -Default '93'
# Step 2 (提示 popup): use Win32 button text match only — coord 0 disables fallback click.
$dlg2CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_X_PCT' -Default '0'
$dlg2CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_Y_PCT' -Default '0'
# Step 3 (single 确定): same LHBTS 安装 window; coord 88%, 93%.
$dlg3CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_COORD_X_PCT' -Default '88'
$dlg3CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_COORD_Y_PCT' -Default '93'

[int]$timeoutSec = 180
$timeoutRaw = [string]$env:LAND_INSTALL_TASK_TIMEOUT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($timeoutRaw)) {
  [int]::TryParse($timeoutRaw, [ref]$timeoutSec) | Out-Null
}
if ($timeoutSec -lt 30) { $timeoutSec = 30 }

function Resolve-InteractiveProfileUserFromExplorer {
  param([string]$HintUser)
  $short = ($HintUser -split '\\')[-1]
  if ([string]::IsNullOrWhiteSpace($short)) { $short = $HintUser }

  $explorers = Get-CimInstance Win32_Process -Filter "Name='explorer.exe'" -ErrorAction SilentlyContinue
  foreach ($e in $explorers) {
    $o = Invoke-CimMethod -InputObject $e -MethodName GetOwner -ErrorAction SilentlyContinue
    if (-not ($o -and $o.ReturnValue -eq 0 -and $o.User)) { continue }
    $user = $o.User.Trim()
    if ($user -ine $short) { continue }
    $dom = [string]$o.Domain
    $dom = $dom.Trim()
    if ($dom.Length -gt 0 -and $dom -notmatch '^(NT AUTHORITY|Window Manager)$') {
      return "$dom\$user"
    }
    return ".\$user"
  }

  if ($HintUser -match '\\') { return $HintUser }
  if (-not [string]::IsNullOrWhiteSpace($short)) { return ".\$short" }
  return ''
}

function Test-ProfileUserInteractiveSession {
  param([string]$ShortName)
  $ctx = Get-InteractiveExplorerContext -ShortName $ShortName
  if (-not $ctx.ok) {
    return @{ ok = $false; explorer_pid = 0; session_id = 0; session_state = '' }
  }
  return @{
    ok = $true
    explorer_pid = $ctx.explorer_pid
    session_id = $ctx.session_id
    session_state = $ctx.session_state
  }
}

function Get-InteractiveExplorerContext {
  param([string]$ShortName)
  $explorers = Get-CimInstance Win32_Process -Filter "Name='explorer.exe'" -ErrorAction SilentlyContinue
  foreach ($e in $explorers) {
    $o = Invoke-CimMethod -InputObject $e -MethodName GetOwner -ErrorAction SilentlyContinue
    if (-not ($o -and $o.ReturnValue -eq 0 -and $o.User -and ($o.User.Trim() -ieq $ShortName))) { continue }
    $sessionId = 0
    try {
      $sessionId = (Get-Process -Id $e.ProcessId -ErrorAction Stop).SessionId
    } catch {
      continue
    }
    $state = Get-UserSessionState -ShortName $ShortName -SessionId $sessionId
    return @{
      ok = $true
      explorer_pid = [int]$e.ProcessId
      session_id = [int]$sessionId
      session_state = $state
      domain = [string]$o.Domain
      user = [string]$o.User
    }
  }
  return @{ ok = $false; explorer_pid = 0; session_id = 0; session_state = '' }
}

function Get-UserSessionState {
  param([string]$ShortName, [int]$SessionId)
  try {
    $raw = @(query user 2>$null)
  } catch {
    return 'unknown'
  }
  foreach ($line in $raw) {
    if ($line -notmatch [regex]::Escape($ShortName)) { continue }
    if ($line -match '>') { return 'Active' }
    if ($line -match '(?i)\bActive\b|活动') { return 'Active' }
    if ($line -match '(?i)\bDisc\b|断开') { return 'Disconnected' }
  }
  if ($SessionId -gt 0) { return "session_$SessionId" }
  return 'unknown'
}

function Write-InteractiveSessionDiagnostics {
  param([string]$ProfileUser, [hashtable]$Session)
  $winrmUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
  Write-Output "INSTALL_SESSION_DIAG|winrm_user=$winrmUser|profile_user=$ProfileUser|explorer_pid=$($Session.explorer_pid)|session_id=$($Session.session_id)|session_state=$($Session.session_state)"
  try {
    $raw = @(query user 2>$null) | Where-Object { $_ -match '\S' }
    if ($raw.Count -gt 0) {
      Write-Output ("INSTALL_QUERY_USER|" + (($raw | Select-Object -First 6) -join ' || '))
    }
  } catch {
    Write-Output "INSTALL_QUERY_USER|error=$($_.Exception.Message)"
  }
}

function Remove-StaleLandGuiInstallTasks {
  try {
  $tasks = @(Get-ScheduledTask -ErrorAction SilentlyContinue | Where-Object { $_.TaskName -like 'LandGuiInstall-*' })
  foreach ($t in $tasks) {
    try {
      if ($t.State -eq 'Running') {
        Stop-ScheduledTask -TaskName $t.TaskName -ErrorAction SilentlyContinue
      }
    } catch { }
    Unregister-ScheduledTask -TaskName $t.TaskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  }
  if ($tasks.Count -gt 0) {
    Write-Output "INSTALL_TASK_CLEANUP|removed=$($tasks.Count)"
  }
  } catch {
    Write-Output "INSTALL_TASK_CLEANUP|error=$($_.Exception.Message)"
  }
}

function Get-LandGuiWorkerCommandLine {
  param([string]$ConfigPath)
  $helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
  return "-NoProfile -ExecutionPolicy Bypass -File `"$helper`" -ConfigFileArg `"$ConfigPath`""
}

function Start-LandGuiWorkerInUserSession {
  param(
    [int]$SessionId,
    [string]$ConfigPath
  )
  if (-not ([System.Management.Automation.PSTypeName]'LandGuiUserSessionLaunch').Type) {
    try {
      Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class LandGuiUserSessionLaunch {
    [DllImport("wtsapi32.dll", SetLastError = true)]
    public static extern bool WTSQueryUserToken(uint SessionId, out IntPtr phUserToken);
    [DllImport("advapi32.dll", SetLastError = true)]
    public static extern bool DuplicateTokenEx(IntPtr hExistingToken, uint dwDesiredAccess, IntPtr lpTokenAttributes, int impersonationLevel, int tokenType, out IntPtr phNewToken);
    [DllImport("advapi32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
    public static extern bool CreateProcessWithTokenW(IntPtr hToken, uint dwLogonFlags, string lpApplicationName, string lpCommandLine, uint dwCreationFlags, IntPtr lpEnvironment, string lpCurrentDirectory, ref STARTUPINFO lpStartupInfo, out PROCESS_INFORMATION lpProcessInformation);
    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool CloseHandle(IntPtr hObject);
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    public struct STARTUPINFO {
        public int cb; public string lpReserved; public string lpDesktop; public string lpTitle;
        public int dwX; public int dwY; public int dwXSize; public int dwYSize;
        public int dwXCountChars; public int dwYCountChars; public int dwFillAttribute; public int dwFlags;
        public short wShowWindow; public short cbReserved2;
        public IntPtr lpReserved2; public IntPtr hStdInput; public IntPtr hStdOutput; public IntPtr hStdError;
    }
    [StructLayout(LayoutKind.Sequential)]
    public struct PROCESS_INFORMATION {
        public IntPtr hProcess; public IntPtr hThread; public int dwProcessId; public int dwThreadId;
    }
    public const uint MAXIMUM_ALLOWED = 0x02000000;
    public const int SecurityImpersonation = 2;
    public const int TokenPrimary = 1;
    public const uint CREATE_UNICODE_ENVIRONMENT = 0x00000400;
    public const int STARTF_USESHOWWINDOW = 0x00000001;
    public const short SW_SHOW = 5;
    public static string Start(uint sessionId, string commandLine) {
        IntPtr userToken = IntPtr.Zero;
        IntPtr dupToken = IntPtr.Zero;
        try {
            if (!WTSQueryUserToken(sessionId, out userToken)) {
                return "wts_query_user_token_failed|err=" + Marshal.GetLastWin32Error();
            }
            if (!DuplicateTokenEx(userToken, MAXIMUM_ALLOWED, IntPtr.Zero, SecurityImpersonation, TokenPrimary, out dupToken)) {
                return "duplicate_token_failed|err=" + Marshal.GetLastWin32Error();
            }
            STARTUPINFO si = new STARTUPINFO();
            si.cb = Marshal.SizeOf(typeof(STARTUPINFO));
            si.lpDesktop = "winsta0\\default";
            si.dwFlags = STARTF_USESHOWWINDOW;
            si.wShowWindow = SW_SHOW;
            PROCESS_INFORMATION pi;
            if (!CreateProcessWithTokenW(dupToken, 0, null, commandLine, CREATE_UNICODE_ENVIRONMENT, IntPtr.Zero, null, ref si, out pi)) {
                return "create_process_failed|err=" + Marshal.GetLastWin32Error();
            }
            if (pi.hProcess != IntPtr.Zero) { CloseHandle(pi.hProcess); }
            if (pi.hThread != IntPtr.Zero) { CloseHandle(pi.hThread); }
            return "pid=" + pi.dwProcessId;
        } finally {
            if (userToken != IntPtr.Zero) { CloseHandle(userToken); }
            if (dupToken != IntPtr.Zero) { CloseHandle(dupToken); }
        }
    }
}
"@ -ErrorAction Stop
    } catch {
      return @{ ok = $false; error = "add_type_failed|$($_.Exception.Message)" }
    }
  }
  $helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
  if (-not (Test-Path -LiteralPath $helper)) {
    return @{ ok = $false; error = 'helper_missing' }
  }
  $psArgs = Get-LandGuiWorkerCommandLine -ConfigPath $ConfigPath
  $commandLine = "powershell.exe $psArgs"
  try {
    $detail = [LandGuiUserSessionLaunch]::Start([uint32]$SessionId, $commandLine)
    Write-Output "INTERACTIVE_USER_SESSION_LAUNCH|session_id=$SessionId|$detail"
    if ($detail -match '^pid=') {
      return @{ ok = $true; error = '' }
    }
    return @{ ok = $false; error = $detail }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

function Start-LandGuiInstallViaSchTasks {
  param(
    [string]$TaskName,
    [string]$ProfileUser,
    [string]$ConfigPath,
    [string]$RunLevel
  )
  $helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
  if (-not (Test-Path -LiteralPath $helper)) {
    return @{ ok = $false; error = 'helper_missing' }
  }
  $psArgs = Get-LandGuiWorkerCommandLine -ConfigPath $ConfigPath
  $tr = "powershell.exe $psArgs"
  $st = (Get-Date).AddSeconds(5).ToString('HH:mm')
  $sd = (Get-Date).ToString('MM/dd/yyyy')
  $rl = if ($RunLevel -eq 'Highest') { 'HIGHEST' } else { 'LIMITED' }
  try {
    $createArgs = @(
      '/Create', '/F', '/TN', $TaskName,
      '/TR', $tr,
      '/SC', 'ONCE', '/ST', $st, '/SD', $sd,
      '/RU', $ProfileUser, '/IT', '/RL', $rl
    )
    $create = Start-Process -FilePath 'schtasks.exe' -ArgumentList $createArgs -Wait -PassThru -NoNewWindow
    if ($create.ExitCode -ne 0) {
      return @{ ok = $false; error = "schtasks_create_exit=$($create.ExitCode)" }
    }
    $run = Start-Process -FilePath 'schtasks.exe' -ArgumentList @('/Run', '/TN', $TaskName) -Wait -PassThru -NoNewWindow
    Write-Output "INTERACTIVE_SCHTASKS|name=$TaskName|user=$ProfileUser|run_level=$rl|run_exit=$($run.ExitCode)"
    return @{ ok = $true; error = '' }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

function Test-LandInstallerExeRunning {
  param([string]$LiteralPath)
  if ([string]::IsNullOrWhiteSpace($LiteralPath)) { return $false }
  $name = [System.IO.Path]::GetFileName($LiteralPath)
  $cim = @(Get-CimInstance Win32_Process -Filter "Name='$name'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $LiteralPath) })
  return $cim.Count -gt 0
}

function Read-LandGuiInstallWorkerLock {
  param([string]$Path = 'C:\Windows\Temp\sem_land_gui_install_worker.lock')
  if (-not (Test-Path -LiteralPath $Path)) { return $null }
  try {
    $raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8 -ErrorAction Stop
  } catch {
    return $null
  }
  $info = @{ pid = 0; log = ''; config = ''; started = '' }
  if ($raw -match 'pid=(\d+)') { $info.pid = [int]$Matches[1] }
  if ($raw -match 'log=([^\r\n|]+)') { $info.log = [string]$Matches[1] }
  if ($raw -match 'config=([^\r\n|]+)') { $info.config = [string]$Matches[1] }
  if ($raw -match 'started=([^\r\n|]+)') { $info.started = [string]$Matches[1] }
  return $info
}

function Write-LandGuiInstallWorkerLock {
  param([int]$Pid, [string]$LogPath, [string]$ConfigPath)
  $line = "pid=$Pid|log=$LogPath|config=$ConfigPath|started=$((Get-Date).ToString('o'))"
  try {
    $line | Out-File -LiteralPath 'C:\Windows\Temp\sem_land_gui_install_worker.lock' -Encoding UTF8 -Force -ErrorAction Stop
  } catch { }
}

function Remove-LandGuiInstallWorkerLock {
  try {
    Remove-Item -LiteralPath 'C:\Windows\Temp\sem_land_gui_install_worker.lock' -Force -ErrorAction Stop
  } catch { }
}

function Wait-LandGuiInstallFromExistingWorker {
  param([string]$LogPath, [int]$TimeoutSec)
  if ([string]::IsNullOrWhiteSpace($LogPath)) { return $false }
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  $lastLineCount = 0
  do {
    Write-InstallLogTail -Path $LogPath -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $LogPath) {
      return $true
    }
    [System.Threading.Thread]::Sleep(200)
  } while ((Get-Date) -lt $deadline)
  return $false
}

function Get-InstallLogTerminalStatus {
  param([string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) { return 'pending' }
  try {
    $lines = @(Get-Content -LiteralPath $Path -ErrorAction Stop)
  } catch {
    return 'pending'
  }
  for ($i = $lines.Count - 1; $i -ge 0; $i--) {
    $t = ([string]$lines[$i]).Trim()
    if ($t -match '\bINSTALL_COMPLETE\b') { return 'complete' }
    if ($t -match '\bINSTALL_FAILED\b') { return 'failed' }
    if ($t -match '\bINSTALL_ERROR\b') { return 'failed' }
  }
  return 'pending'
}

function Test-InstallLogTerminalLine {
  param([string]$Path)
  return (Get-InstallLogTerminalStatus -Path $Path) -ne 'pending'
}

function Test-InstallWorkerStarted {
  param([string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) { return $false }
  try {
    $text = Get-Content -LiteralPath $Path -Raw -ErrorAction Stop
  } catch {
    return $false
  }
  return ($text -match 'INSTALL_WORKER_BOOT\b')
}

function Write-InstallLogTail {
  param([string]$Path, [ref]$LastLineCount)
  if (-not (Test-Path -LiteralPath $Path)) { return }
  try {
    $lines = @(Get-Content -LiteralPath $Path -ErrorAction Stop)
  } catch {
    return
  }
  if ($lines.Count -gt $LastLineCount.Value) {
    for ($i = $LastLineCount.Value; $i -lt $lines.Count; $i++) {
      Write-Output $lines[$i]
    }
    $LastLineCount.Value = $lines.Count
  }
}

function Write-InstallBootstrapLog {
  param([string]$Path, [string]$Line)
  $ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
  $out = "[$ts] $Line"
  try {
    if (Test-Path -LiteralPath $Path) {
      Add-Content -LiteralPath $Path -Value $out -Encoding UTF8 -ErrorAction Stop
    } else {
      $out | Out-File -LiteralPath $Path -Encoding UTF8 -Force -ErrorAction Stop
    }
  } catch {
    Write-Output "INSTALL_LOG_WRITE_FAILED|path=$Path|msg=$($_.Exception.Message)"
  }
}

function Start-LandGuiInstallScheduledTask {
  param(
    [string]$TaskName,
    [string]$ProfileUser,
    [string]$ConfigPath,
    [string]$LogPath,
    [string]$RunLevel
  )
  $helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
  if (-not (Test-Path -LiteralPath $helper)) {
    return @{ ok = $false; error = 'helper_missing'; helper = $helper }
  }

  $psArgs = Get-LandGuiWorkerCommandLine -ConfigPath $ConfigPath
  try {
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15) -MultipleInstances Queue
    $principal = New-ScheduledTaskPrincipal -UserId $ProfileUser -LogonType Interactive -RunLevel $RunLevel
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
    Start-Sleep -Seconds 2
    Write-Output "INTERACTIVE_INSTALL_TASK|name=$TaskName|user=$ProfileUser|run_level=$RunLevel|config=$ConfigPath|log=$LogPath"
    return @{ ok = $true; error = '' }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

if ([string]::IsNullOrWhiteSpace($profileUserHint)) {
  Write-Output 'INTERACTIVE_SESSION_REQUIRED|reason=profile_user_empty'
  Write-Output 'INSTALL_ERROR|reason=profile_user_empty'
  exit 1
}

if ([string]::IsNullOrWhiteSpace($installerPath)) {
  Write-Output 'INSTALL_ERROR|reason=installer_path_empty_in_parent_env'
  exit 1
}

if (-not (Test-Path -LiteralPath $installerPath)) {
  Write-Output "INSTALL_ERROR|reason=installer_not_found_before_task|path=$installerPath"
  exit 1
}

$profileUser = Resolve-InteractiveProfileUserFromExplorer -HintUser $profileUserHint
$profileShort = ($profileUser -split '\\')[-1]
$session = Test-ProfileUserInteractiveSession -ShortName $profileShort
if (-not $session.ok) {
  Write-Output "INTERACTIVE_SESSION_REQUIRED|user=$profileUser|hint=$profileUserHint|reason=no_explorer_for_user"
  Write-Output 'INSTALL_ERROR|reason=no_interactive_session'
  exit 1
}
Write-InteractiveSessionDiagnostics -ProfileUser $profileUser -Session $session
Remove-StaleLandGuiInstallTasks

$ts = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$configPath = "C:\Windows\Temp\sem_land_install_cfg_$ts.json"
$outFile = "C:\Windows\Temp\sem_land_install_$ts.log"
$taskName = "LandGuiInstall-$ts"

$config = [ordered]@{
  installer_path = $installerPath
  log_file = $outFile
  dlg1_title = $dlg1Title
  dlg1_button = $dlg1Button
  dlg2_title = $dlg2Title
  dlg2_button = $dlg2Button
  dlg3_title = $dlg3Title
  dlg3_button = $dlg3Button
  step_timeout_seconds = $stepTimeout
  dlg1_coord_x_pct = $dlg1CoordX
  dlg1_coord_y_pct = $dlg1CoordY
  dlg2_coord_x_pct = $dlg2CoordX
  dlg2_coord_y_pct = $dlg2CoordY
  dlg3_coord_x_pct = $dlg3CoordX
  dlg3_coord_y_pct = $dlg3CoordY
}
try {
  ($config | ConvertTo-Json -Compress) | Out-File -LiteralPath $configPath -Encoding UTF8 -Force
} catch {
  Write-Output "INSTALL_ERROR|reason=config_write_failed|msg=$($_.Exception.Message)"
  exit 1
}

Write-InstallBootstrapLog -Path $outFile -Line "INSTALL_SCHEDULED|task=$taskName|user=$profileUser|installer=$installerPath"

$existingLock = Read-LandGuiInstallWorkerLock
if ($existingLock -and $existingLock.pid -gt 0) {
  $lockProc = Get-Process -Id $existingLock.pid -ErrorAction SilentlyContinue
  if ($lockProc -and -not $lockProc.HasExited) {
    Write-Output "INSTALL_ALREADY_RUNNING|pid=$($existingLock.pid)|log=$($existingLock.log)|action=wait_existing_worker"
    $outFile = if ($existingLock.log) { $existingLock.log } else { $outFile }
    if (Wait-LandGuiInstallFromExistingWorker -LogPath $outFile -TimeoutSec $timeoutSec) {
      if ((Get-InstallLogTerminalStatus -Path $outFile) -eq 'complete') { exit 0 }
    }
    Write-Output 'INSTALL_ERROR|reason=existing_worker_no_complete'
    exit 1
  }
  Remove-LandGuiInstallWorkerLock
}

if (Test-LandInstallerExeRunning -LiteralPath $installerPath) {
  Write-Output "INSTALL_INSTALLER_ALREADY_RUNNING|path=$installerPath|action=single_worker_only"
}

$started = $false
$lastStartError = ''
$script:usedScheduledTaskName = ''
$launchModes = @(
  @{ mode = 'scheduled_highest'; run_level = 'Highest' },
  @{ mode = 'schtasks_highest'; run_level = 'Highest' }
)
foreach ($launch in $launchModes) {
  if ($started) { break }
  if ($launch.mode -ne 'scheduled_highest') {
    if (Test-InstallWorkerStarted -Path $outFile) {
      $started = $true
      break
    }
    if (Test-LandInstallerExeRunning -LiteralPath $installerPath) {
      Write-Output "INSTALL_SKIP_LAUNCH|mode=$($launch.mode)|reason=installer_already_running"
      break
    }
  }
  $mode = [string]$launch.mode
  $runLevel = [string]$launch.run_level
  if ($mode -like 'schtasks_*') {
    $result = Start-LandGuiInstallViaSchTasks -TaskName $taskName -ProfileUser $profileUser -ConfigPath $configPath -RunLevel $runLevel
  } else {
    $result = Start-LandGuiInstallScheduledTask -TaskName $taskName -ProfileUser $profileUser -ConfigPath $configPath -LogPath $outFile -RunLevel $runLevel
  }
  if (-not $result.ok) {
    $lastStartError = [string]$result.error
    Write-Output "INSTALL_TASK_START_FAILED|mode=$mode|run_level=$runLevel|error=$lastStartError"
    if ($mode -like 'scheduled_*' -or $mode -like 'schtasks_*') {
      Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
    }
    continue
  }
  $earlyLineCount = 0
  $bootDeadline = (Get-Date).AddSeconds(60)
  do {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$earlyLineCount)
    if (Test-InstallWorkerStarted -Path $outFile) {
      $started = $true
      Write-Output "INSTALL_WORKER_BOOT_OK|mode=$mode|user=$profileUser"
      if ($mode -like 'scheduled_*' -or $mode -like 'schtasks_*') {
        $script:usedScheduledTaskName = $taskName
      }
      break
    }
    if ($mode -like 'scheduled_*' -or $mode -like 'schtasks_*') {
      $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
      if ($taskInfo) {
        $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
        if ($lastHex -eq '800710E0' -and -not (Test-InstallWorkerStarted -Path $outFile)) {
          Write-Output "INSTALL_TASK_EARLY_STATE|mode=$mode|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)"
          Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
          break
        }
      }
    }
    if (Test-LandInstallerExeRunning -LiteralPath $installerPath) {
      Write-Output "INSTALL_INSTALLER_STARTED_DURING_BOOT|path=$installerPath|action=stop_extra_launch_modes"
      break
    }
    [System.Threading.Thread]::Sleep(200)
  } while ((Get-Date) -lt $bootDeadline)
  if (-not $started -and ($mode -like 'scheduled_*' -or $mode -like 'schtasks_*')) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  }
}

if (-not $started) {
  Write-Output "INSTALL_ERROR|reason=scheduled_task_never_started_worker|last_error=$lastStartError|user=$profileUser|hint=manual_ok_means_rdp_interactive_direct_worker|semaphore_needs_interactive_session"
  if (-not (Test-Path -LiteralPath $outFile)) {
    Write-Output "INSTALL_TASK_NO_OUTPUT|task=$taskName|timeout=0s|log=$outFile|hint=worker_never_booted"
  }
  exit 1
}

try {
  $deadline = (Get-Date).AddSeconds($timeoutSec)
  $logComplete = $false
  $lastLineCount = 0
  $taskStateLogged = $false
  do {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $outFile) {
      $logComplete = $true
      break
    }
    $task = $null
    if (-not [string]::IsNullOrWhiteSpace($script:usedScheduledTaskName)) {
      $task = Get-ScheduledTask -TaskName $script:usedScheduledTaskName -ErrorAction SilentlyContinue
    }
    if ($task -and $task.State -ne 'Running') {
      Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
      if (-not $taskStateLogged) {
        $taskStateLogged = $true
        $taskInfo = Get-ScheduledTaskInfo -TaskName $script:usedScheduledTaskName -ErrorAction SilentlyContinue
        if ($taskInfo) {
          $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
          Write-Output "INSTALL_TASK_STATE|name=$($script:usedScheduledTaskName)|state=$($task.State)|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)"
          if ($lastHex -eq '800710E0') {
            Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
          }
          if ($lastHex -eq 'C000013A') {
            Write-Output 'INSTALL_TASK_HINT|code=0xC000013A|meaning=worker进程异常退出(脚本错误或原生调用崩溃)'
          }
          if (($lastHex -eq 'C000013A') -and -not (Test-InstallLogTerminalLine -Path $outFile)) {
            break
          }
        }
      }
      if (Test-InstallLogTerminalLine -Path $outFile) {
        $logComplete = $true
        break
      }
    }
    [System.Threading.Thread]::Sleep(0)
  } while ((Get-Date) -lt $deadline)

  if (-not $logComplete -and (Test-Path -LiteralPath $outFile)) {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $outFile) { $logComplete = $true }
  }

  if (Test-Path -LiteralPath $outFile) {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (-not $logComplete) {
      Write-Output "INSTALL_TASK_LOG_INCOMPLETE|task=$taskName|log=$outFile|hint=check_DIALOG_POLL_or_INTERACTIVE_SESSION"
    }
  } else {
    Write-Output "INSTALL_TASK_NO_OUTPUT|task=$taskName|timeout=${timeoutSec}s|log=$outFile|hint=scheduled_task_never_wrote_log"
    Write-Output 'INSTALL_ERROR|reason=no_task_log'
  }
} catch {
  Write-Output "INSTALL_TASK_ERROR:$($_.Exception.Message)"
  Write-Output 'INSTALL_ERROR|reason=task_failed'
  exit 1
} finally {
  if (-not [string]::IsNullOrWhiteSpace($script:usedScheduledTaskName)) {
    Unregister-ScheduledTask -TaskName $script:usedScheduledTaskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  }
  Remove-LandGuiInstallWorkerLock
  Remove-Item -LiteralPath $configPath -Force -ErrorAction SilentlyContinue
}

$terminalStatus = Get-InstallLogTerminalStatus -Path $outFile
if ($terminalStatus -eq 'complete') {
  exit 0
}
exit 1
