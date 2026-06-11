# Interactive scheduled task start for desktop profile user (reads SEMAPHORE_* env vars)
$startExe = [string]$env:SEMAPHORE_EXE_PATH
if ($null -eq $startExe) { $startExe = '' }
$startExe = $startExe.Trim()
# PS 5.1: Split-Path -LiteralPath 与 -Parent 不能同用；用 .NET API 兼容带空格路径
$workDir = [System.IO.Path]::GetDirectoryName($startExe)
if ([string]::IsNullOrWhiteSpace($workDir)) {
  $workDir = Split-Path -Path $startExe -Parent
}
$procName = [System.IO.Path]::GetFileNameWithoutExtension($startExe)
$envProcName = [string]$env:SEMAPHORE_EXE_NAME
if ($null -eq $envProcName) { $envProcName = '' }
$envProcName = $envProcName.Trim()
$waitSeconds = [int]$env:SEMAPHORE_RESTART_DELAY
$pollSeconds = [int]$env:SEMAPHORE_PROCESS_VERIFY_POLL_SECONDS
if ($pollSeconds -lt 1) { $pollSeconds = 5 }
$profileUser = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUser) { $profileUser = '' }
$profileUser = $profileUser.Trim()
$profileShort = ($profileUser -split '\\')[-1]
$argLine = [string]$env:SEMAPHORE_EXE_ARGS
if ($null -eq $argLine) { $argLine = '' }
$taskName = [string]$env:SEMAPHORE_TASK_NAME
if ($null -eq $taskName -or $taskName.Trim().Length -eq 0) {
  $taskName = 'Start-' + $envProcName
}
$startPopupKeyword = [string]$env:SEMAPHORE_START_POPUP_KEYWORD
if ($null -eq $startPopupKeyword) { $startPopupKeyword = '' }
$startPopupKeyword = $startPopupKeyword.Trim()
$startPopupWaitSec = 2
$startPopupWaitRaw = [string]$env:SEMAPHORE_START_POPUP_WAIT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($startPopupWaitRaw)) {
  [int]::TryParse($startPopupWaitRaw, [ref]$startPopupWaitSec) | Out-Null
}
if ($startPopupWaitSec -lt 0) { $startPopupWaitSec = 0 }

