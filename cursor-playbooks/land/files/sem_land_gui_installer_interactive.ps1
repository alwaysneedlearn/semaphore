# Run LAND GUI installer worker in logged-in desktop user's interactive session.
# Pattern matches sem_stop_close_main_window_confirm_interactive.ps1 (single scheduled task).

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
  $explorers = Get-CimInstance Win32_Process -Filter "Name='explorer.exe'" -ErrorAction SilentlyContinue
  foreach ($e in $explorers) {
    $o = Invoke-CimMethod -InputObject $e -MethodName GetOwner -ErrorAction SilentlyContinue
    if ($o -and $o.ReturnValue -eq 0 -and $o.User -and ($o.User.Trim() -ieq $ShortName)) {
      return @{ ok = $true; explorer_pid = [int]$e.ProcessId }
    }
  }
  return @{ ok = $false; explorer_pid = 0 }
}

function Write-InteractiveSessionDiagnostics {
  param([string]$ProfileUser, [hashtable]$Session)
  $winrmUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
  Write-Output "INSTALL_SESSION_DIAG|winrm_user=$winrmUser|profile_user=$ProfileUser|explorer_pid=$($Session.explorer_pid)"
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
  param([string]$ConfigPath, [string]$LogPath)
  $helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
  # Match stop-script arg join: pass LogFileArg on CLI (scheduled task does not inherit parent env).
  $psArgList = @(
    '-NoProfile',
    '-ExecutionPolicy', 'Bypass',
    '-File', "`"$helper`"",
    '-ConfigFileArg', "`"$ConfigPath`"",
    '-LogFileArg', "`"$LogPath`""
  )
  return ($psArgList -join ' ')
}

function Grant-LandInstallConfigRead {
  param([string]$ConfigPath)
  if ([string]::IsNullOrWhiteSpace($ConfigPath)) { return }
  if (-not (Test-Path -LiteralPath $ConfigPath)) { return }
  try {
    & icacls.exe $ConfigPath /grant 'Users:(R)' /grant 'Everyone:(R)' 2>$null | Out-Null
  } catch { }
}

function Write-InstallTraceTail {
  $tracePath = 'C:\Windows\Temp\sem_land_gui_install_trace.log'
  if (-not (Test-Path -LiteralPath $tracePath)) { return }
  try {
    $tail = @(Get-Content -LiteralPath $tracePath -ErrorAction Stop | Select-Object -Last 6)
    if ($tail.Count -gt 0) {
      Write-Output ("INSTALL_TRACE_TAIL|" + ($tail -join ' || '))
    }
  } catch { }
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
  Grant-LandInstallConfigRead -ConfigPath $configPath
} catch {
  Write-Output "INSTALL_ERROR|reason=config_write_failed|msg=$($_.Exception.Message)"
  exit 1
}

Write-Output "INSTALL_SCHEDULED|task=$taskName|user=$profileUser|installer=$installerPath|config=$configPath|log=$outFile"

$helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
if (-not (Test-Path -LiteralPath $helper)) {
  Write-Output "INSTALL_ERROR|reason=helper_missing|path=$helper"
  exit 1
}

$psArgs = Get-LandGuiWorkerCommandLine -ConfigPath $configPath -LogPath $outFile
Write-Output "INSTALL_TASK_CMD|$psArgs"
try {
  $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
  $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15) -MultipleInstances Queue
  $principal = New-ScheduledTaskPrincipal -UserId $profileUser -LogonType Interactive -RunLevel Highest
  Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
  Start-ScheduledTask -TaskName $taskName
  Write-Output "INTERACTIVE_INSTALL_TASK|name=$taskName|user=$profileUser|config=$configPath|log=$outFile"
} catch {
  Write-Output "INSTALL_ERROR|reason=scheduled_task_create_failed|msg=$($_.Exception.Message)"
  exit 1
}

$bootDeadline = (Get-Date).AddSeconds(90)
$earlyLineCount = 0
$workerStarted = $false
do {
  Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$earlyLineCount)
  if (Test-InstallWorkerStarted -Path $outFile) {
    $workerStarted = $true
    Write-Output 'INSTALL_WORKER_BOOT_OK'
    break
  }
  $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
  if ($taskInfo) {
    $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
    if ($lastHex -eq '800710E0') {
      Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
      break
    }
    $taskState = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
    if ($taskState -and $taskState -ne 'Running' -and $taskInfo.LastTaskResult -ne 267009) {
      Write-Output "INSTALL_TASK_EARLY_EXIT|state=$taskState|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex"
      break
    }
  }
  Start-Sleep -Seconds 1
} while ((Get-Date) -lt $bootDeadline)

if (-not $workerStarted) {
  Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$earlyLineCount)
  if (Test-Path -LiteralPath $outFile) {
    $tail = @(Get-Content -LiteralPath $outFile -ErrorAction SilentlyContinue | Select-Object -Last 8)
    if ($tail.Count -gt 0) {
      Write-Output ("INSTALL_TASK_LOG_TAIL|" + ($tail -join ' || '))
    }
  } else {
    Write-Output "INSTALL_TASK_LOG_EMPTY|log=$outFile"
  }
  Write-InstallTraceTail
  if ($taskInfo) {
    $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
    Write-Output "INSTALL_TASK_BOOT_TIMEOUT|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex"
  } else {
    Write-Output 'INSTALL_TASK_BOOT_TIMEOUT|last_result=unknown'
  }
  Write-Output 'INSTALL_ERROR|reason=worker_never_booted'
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  exit 1
}

try {
  $deadline = (Get-Date).AddSeconds($timeoutSec)
  $logComplete = $false
  $lastLineCount = $earlyLineCount
  do {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $outFile) {
      $logComplete = $true
      break
    }
    Start-Sleep -Seconds 1
  } while ((Get-Date) -lt $deadline)

  if (-not $logComplete) {
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    Write-Output "INSTALL_TASK_LOG_INCOMPLETE|log=$outFile"
  }
} catch {
  Write-Output "INSTALL_TASK_ERROR|msg=$($_.Exception.Message)"
  Write-Output 'INSTALL_ERROR|reason=task_failed'
  exit 1
} finally {
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  Remove-Item -LiteralPath $configPath -Force -ErrorAction SilentlyContinue
}

if ((Get-InstallLogTerminalStatus -Path $outFile) -eq 'complete') {
  exit 0
}
exit 1
