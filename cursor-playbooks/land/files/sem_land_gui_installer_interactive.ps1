# Run LAND GUI installer worker in logged-in desktop user's interactive session.
# Launch: scheduled task (same pattern as sem_stop_close_main_window_confirm_interactive.ps1).
# Temp files under C:\Windows\Temp\ (same dir as deployed sem_*.ps1).
# INSTALL_SCRIPT_REV bumps when launch logic changes — must appear in task stdout.

Write-Output 'INSTALL_SCRIPT_REV=20260707-registry-version-v8'

$SemLandTempDir = 'C:\Windows\Temp'
if (-not [string]::IsNullOrWhiteSpace($PSScriptRoot) -and (Test-Path -LiteralPath $PSScriptRoot)) {
  $SemLandTempDir = $PSScriptRoot
}

$profileUserHint = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUserHint) { $profileUserHint = '' }
$profileUserHint = $profileUserHint.Trim()

$installerPath = [string]$env:LAND_INSTALLER_EXE_PATH
if ($null -eq $installerPath) { $installerPath = '' }
$installerPath = $installerPath.Trim()

function Resolve-LandExpectedInstallVersionFromInstaller {
  param([string]$InstallerPath)
  $fromEnv = Get-EnvOrDefault -Name 'LAND_INSTALL_EXPECTED_VERSION' -Default ''
  if (-not [string]::IsNullOrWhiteSpace($fromEnv)) { return $fromEnv.Trim() }
  if ([string]::IsNullOrWhiteSpace($InstallerPath)) { return '' }
  $name = [System.IO.Path]::GetFileName($InstallerPath)
  if ($name -match '(?i)LHBTS[_-]?Setup[_-]?(\d+\.\d+\.\d+\.\d+)') {
    return $Matches[1]
  }
  if ($name -match '(\d+\.\d+\.\d+\.\d+)') {
    return $Matches[1]
  }
  return ''
}

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
$dlg2CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_X_PCT' -Default '0'
$dlg2CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_Y_PCT' -Default '0'
$dlg3CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_COORD_X_PCT' -Default '88'
$dlg3CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_COORD_Y_PCT' -Default '93'

[int]$timeoutSec = 300
$timeoutRaw = [string]$env:LAND_INSTALL_TASK_TIMEOUT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($timeoutRaw)) {
  [int]::TryParse($timeoutRaw, [ref]$timeoutSec) | Out-Null
}
if ($timeoutSec -lt 60) { $timeoutSec = 60 }

$script:usedScheduledTaskName = ''

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
    return @{
      ok = $true
      explorer_pid = [int]$e.ProcessId
      session_id = [int]$sessionId
    }
  }
  return @{ ok = $false; explorer_pid = 0; session_id = 0 }
}

function Test-ProfileUserInteractiveSession {
  param([string]$ShortName)
  $ctx = Get-InteractiveExplorerContext -ShortName $ShortName
  if (-not $ctx.ok) {
    return @{ ok = $false; explorer_pid = 0; session_id = 0 }
  }
  return @{
    ok = $true
    explorer_pid = $ctx.explorer_pid
    session_id = $ctx.session_id
  }
}

