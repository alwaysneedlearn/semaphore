# Run LAND GUI installer worker in logged-in desktop user's interactive session.
# Env:
#   LAND_INSTALLER_EXE_PATH (required)
#   SEMAPHORE_PROFILE_USER (required)
#   LAND_INSTALL_TASK_TIMEOUT_SECONDS (optional, default 180)

$profileUser = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUser) { $profileUser = '' }
$profileUser = $profileUser.Trim()
$profileShort = ($profileUser -split '\\')[-1]

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
    if ($t -match '^(INSTALL_COMPLETE|INSTALL_FAILED|INSTALL_ERROR)\b') {
      return $true
    }
  }
  return $false
}

if ([string]::IsNullOrWhiteSpace($profileUser)) {
  Write-Output 'INTERACTIVE_SESSION_REQUIRED|reason=profile_user_empty'
  Write-Output 'INSTALL_ERROR|reason=profile_user_empty'
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
  '-File', "`"$helper`"",
  '-LogFileArg', "`"$outFile`""
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
  Write-Output "INTERACTIVE_INSTALL_TASK|name=$taskName|user=$profileUser|explorer_pid=$($session.explorer_pid)"

  $deadline = (Get-Date).AddSeconds($timeoutSec)
  $logComplete = $false
  do {
    Start-Sleep -Seconds 1
    if (Test-InstallLogTerminalLine -Path $outFile) {
      $logComplete = $true
      break
    }
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($task -and $task.State -ne 'Running') {
      Start-Sleep -Seconds 2
      if (Test-InstallLogTerminalLine -Path $outFile) {
        $logComplete = $true
        break
      }
    }
  } while ((Get-Date) -lt $deadline)

  if (-not $logComplete -and (Test-Path -LiteralPath $outFile)) {
    Start-Sleep -Seconds 3
    if (Test-InstallLogTerminalLine -Path $outFile) { $logComplete = $true }
  }

  if (Test-Path -LiteralPath $outFile) {
    Get-Content -LiteralPath $outFile | ForEach-Object { Write-Output $_ }
    if (-not $logComplete) {
      Write-Output "INSTALL_TASK_LOG_INCOMPLETE|task=$taskName|log=$outFile"
    }
  } else {
    Write-Output "INSTALL_TASK_NO_OUTPUT|task=$taskName|timeout=${timeoutSec}s|log=$outFile"
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
