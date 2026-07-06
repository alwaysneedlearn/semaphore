# Run LAND GUI installer worker in logged-in desktop user's interactive session.
# Reads LAND_INSTALL_* / SEMAPHORE_PROFILE_USER from env (WinRM parent) and passes args to worker.

$profileUser = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUser) { $profileUser = '' }
$profileUser = $profileUser.Trim()
$profileShort = ($profileUser -split '\\')[-1]

$installerPath = [string]$env:LAND_INSTALLER_EXE_PATH
if ($null -eq $installerPath) { $installerPath = '' }
$installerPath = $installerPath.Trim()

function Get-EnvOrDefault {
  param([string]$Name, [string]$Default)
  $v = [string]$env:$Name
  if ($null -eq $v) { return $Default }
  $v = $v.Trim()
  if ($v.Length -eq 0) { return $Default }
  return $v
}

function Quote-SemPsArg {
  param([string]$Value)
  if ($null -eq $Value) { $Value = '' }
  if ($Value -match '[\s"]') {
    return '"' + ($Value -replace '"', '""') + '"'
  }
  return $Value
}

$dlg1Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_TITLE' -Default 'LHBTS 安装'
$dlg1Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG1_BUTTON' -Default '升级'
$dlg2Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_TITLE' -Default '提示'
$dlg2Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG2_BUTTON' -Default '确认'
$dlg3Title = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_TITLE' -Default 'LHBTS 安装'
$dlg3Button = Get-EnvOrDefault -Name 'LAND_INSTALL_DLG3_BUTTON' -Default '确定'
$stepTimeout = Get-EnvOrDefault -Name 'LAND_INSTALL_STEP_TIMEOUT_SECONDS' -Default '90'
$clickSettleMs = Get-EnvOrDefault -Name 'LAND_INSTALL_CLICK_SETTLE_MS' -Default '400'

[int]$timeoutSec = 180
$timeoutRaw = [string]$env:LAND_INSTALL_TASK_TIMEOUT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($timeoutRaw)) {
  [int]::TryParse($timeoutRaw, [ref]$timeoutSec) | Out-Null
}
if ($timeoutSec -lt 30) { $timeoutSec = 30 }

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

if ([string]::IsNullOrWhiteSpace($profileUser)) {
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

$session = Test-ProfileUserInteractiveSession -ShortName $profileShort
if (-not $session.ok) {
  Write-Output "INTERACTIVE_SESSION_REQUIRED|user=$profileUser|reason=no_explorer_for_user"
  Write-Output 'INSTALL_ERROR|reason=no_interactive_session'
  exit 1
}

$helper = 'C:\Windows\Temp\sem_land_gui_installer_worker.ps1'
$outFile = "C:\Windows\Temp\sem_land_install_$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds()).log"

if (-not (Test-Path -LiteralPath $helper)) {
  Write-Output "INSTALL_HELPER_MISSING|path=$helper"
  Write-Output 'INSTALL_ERROR|reason=helper_missing'
  exit 1
}

$psArgList = @(
  '-NoProfile',
  '-ExecutionPolicy', 'Bypass',
  '-File', (Quote-SemPsArg $helper),
  '-InstallerPathArg', (Quote-SemPsArg $installerPath),
  '-LogFileArg', (Quote-SemPsArg $outFile),
  '-Dlg1TitleArg', (Quote-SemPsArg $dlg1Title),
  '-Dlg1ButtonArg', (Quote-SemPsArg $dlg1Button),
  '-Dlg2TitleArg', (Quote-SemPsArg $dlg2Title),
  '-Dlg2ButtonArg', (Quote-SemPsArg $dlg2Button),
  '-Dlg3TitleArg', (Quote-SemPsArg $dlg3Title),
  '-Dlg3ButtonArg', (Quote-SemPsArg $dlg3Button),
  '-StepTimeoutArg', (Quote-SemPsArg $stepTimeout),
  '-ClickSettleMsArg', (Quote-SemPsArg $clickSettleMs)
)
$psArgs = $psArgList -join ' '
$taskName = "LandGuiInstall-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"

try {
  $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
  $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 15)
  $principal = New-ScheduledTaskPrincipal -UserId $profileUser -LogonType Interactive -RunLevel Highest
  Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
  Start-ScheduledTask -TaskName $taskName
  Write-Output "INTERACTIVE_INSTALL_TASK|name=$taskName|user=$profileUser|explorer_pid=$($session.explorer_pid)|installer=$installerPath"

  $deadline = (Get-Date).AddSeconds($timeoutSec)
  $logComplete = $false
  $lastLineCount = 0
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
      $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
      if ($taskInfo) {
        $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
        Write-Output "INSTALL_TASK_STATE|name=$taskName|state=$($task.State)|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)"
        if ($lastHex -eq '800710E0') {
          Write-Output 'INSTALL_TASK_HINT|code=0x800710E0|meaning=用户未在交互桌面登录或UAC/策略拒绝'
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
}

if ((Get-Content -LiteralPath $outFile -ErrorAction SilentlyContinue | Out-String) -match 'INSTALL_COMPLETE') {
  exit 0
}
exit 1