function Write-InteractiveSessionDiagnostics {
  param([string]$ProfileUser, [hashtable]$Session)
  $winrmUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
  Write-Output "INSTALL_SESSION_DIAG|winrm_user=$winrmUser|profile_user=$ProfileUser|explorer_pid=$($Session.explorer_pid)|session_id=$($Session.session_id)"
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

function Get-LandGuiSchTasksTrArgument {
  param([string]$BatPath)
  $run = "cmd.exe /c `"$BatPath`""
  $escaped = $run.Replace('"', '\"')
  return "`"$escaped`""
}

function Write-LandInstallBatFile {
  param(
    [string]$Path,
    [string]$Content
  )
  $normalized = ($Content -replace "`r?`n", "`r`n").TrimEnd() + "`r`n"
  [System.IO.File]::WriteAllText($Path, $normalized, [System.Text.Encoding]::ASCII)
}

function Grant-LandInstallExecuteRead {
  param([string]$Path)
  if ([string]::IsNullOrWhiteSpace($Path)) { return }
  if (-not (Test-Path -LiteralPath $Path)) { return }
  try {
    & icacls.exe $Path /grant 'Users:(RX)' /grant 'Everyone:(RX)' 2>$null | Out-Null
  } catch { }
}

function New-LandInstallTaskBat {
  param(
    [string]$Timestamp,
    [string]$WorkerPath,
    [string]$ConfigPath,
    [string]$LogPath
  )
  $batPath = Join-Path $SemLandTempDir "sem_land_install_run_$Timestamp.bat"
  $tracePath = Join-Path $SemLandTempDir 'sem_land_gui_install_trace.log'
  $psOutPath = Join-Path $SemLandTempDir "sem_land_install_ps_$Timestamp.log"
  # NOTE: bat echo lines must use ";" not "|" — cmd treats "|" as pipe and exits 255.
  $content = @"
@echo off
echo BAT_START;ts=$Timestamp>>"$tracePath"
echo BAT_CMD;worker=$WorkerPath;config=$ConfigPath;log=$LogPath>>"$tracePath"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$WorkerPath" -ConfigFileArg "$ConfigPath" -LogFileArg "$LogPath" 1>>"$psOutPath" 2>&1
set RC=%ERRORLEVEL%
(echo BAT_END;ts=$Timestamp;rc=%RC%)>>"$tracePath"
exit /b %RC%
"@
  Write-LandInstallBatFile -Path $batPath -Content $content
  Grant-LandInstallExecuteRead -Path $batPath
  return @{ bat = $batPath; ps_out = $psOutPath }
}

function Get-LandInstallTaskCmdLine {
  param([string]$BatPath)
  return "/c `"$BatPath`""
}

function Get-LandGuiWorkerPsArgs {
  param(
    [string]$ConfigPath,
    [string]$LogPath = '',
    [switch]$ConfigOnly
  )
  $helper = Join-Path $SemLandTempDir 'sem_land_gui_installer_worker.ps1'
  if ($ConfigOnly -or [string]::IsNullOrWhiteSpace($LogPath)) {
    return "-NoProfile -ExecutionPolicy Bypass -File `"$helper`" -ConfigFileArg `"$ConfigPath`""
  }
  return "-NoProfile -ExecutionPolicy Bypass -File `"$helper`" -ConfigFileArg `"$ConfigPath`" -LogFileArg `"$LogPath`""
}

function Ensure-LandGuiUserSessionLaunchType {
  if (([System.Management.Automation.PSTypeName]'LandGuiUserSessionLaunch').Type) {
    return $true
  }
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
    return $true
  } catch {
    Write-Output "INSTALL_WTS_ADD_TYPE_FAILED|msg=$($_.Exception.Message)"
    return $false
  }
}

