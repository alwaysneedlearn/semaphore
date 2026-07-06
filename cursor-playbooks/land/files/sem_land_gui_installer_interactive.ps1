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
$dlg2Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_BUTTON' -Default '确认'
$dlg3Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_TITLE' -Default 'LHBTS 安装'
$dlg3Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_BUTTON' -Default '确定'
$stepTimeout = Get-EnvOrDefault -Name 'LAND_INSTALL_STEP_TIMEOUT_SECONDS' -Default '90'
$clickSettleMs = Get-EnvOrDefault -Name 'LAND_INSTALL_CLICK_SETTLE_MS' -Default '1500'
$coordMoveDelayMs = Get-EnvOrDefault -Name 'LAND_INSTALL_COORD_MOVE_DELAY_MS' -Default '1200'
$pollDelayMs = Get-EnvOrDefault -Name 'LAND_INSTALL_POLL_DELAY_MS' -Default '800'
$dlg1CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_COORD_X_PCT' -Default '18'
$dlg1CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_COORD_Y_PCT' -Default '93'
# Step 2 (提示 popup): use Win32 button text match only — coord 0 disables fallback click.
$dlg2CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_X_PCT' -Default '0'
$dlg2CoordY = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_COORD_Y_PCT' -Default '0'
# Step 3 (single 确定): same client-area position as 取消 on step 1 (right).
$dlg3CoordX = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_COORD_X_PCT' -Default '83'
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
  $explorers = Get-CimInstance Win32_Process -Filter "Name='explorer.exe'" -ErrorAction SilentlyContinue
  foreach ($e in $explorers) {
    $o = Invoke-CimMethod -InputObject $e -MethodName GetOwner -ErrorAction SilentlyContinue
    if ($o -and $o.ReturnValue -eq 0 -and $o.User -and ($o.User.Trim() -ieq $ShortName)) {
      return @{ ok = $true; explorer_pid = $e.ProcessId }
    }
  }
  return @{ ok = $false; explorer_pid = 0 }
}

function Test-InstallLogTerminalLine {
  param([string]$Path)
  if (-not (Test-Path -LiteralPath $Path)) { return $false }
  try {
    $lines = @(Get-Content -LiteralPath $Path -ErrorAction Stop)
  } catch {
    return $false
  }
  foreach ($line in $lines) {
    $t = ([string]$line).Trim()
    if ($t -match '(INSTALL_COMPLETE|INSTALL_FAILED|INSTALL_ERROR)\b') {
      return $true
    }
  }
  return $false
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

  $psArgs = "-NoProfile -ExecutionPolicy Bypass -File `"$helper`" -ConfigFileArg `"$ConfigPath`""
  try {
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15)
    $principal = New-ScheduledTaskPrincipal -UserId $ProfileUser -LogonType Interactive -RunLevel $RunLevel
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $TaskName
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
  click_settle_ms = $clickSettleMs
  coord_move_delay_ms = $coordMoveDelayMs
  poll_delay_ms = $pollDelayMs
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

$started = $false
$lastStartError = ''
foreach ($runLevel in @('Highest', 'Limited')) {
  $result = Start-LandGuiInstallScheduledTask -TaskName $taskName -ProfileUser $profileUser -ConfigPath $configPath -LogPath $outFile -RunLevel $runLevel
  if (-not $result.ok) {
    $lastStartError = [string]$result.error
    Write-Output "INSTALL_TASK_START_FAILED|run_level=$runLevel|error=$lastStartError"
    continue
  }
  Start-Sleep -Seconds 5
  $earlyLineCount = 0
  Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$earlyLineCount)
  if (Test-InstallWorkerStarted -Path $outFile) {
    $started = $true
    break
  }
  $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
  if ($taskInfo) {
    $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
    Write-Output "INSTALL_TASK_EARLY_STATE|run_level=$runLevel|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)"
    if ($lastHex -eq '800710E0') {
      Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
    }
  }
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  if ($runLevel -eq 'Limited') { break }
  $taskName = "LandGuiInstall-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
}

if (-not $started) {
  Write-Output "INSTALL_ERROR|reason=scheduled_task_never_started_worker|last_error=$lastStartError|user=$profileUser"
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
    Start-Sleep -Seconds 2
    Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
    if (Test-InstallLogTerminalLine -Path $outFile) {
      $logComplete = $true
      break
    }
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($task -and $task.State -ne 'Running') {
      Start-Sleep -Seconds 2
      Write-InstallLogTail -Path $outFile -LastLineCount ([ref]$lastLineCount)
      if (-not $taskStateLogged) {
        $taskStateLogged = $true
        $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
        if ($taskInfo) {
          $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
          Write-Output "INSTALL_TASK_STATE|name=$taskName|state=$($task.State)|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)"
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
  } while ((Get-Date) -lt $deadline)

  if (-not $logComplete -and (Test-Path -LiteralPath $outFile)) {
    Start-Sleep -Seconds 2
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
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  Remove-Item -LiteralPath $configPath -Force -ErrorAction SilentlyContinue
}

if ((Get-Content -LiteralPath $outFile -ErrorAction SilentlyContinue | Out-String) -match 'INSTALL_COMPLETE') {
  exit 0
}
exit 1