if ($envProcName.Length -gt 0 -and $envProcName -ne $procName) {
  Write-Output "RECONFIG_PROC_NAME_HINT=exe_file=$procName|EXE_NAME=$envProcName"
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

function Test-ExeProcessRunning {
  param([string]$Name, [string]$LiteralPath)
  $helper = 'C:\Windows\Temp\sem_process_alive_windows.ps1'
  if (Test-Path -LiteralPath $helper) {
    . $helper
    $t = Get-SemaphoreProcessAliveThresholds
    $p = Get-SemaphoreAliveProcess -ProcessName $Name -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
    if ($p) { return $p }
    if ($envProcName.Length -gt 0 -and $envProcName -ne $Name) {
      $p2 = Get-SemaphoreAliveProcess -ProcessName $envProcName -MinHandles $t.MinHandles -MinWsBytes $t.MinWsBytes
      if ($p2) { return $p2 }
    }
  } else {
    $p = Get-Process -Name $Name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($p) { return $p }
    if ($envProcName.Length -gt 0 -and $envProcName -ne $Name) {
      $p2 = Get-Process -Name $envProcName -ErrorAction SilentlyContinue | Select-Object -First 1
      if ($p2) { return $p2 }
    }
  }
  $cim = Get-CimInstance Win32_Process -Filter "Name='$Name.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -eq $LiteralPath) } |
    Select-Object -First 1
  if ($cim) {
    return [PSCustomObject]@{ Id = $cim.ProcessId; ProcessName = $cim.Name }
  }
  return $null
}

function Invoke-InteractiveStartPopupConfirm {
  param(
    [string]$ProfileUser,
    [string]$Keyword,
    [int]$WaitSec
  )
  if ([string]::IsNullOrWhiteSpace($Keyword)) {
    return $false
  }
  $helper = 'C:\Windows\Temp\sem_popup_confirm_by_keyword.ps1'
  if (-not (Test-Path -LiteralPath $helper)) {
    Write-Output "START_POPUP_HELPER_MISSING|path=$helper"
    return $false
  }
  $taskName = "StartPopup-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
  $logFile = "C:\Windows\Temp\sem_start_popup_$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds()).log"
  $psArgList = @(
    '-NoProfile',
    '-ExecutionPolicy', 'Bypass',
    '-File', "`"$helper`"",
    '-PopupKeywordArg', $Keyword,
    '-PopupWaitSecondsArg', $WaitSec,
    '-LogFileArg', "`"$logFile`""
  )
  $psArgs = $psArgList -join ' '
  try {
    $action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $psArgs
    $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(1))
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 2)
    $principal = New-ScheduledTaskPrincipal -UserId $ProfileUser -LogonType Interactive -RunLevel Highest
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $taskName
    Write-Output "START_POPUP_TASK|name=$taskName|user=$ProfileUser|keyword=$Keyword"
    $deadline = (Get-Date).AddSeconds(20)
    do {
      Start-Sleep -Seconds 1
      if (Test-Path -LiteralPath $logFile) {
        $lines = @(Get-Content -LiteralPath $logFile -ErrorAction SilentlyContinue)
        foreach ($line in $lines) {
          Write-Output $line
          if ($line -match '^POPUP_CONFIRMED\b') {
            return $true
          }
          if ($line -match '^POPUP_NOT_FOUND\b') {
            return $false
          }
        }
      }
      $info = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
      if ($info -and (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State -ne 'Running') {
        Start-Sleep -Seconds 1
        if (Test-Path -LiteralPath $logFile) {
          Get-Content -LiteralPath $logFile -ErrorAction SilentlyContinue | ForEach-Object { Write-Output $_ }
        }
        break
      }
    } while ((Get-Date) -lt $deadline)
  } catch {
    Write-Output "START_POPUP_ERROR:$($_.Exception.Message)"
  } finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  }
  return $false
}

function Wait-ExeProcess {
  param(
    [int]$TotalSeconds,
    [int]$IntervalSeconds,
    [string]$Name,
    [string]$LiteralPath,
    [string]$ProfileUser,
    [string]$PopupKeyword,
    [int]$PopupWaitSec
  )
  $deadline = (Get-Date).AddSeconds($TotalSeconds)
  $attempt = 0
  $popupKeywordSet = -not [string]::IsNullOrWhiteSpace($PopupKeyword)
  do {
    $attempt++
    $found = Test-ExeProcessRunning -Name $Name -LiteralPath $LiteralPath
    if ($found) {
      if ($popupKeywordSet -and ($attempt -eq 1 -or ($attempt % 2) -eq 0)) {
        [void](Invoke-InteractiveStartPopupConfirm -ProfileUser $ProfileUser -Keyword $PopupKeyword -WaitSec $PopupWaitSec)
        Start-Sleep -Seconds 1
        $found = Test-ExeProcessRunning -Name $Name -LiteralPath $LiteralPath
      }
      if ($found) {
        Write-Output "VERIFY_POLL|attempt=$attempt|found=true|pid=$($found.Id)|name=$($found.ProcessName)"
        return $found
      }
    }
    if ($popupKeywordSet -and (($attempt % 2) -eq 1)) {
      [void](Invoke-InteractiveStartPopupConfirm -ProfileUser $ProfileUser -Keyword $PopupKeyword -WaitSec $PopupWaitSec)
    }
    Write-Output "VERIFY_POLL|attempt=$attempt|found=false|waited=$((Get-Date))"
    if ((Get-Date) -ge $deadline) { break }
    Start-Sleep -Seconds $IntervalSeconds
  } while ($true)
  return $null
}