function Start-LandGuiWorkerInUserSession {
  param(
    [int]$SessionId,
    [string]$ConfigPath,
    [string]$LogPath
  )
  if (-not (Ensure-LandGuiUserSessionLaunchType)) {
    return @{ ok = $false; error = 'wts_add_type_failed' }
  }
  $helper = Join-Path $SemLandTempDir 'sem_land_gui_installer_worker.ps1'
  if (-not (Test-Path -LiteralPath $helper)) {
    return @{ ok = $false; error = "helper_missing|$helper" }
  }
  $psArgs = Get-LandGuiWorkerPsArgs -ConfigPath $ConfigPath -LogPath $LogPath
  $commandLine = "powershell.exe $psArgs"
  Write-Output "INSTALL_WTS_CMD|$commandLine"
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
    [string]$BatPath,
    [string]$RunLevel
  )
  $tr = Get-LandGuiSchTasksTrArgument -BatPath $BatPath
  Write-Output "INSTALL_SCHTASKS_TR|$tr"
  $st = (Get-Date).AddSeconds(5).ToString('HH:mm')
  $sd = (Get-Date).ToString('yyyy/MM/dd')
  $rl = if ($RunLevel -eq 'Highest') { 'HIGHEST' } else { 'LIMITED' }
  try {
    $stderrFile = [System.IO.Path]::GetTempFileName()
    $createArgs = @(
      '/Create', '/F', '/TN', $TaskName,
      '/TR', $tr,
      '/SC', 'ONCE', '/ST', $st, '/SD', $sd,
      '/RU', $ProfileUser, '/IT', '/RL', $rl
    )
    $create = Start-Process -FilePath 'schtasks.exe' -ArgumentList $createArgs -Wait -PassThru -NoNewWindow -RedirectStandardError $stderrFile
    $stderrText = ''
    if (Test-Path -LiteralPath $stderrFile) {
      $stderrText = (Get-Content -LiteralPath $stderrFile -Raw -ErrorAction SilentlyContinue)
      Remove-Item -LiteralPath $stderrFile -Force -ErrorAction SilentlyContinue
    }
    if ($create.ExitCode -ne 0) {
      $errDetail = if ($stderrText) { $stderrText.Trim() } else { '' }
      return @{ ok = $false; error = "schtasks_create_exit=$($create.ExitCode)|stderr=$errDetail" }
    }
    $runOutFile = [System.IO.Path]::GetTempFileName()
    $runErrFile = [System.IO.Path]::GetTempFileName()
    $run = Start-Process -FilePath 'schtasks.exe' -ArgumentList @('/Run', '/TN', $TaskName) -Wait -PassThru -NoNewWindow -RedirectStandardOutput $runOutFile -RedirectStandardError $runErrFile
    Remove-Item -LiteralPath $runOutFile -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $runErrFile -Force -ErrorAction SilentlyContinue
    Write-Output "INTERACTIVE_SCHTASKS|name=$TaskName|user=$ProfileUser|run_level=$rl|run_exit=$($run.ExitCode)|bat=$BatPath"
    return @{ ok = $true; error = '' }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

function Start-LandGuiInstallScheduledPowerShell {
  param(
    [string]$TaskName,
    [string]$ProfileUser,
    [string]$WorkerPath,
    [string]$ConfigPath,
    [string]$LogPath,
    [string]$RunLevel
  )
  if (-not (Test-Path -LiteralPath $WorkerPath)) {
    return @{ ok = $false; error = "helper_missing|$WorkerPath" }
  }
  $psArgList = @(
    '-NoProfile',
    '-ExecutionPolicy', 'Bypass',
    '-File', "`"$WorkerPath`"",
    '-ConfigFileArg', "`"$ConfigPath`"",
    '-LogFileArg', "`"$LogPath`""
  )
  $psArgs = $psArgList -join ' '
  Write-Output "INSTALL_TASK_CMD|powershell.exe $psArgs"
  try {
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15)
    $principal = New-ScheduledTaskPrincipal -UserId $ProfileUser -LogonType Interactive -RunLevel $RunLevel
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
    Start-Sleep -Seconds 2
    Write-Output "INTERACTIVE_INSTALL_TASK|name=$TaskName|user=$ProfileUser|run_level=$RunLevel|launcher=powershell|config=$ConfigPath|log=$LogPath"
    return @{ ok = $true; error = '' }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

function Start-LandGuiInstallScheduledTask {
  param(
    [string]$TaskName,
    [string]$ProfileUser,
    [string]$BatPath,
    [string]$ConfigPath,
    [string]$LogPath,
    [string]$RunLevel
  )
  if (-not (Test-Path -LiteralPath $BatPath)) {
    return @{ ok = $false; error = "bat_missing|$BatPath" }
  }
  $taskCmd = Get-LandInstallTaskCmdLine -BatPath $BatPath
  Write-Output "INSTALL_TASK_CMD|cmd.exe $taskCmd"
  Write-Output "INSTALL_TASK_BAT|path=$BatPath"
  try {
    $action = New-ScheduledTaskAction -Execute 'cmd.exe' -Argument $taskCmd
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15)
    $principal = New-ScheduledTaskPrincipal -UserId $ProfileUser -LogonType Interactive -RunLevel $RunLevel
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
    Start-Sleep -Seconds 2
    Write-Output "INTERACTIVE_INSTALL_TASK|name=$TaskName|user=$ProfileUser|run_level=$RunLevel|launcher=cmd|config=$ConfigPath|log=$LogPath|bat=$BatPath"
    return @{ ok = $true; error = '' }
  } catch {
    return @{ ok = $false; error = $_.Exception.Message }
  }
}

function Test-InstallWorkerStarted {
  param([string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) { return $false }
  try {
    $text = Get-Content -LiteralPath $Path -Raw -ErrorAction Stop
  } catch {
    return $false
  }
  return ($text -match 'INSTALL_WORKER_BOOT\b|INSTALL_WORKER_START\b|INSTALL_START\b|WORKER_LINE0\b')
}

function Test-InstallLogTerminalLine {
  param([string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) { return $false }
  try {
    $text = Get-Content -LiteralPath $Path -Raw -ErrorAction Stop
  } catch {
    return $false
  }
  return ($text -match '\bINSTALL_COMPLETE\b|\bINSTALL_FAILED\b|\bINSTALL_ERROR\b')
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

function Grant-LandInstallConfigRead {
  param([string]$ConfigPath)
  if ([string]::IsNullOrWhiteSpace($ConfigPath)) { return }
  if (-not (Test-Path -LiteralPath $ConfigPath)) { return }
  try {
    & icacls.exe $ConfigPath /grant 'Users:(R)' /grant 'Everyone:(R)' 2>$null | Out-Null
  } catch { }
}

function Grant-LandInstallLogWrite {
  param([string]$LogPath)
  if ([string]::IsNullOrWhiteSpace($LogPath)) { return }
  try {
    if (Test-Path -LiteralPath $LogPath) {
      Remove-Item -LiteralPath $LogPath -Force -ErrorAction SilentlyContinue
    }
    & icacls.exe (Split-Path -Parent $LogPath) /grant 'Users:(M)' /grant 'Everyone:(M)' 2>$null | Out-Null
  } catch { }
}

function Initialize-LandInstallTraceFile {
  param([string]$TracePath, [string]$RunId)
  try {
    $line = "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] INSTALL_RUN_START|ts=$RunId|user=$([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"
    $line | Out-File -LiteralPath $TracePath -Encoding UTF8 -Force
    & icacls.exe $TracePath /grant 'Users:(M)' /grant 'Everyone:(M)' 2>$null | Out-Null
  } catch { }
}

function Remove-StaleLandGuiInstallWorkerLock {
  $lockPath = Join-Path $SemLandTempDir 'sem_land_gui_install_worker.lock'
  Remove-Item -LiteralPath $lockPath -Force -ErrorAction SilentlyContinue
}

function Write-LandInstallActiveFile {
  param(
    [string]$ConfigPath,
    [string]$LogPath
  )
  $activePath = Join-Path $SemLandTempDir 'sem_land_install_active.json'
  $active = [ordered]@{
    config_path = $ConfigPath
    log_path = $LogPath
    updated = (Get-Date).ToString('o')
  }
  try {
    ($active | ConvertTo-Json -Compress) | Out-File -LiteralPath $activePath -Encoding UTF8 -Force
    Grant-LandInstallConfigRead -ConfigPath $activePath
    Write-Output "INSTALL_ACTIVE|path=$activePath|config=$ConfigPath|log=$LogPath"
  } catch {
    Write-Output "INSTALL_ACTIVE_FAILED|msg=$($_.Exception.Message)"
  }
}

function Write-InstallTaskDiag {
  param([string]$TaskName)
  if ([string]::IsNullOrWhiteSpace($TaskName)) { return }
  try {
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    $info = Get-ScheduledTaskInfo -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($task) {
      Write-Output "INSTALL_TASK_DIAG|name=$TaskName|state=$($task.State)|actions=$($task.Actions.Count)"
      if ($task.Actions.Count -gt 0) {
        $a = $task.Actions[0]
        Write-Output "INSTALL_TASK_ACTION|execute=$($a.Execute)|arguments=$($a.Arguments)"
      }
    }
    if ($info) {
      $lastHex = '{0:X8}' -f ([uint32]($info.LastTaskResult))
      Write-Output "INSTALL_TASK_INFO|last_result=$($info.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($info.LastRunTime)|next_run=$($info.NextRunTime)"
    }
  } catch {
    Write-Output "INSTALL_TASK_DIAG|error=$($_.Exception.Message)"
  }
}

function Write-InstallTraceTail {
  param([int]$MaxLines = 8)
  $tracePath = Join-Path $SemLandTempDir 'sem_land_gui_install_trace.log'
  if (-not (Test-Path -LiteralPath $tracePath)) {
    Write-Output 'INSTALL_TRACE_TAIL|hint=trace_missing'
    return
  }
  try {
    $lines = @(Get-Content -LiteralPath $tracePath -ErrorAction Stop)
    $tail = @($lines | Select-Object -Last $MaxLines)
    Write-Output ("INSTALL_TRACE_TAIL|" + ($tail -join ' || '))
  } catch {
    Write-Output "INSTALL_TRACE_TAIL|error=$($_.Exception.Message)"
  }
}

function Test-InstallWorkerStartedViaTrace {
  param(
    [string]$ConfigPath,
    [string]$RunId
  )
  $tracePath = Join-Path $SemLandTempDir 'sem_land_gui_install_trace.log'
  if (-not (Test-Path -LiteralPath $tracePath)) { return $false }
  try {
    $text = Get-Content -LiteralPath $tracePath -Raw -ErrorAction Stop
  } catch {
    return $false
  }
  if ($text -notmatch 'WORKER_INVOKED|WORKER_ENTRY') { return $false }
  if (-not [string]::IsNullOrWhiteSpace($RunId) -and $text -notmatch "ts=$RunId|sem_land_install_cfg_$RunId|sem_land_install_$RunId") {
    if (-not [string]::IsNullOrWhiteSpace($ConfigPath) -and $text -notmatch [regex]::Escape($ConfigPath)) {
      return $false
    }
  }
  return $true
}

function Test-InstallBatEnded {
  param([string]$RunId)
  $tracePath = Join-Path $SemLandTempDir 'sem_land_gui_install_trace.log'
  if (-not (Test-Path -LiteralPath $tracePath)) { return $null }
  try {
    $text = Get-Content -LiteralPath $tracePath -Raw -ErrorAction Stop
  } catch {
    return $null
  }
  if ($text -match "BAT_END[;|]ts=$RunId[;|]rc=(\d+)") {
    return [int]$Matches[1]
  }
  return $null
}

function Write-InstallPsOutTail {
  param([string]$PsOutPath, [int]$MaxLines = 12)
  if ([string]::IsNullOrWhiteSpace($PsOutPath)) { return }
  if (-not (Test-Path -LiteralPath $PsOutPath)) {
    Write-Output "INSTALL_PS_OUT|hint=missing|path=$PsOutPath"
    return
  }
  try {
    $lines = @(Get-Content -LiteralPath $PsOutPath -ErrorAction Stop)
    $tail = @($lines | Select-Object -Last $MaxLines)
    Write-Output ("INSTALL_PS_OUT|" + ($tail -join ' || '))
  } catch {
    Write-Output "INSTALL_PS_OUT|error=$($_.Exception.Message)"
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
$configPath = Join-Path $SemLandTempDir "sem_land_install_cfg_$ts.json"
$outFile = Join-Path $SemLandTempDir "sem_land_install_$ts.log"
$taskName = "LandGuiInstall-$ts"
$expectedVersion = Resolve-LandExpectedInstallVersionFromInstaller -InstallerPath $installerPath

$config = [ordered]@{
  installer_path = $installerPath
  log_file = $outFile
  expected_version = $expectedVersion
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
  Grant-LandInstallConfigRead -ConfigPath $configPath
} catch {
  Write-Output "INSTALL_ERROR|reason=config_write_failed|msg=$($_.Exception.Message)"
  exit 1
}

Write-Output "INSTALL_SCHEDULED|task=$taskName|user=$profileUser|installer=$installerPath|config=$configPath|log=$outFile|expected_version=$expectedVersion"

Write-LandInstallActiveFile -ConfigPath $configPath -LogPath $outFile
Grant-LandInstallLogWrite -LogPath $outFile
Remove-StaleLandGuiInstallWorkerLock

$workerScript = Join-Path $SemLandTempDir 'sem_land_gui_installer_worker.ps1'
$tracePath = Join-Path $SemLandTempDir 'sem_land_gui_install_trace.log'
if (-not (Test-Path -LiteralPath $workerScript)) {
  Write-Output "INSTALL_ERROR|reason=helper_missing|path=$workerScript"
  exit 1
}
try {
  Unblock-File -LiteralPath $workerScript -ErrorAction SilentlyContinue
  & icacls.exe $workerScript /grant 'Users:(RX)' /grant 'Everyone:(RX)' 2>$null | Out-Null
  Initialize-LandInstallTraceFile -TracePath $tracePath -RunId $ts
} catch { }

try {
  $batInfo = New-LandInstallTaskBat -Timestamp $ts -WorkerPath $workerScript -ConfigPath $configPath -LogPath $outFile
  $batPath = $batInfo.bat
  $psOutPath = $batInfo.ps_out
  Grant-LandInstallExecuteRead -Path $batPath
  Grant-LandInstallLogWrite -LogPath $psOutPath
} catch {
  Write-Output "INSTALL_ERROR|reason=task_bat_write_failed|msg=$($_.Exception.Message)"
  exit 1
}

$started = $false
$lastStartError = ''
$launchModes = @(
  @{ mode = 'scheduled_ps_highest'; run_level = 'Highest'; launch = 'powershell' }
  @{ mode = 'scheduled_cmd_highest'; run_level = 'Highest'; launch = 'cmd' }
  @{ mode = 'schtasks_cmd_highest'; run_level = 'Highest'; launch = 'schtasks' }
)

foreach ($launch in $launchModes) {
  if ($started) { break }
  $mode = [string]$launch.mode
  $runLevel = [string]$launch.run_level
  $launchKind = [string]$launch.launch

  if ($launchKind -eq 'schtasks') {
    $result = Start-LandGuiInstallViaSchTasks -TaskName $taskName -ProfileUser $profileUser -BatPath $batPath -RunLevel $runLevel
  } elseif ($launchKind -eq 'powershell') {
    $result = Start-LandGuiInstallScheduledPowerShell -TaskName $taskName -ProfileUser $profileUser -WorkerPath $workerScript -ConfigPath $configPath -LogPath $outFile -RunLevel $runLevel
  } elseif ($mode -eq 'user_session') {
    if ($session.session_id -le 0) {
      Write-Output 'INSTALL_TASK_START_SKIPPED|mode=user_session|reason=no_session_id'
      continue
    }
    $result = Start-LandGuiWorkerInUserSession -SessionId $session.session_id -ConfigPath $configPath -LogPath $outFile
  } else {
    $result = Start-LandGuiInstallScheduledTask -TaskName $taskName -ProfileUser $profileUser -BatPath $batPath -ConfigPath $configPath -LogPath $outFile -RunLevel $runLevel
  }

  if (-not $result.ok) {
    $lastStartError = [string]$result.error
    Write-Output "INSTALL_TASK_START_FAILED|mode=$mode|run_level=$runLevel|error=$lastStartError"
    if ($mode -eq 'user_session' -and $lastStartError -match 'err=1314') {
      Write-Output 'INSTALL_WTS_HINT|err=1314|meaning=WinRM账户无WTSQueryUserToken权限，已改走计划任务'
    }
    if ($launchKind -ne 'schtasks' -and $mode -like 'scheduled_*') {
      Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
    }
    continue
  }

  $earlyLineCount = 0
  $bootDeadline = (Get-Date).AddSeconds(90)
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
    if (Test-InstallWorkerStartedViaTrace -ConfigPath $configPath -RunId $ts) {
      $started = $true
      Write-Output "INSTALL_WORKER_BOOT_OK|mode=$mode|via=trace|user=$profileUser"
      if ($mode -like 'scheduled_*' -or $mode -like 'schtasks_*') {
        $script:usedScheduledTaskName = $taskName
      }
      break
    }
    $batRc = Test-InstallBatEnded -RunId $ts
    if ($null -ne $batRc -and -not $started) {
      Write-Output "INSTALL_BAT_ENDED|ts=$ts|rc=$batRc"
      Write-InstallTraceTail
      Write-InstallPsOutTail -PsOutPath $psOutPath
      if ($batRc -ne 0) {
        $lastStartError = "bat_exit=$batRc|worker_never_invoked"
        break
      }
    }
    if ($launchKind -ne 'schtasks' -and $mode -like 'scheduled_*') {
      $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
      if ($taskInfo) {
        $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
        if ($lastHex -eq '800710E0') {
          Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
          break
        }
      }
    }
    Start-Sleep -Milliseconds 300
  } while ((Get-Date) -lt $bootDeadline)

  if (-not $started) {
    Write-InstallTaskDiag -TaskName $taskName
    Write-InstallTraceTail
    if ($launchKind -ne 'schtasks' -and ($mode -like 'scheduled_*')) {
      Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
    }
    if ($launchKind -eq 'schtasks') {
      schtasks.exe /Delete /TN $taskName /F 2>$null | Out-Null
    }
  }
}

if (-not $started) {
  Write-InstallTaskDiag -TaskName $taskName
  Write-InstallTraceTail
  Write-InstallPsOutTail -PsOutPath $psOutPath
  if (Test-Path -LiteralPath $outFile) {
    $tail = @(Get-Content -LiteralPath $outFile -ErrorAction SilentlyContinue | Select-Object -Last 8)
    if ($tail.Count -gt 0) {
      Write-Output ("INSTALL_TASK_LOG_TAIL|" + ($tail -join ' || '))
    }
  } else {
    Write-Output "INSTALL_TASK_NO_OUTPUT|log=$outFile|hint=worker_never_booted"
  }
  Write-Output "INSTALL_ERROR|reason=worker_never_booted|last_error=$lastStartError|user=$profileUser"
  exit 1
}

try {
  $deadline = (Get-Date).AddSeconds($timeoutSec)
  $logComplete = $false
  $lastLineCount = 0
  do {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $outFile) {
      $logComplete = $true
      break
    }
    Start-Sleep -Seconds 1
  } while ((Get-Date) -lt $deadline)

  if (Test-Path -LiteralPath $outFile) {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (-not $logComplete) {
      Write-Output "INSTALL_TASK_LOG_INCOMPLETE|log=$outFile"
      Write-InstallTraceTail
      Write-InstallPsOutTail -PsOutPath $psOutPath
      Write-Output 'INSTALL_ERROR|reason=install_timeout_or_worker_stuck'
    }
  }
} catch {
  Write-Output "INSTALL_TASK_ERROR|msg=$($_.Exception.Message)"
  Write-Output 'INSTALL_ERROR|reason=task_failed'
  exit 1
} finally {
  if (-not [string]::IsNullOrWhiteSpace($script:usedScheduledTaskName)) {
    Unregister-ScheduledTask -TaskName $script:usedScheduledTaskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
    schtasks.exe /Delete /TN $script:usedScheduledTaskName /F 2>$null | Out-Null
  }
  Remove-Item -LiteralPath $configPath -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath (Join-Path $SemLandTempDir 'sem_land_install_active.json') -Force -ErrorAction SilentlyContinue
  if ($batPath) {
    Remove-Item -LiteralPath $batPath -Force -ErrorAction SilentlyContinue
  }
  if ($psOutPath) {
    Remove-Item -LiteralPath $psOutPath -Force -ErrorAction SilentlyContinue
  }
}

if (Test-InstallLogTerminalLine -Path $outFile) {
  if ((Get-Content -LiteralPath $outFile -Raw -ErrorAction SilentlyContinue) -match '\bINSTALL_COMPLETE\b') {
    exit 0
  }
}
exit 1
