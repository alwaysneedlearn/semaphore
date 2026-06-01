# Run graceful-stop helper in logged-in desktop user's interactive session.
# Env:
#   STOP_GRACEFUL_PROCESS_NAME, STOP_POPUP_WAIT_SECONDS, STOP_POPUP_KEYWORD,
#   STOP_FORCE_AFTER_GRACEFUL, STOP_VERIFY_PROCESS_NAME
#   SEMAPHORE_PROFILE_USER (required interactive user)
#   STOP_TASK_TIMEOUT_SECONDS (optional, default 45)

$procName = [string]$env:STOP_GRACEFUL_PROCESS_NAME
if ($null -eq $procName) { $procName = '' }
$procName = $procName.Trim()
if ($procName.Length -eq 0) { $procName = 'LHBTS' }

$popupKeyword = [string]$env:STOP_POPUP_KEYWORD
if ($null -eq $popupKeyword) { $popupKeyword = '' }
$popupKeyword = $popupKeyword.Trim()
if ($popupKeyword.Length -eq 0) { $popupKeyword = '警告' }

$waitSecRaw = [string]$env:STOP_POPUP_WAIT_SECONDS
if ($null -eq $waitSecRaw -or $waitSecRaw.Trim().Length -eq 0) { $waitSecRaw = '2' }
[int]$waitSec = 2
[int]::TryParse($waitSecRaw, [ref]$waitSec) | Out-Null
if ($waitSec -lt 0) { $waitSec = 0 }

$forceRaw = [string]$env:STOP_FORCE_AFTER_GRACEFUL
if ($null -eq $forceRaw) { $forceRaw = '' }
$forceRaw = $forceRaw.Trim()
if ($forceRaw.Length -eq 0) { $forceRaw = 'true' }

$verifyName = [string]$env:STOP_VERIFY_PROCESS_NAME
if ($null -eq $verifyName) { $verifyName = '' }
$verifyName = $verifyName.Trim()
if ($verifyName.Length -eq 0) { $verifyName = $procName }

$profileUser = [string]$env:SEMAPHORE_PROFILE_USER
if ($null -eq $profileUser) { $profileUser = '' }
$profileUser = $profileUser.Trim()
$profileShort = ($profileUser -split '\\')[-1]

[int]$timeoutSec = 45
$timeoutRaw = [string]$env:STOP_TASK_TIMEOUT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($timeoutRaw)) {
  [int]::TryParse($timeoutRaw, [ref]$timeoutSec) | Out-Null
}
if ($timeoutSec -lt 10) { $timeoutSec = 10 }

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

if ([string]::IsNullOrWhiteSpace($profileUser)) {
  Write-Output "INTERACTIVE_SESSION_REQUIRED|reason=profile_user_empty"
  Write-Output "STILL_RUNNING"
  exit 1
}

$session = Test-ProfileUserInteractiveSession -ShortName $profileShort
if (-not $session.ok) {
  Write-Output "INTERACTIVE_SESSION_REQUIRED|user=$profileUser|reason=no_explorer_for_user"
  Write-Output "STILL_RUNNING"
  exit 1
}

$taskName = "StopGraceful-$($procName)-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
$helper = 'C:\Windows\Temp\sem_stop_close_main_window_confirm.ps1'
$outFile = "C:\Windows\Temp\sem_stop_out_$($procName)_$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds()).log"

$args = @(
  '-NoProfile',
  '-ExecutionPolicy', 'Bypass',
  '-File', $helper,
  '-ProcNameArg', $procName,
  '-PopupKeywordArg', $popupKeyword,
  '-PopupWaitSecondsArg', $waitSec,
  '-ForceAfterArg', $forceRaw,
  '-VerifyNameArg', $verifyName
) -join ' '

$escapedOut = $outFile.Replace("'", "''")
$cmd = "powershell.exe $args *> '$escapedOut'"
$action = New-ScheduledTaskAction -Execute 'cmd.exe' -Argument "/c $cmd"
$trigger = New-ScheduledTaskTrigger -Once -At ((Get-Date).AddSeconds(2))
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 5)
$principal = New-ScheduledTaskPrincipal -UserId $profileUser -LogonType Interactive -RunLevel Highest

try {
  Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
  Start-ScheduledTask -TaskName $taskName
  Write-Output "INTERACTIVE_STOP_TASK|name=$taskName|user=$profileUser|explorer_pid=$($session.explorer_pid)"

  $deadline = (Get-Date).AddSeconds($timeoutSec)
  do {
    Start-Sleep -Seconds 1
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
    if ($state -ne 'Running' -and (Test-Path -LiteralPath $outFile)) { break }
  } while ((Get-Date) -lt $deadline)

  if (Test-Path -LiteralPath $outFile) {
    Get-Content -LiteralPath $outFile | ForEach-Object { Write-Output $_ }
  } else {
    Write-Output "STOP_TASK_NO_OUTPUT|task=$taskName|timeout=${timeoutSec}s"
  }
} catch {
  Write-Output "STOP_TASK_ERROR:$($_.Exception.Message)"
  Write-Output "STILL_RUNNING"
  exit 1
} finally {
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
}