Write-Output "RECONFIG_START_EXE_PATH=$startExe|exists=$(Test-Path -LiteralPath $startExe)|workdir=$workDir|proc=$procName"
Write-Output "RECONFIG_RESTART_DELAY_SECONDS=$waitSeconds"
Write-Output "RECONFIG_PROFILE_USER=$profileUser"
Write-Output "RECONFIG_START_POPUP_KEYWORD=$startPopupKeyword"
Write-Output "RECONFIG_WINRM_RUN_AS=$([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)|note=WinRM_for_deploy_only_not_exe_start"

if (-not (Test-Path -LiteralPath $startExe)) {
  Write-Output "EXE_NOT_FOUND_AT_START|path=$startExe"
  exit 0
}

$session = Test-ProfileUserInteractiveSession -ShortName $profileShort
if (-not $session.ok) {
  Write-Output "INTERACTIVE_SESSION_REQUIRED|user=$profileUser|reason=no_explorer_for_user|action=请让用户RDP登录桌面后再执行模板"
  Write-Output "VERIFY_FAILED|proc=$procName|reason=interactive_user_not_logged_on"
  exit 0
}
Write-Output "RECONFIG_INTERACTIVE_SESSION|user=$profileUser|explorer_pid=$($session.explorer_pid)|ok=true"

try {
  if ($argLine -ne '') {
    $action = New-ScheduledTaskAction -Execute $startExe -Argument $argLine -WorkingDirectory $workDir
  } else {
    $action = New-ScheduledTaskAction -Execute $startExe -WorkingDirectory $workDir
  }
  $trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
  # ExecutionTimeLimit controls how long the scheduled task is allowed to run.
  # We expect the program to keep running; when no better option exists, use a very long limit.
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Days 365)
  $principal = New-ScheduledTaskPrincipal -UserId $profileUser -LogonType Interactive -RunLevel Highest
  Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
  $task = Get-ScheduledTask -TaskName $taskName
  Write-Output "RECONFIG_TASK_REGISTERED|name=$taskName|state=$($task.State)|user=$profileUser|logon=Interactive|start_as=desktop_user_only"
  if ($task.State -ne 'Running') {
    Start-ScheduledTask -TaskName $taskName
    Start-Sleep -Seconds 2
  }
  if ($startPopupKeyword.Length -gt 0) {
    [void](Invoke-InteractiveStartPopupConfirm -ProfileUser $profileUser -Keyword $startPopupKeyword -WaitSec $startPopupWaitSec)
  }
  $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName
  $lastHex = '{0:X8}' -f ([uint32]($taskInfo.LastTaskResult))
  Write-Output "RECONFIG_TASK_INFO|last_result=$($taskInfo.LastTaskResult)|last_result_hex=0x$lastHex|last_run=$($taskInfo.LastRunTime)|state=$((Get-ScheduledTask -TaskName $taskName).State)"
  if ($lastHex -eq '800710E0') {
    Write-Output "RECONFIG_TASK_HINT|code=0x800710E0|meaning=操作员或系统管理员拒绝了请求|likely=用户未在交互桌面登录或UAC策略拒绝"
  }

  $verify = Wait-ExeProcess -TotalSeconds $waitSeconds -IntervalSeconds $pollSeconds -Name $procName -LiteralPath $startExe -ProfileUser $profileUser -PopupKeyword $startPopupKeyword -PopupWaitSec $startPopupWaitSec
  if ($verify) {
    Write-Output "VERIFY_OK|PID:$($verify.Id)|method=scheduled_task_interactive|user=$profileUser"
    exit 0
  }

  Write-Output "VERIFY_SCHEDULED_TASK_MISS|last_result=$($taskInfo.LastTaskResult)|user=$profileUser|hint=仅通过已登录用户的Interactive计划任务启动；请确认RDP会话在线或增大RESTART_DELAY"
  Write-Output "VERIFY_FAILED|proc=$procName|scheduled_last_result=$($taskInfo.LastTaskResult)|reason=interactive_scheduled_task_no_process"
} catch {
  Write-Output "START_ERROR:$($_.Exception.Message)"
}
