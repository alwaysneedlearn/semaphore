# Graceful stop: CloseMainWindow -> wait -> confirm popup by title keyword (Enter).
# Supports args (preferred for scheduled-task invocation) and env fallback.
param(
  [string]$ProcNameArg = '',
  [string]$PopupKeywordArg = '',
  [int]$PopupWaitSecondsArg = -1,
  [string]$ForceAfterArg = '',
  [string]$VerifyNameArg = '',
  [string]$LogFileArg = ''
)

function Write-StopLine {
  param([string]$Line)
  Write-Output $Line
  if (-not [string]::IsNullOrWhiteSpace($script:StopLogPath)) {
    Add-Content -LiteralPath $script:StopLogPath -Value $Line -Encoding UTF8 -ErrorAction SilentlyContinue
  }
}

$script:StopLogPath = $LogFileArg
if ([string]::IsNullOrWhiteSpace($script:StopLogPath)) {
  $script:StopLogPath = [string]$env:STOP_LOG_FILE
}
if (-not [string]::IsNullOrWhiteSpace($script:StopLogPath)) {
  $logDir = Split-Path -Parent $script:StopLogPath
  if ($logDir -and -not (Test-Path -LiteralPath $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
  }
  Set-Content -LiteralPath $script:StopLogPath -Value '' -Encoding UTF8 -ErrorAction SilentlyContinue
}

# Env fallback: STOP_GRACEFUL_PROCESS_NAME, STOP_POPUP_WAIT_SECONDS, STOP_POPUP_KEYWORD,
#               STOP_FORCE_AFTER_GRACEFUL (true/false), optional STOP_VERIFY_PROCESS_NAME
$ErrorActionPreference = 'Continue'

$procName = $ProcNameArg
if ([string]::IsNullOrWhiteSpace($procName)) {
  $procName = [string]$env:STOP_GRACEFUL_PROCESS_NAME
}
if ([string]::IsNullOrWhiteSpace($procName)) { $procName = 'LHBTS' }

$waitSec = $PopupWaitSecondsArg
if ($waitSec -lt 0) {
  $waitSec = 2
  if (-not [string]::IsNullOrWhiteSpace($env:STOP_POPUP_WAIT_SECONDS)) {
    [int]::TryParse($env:STOP_POPUP_WAIT_SECONDS, [ref]$waitSec) | Out-Null
  }
}
if ($waitSec -lt 0) { $waitSec = 0 }

$keyword = $PopupKeywordArg
if ([string]::IsNullOrWhiteSpace($keyword)) {
  $keyword = [string]$env:STOP_POPUP_KEYWORD
}
if ([string]::IsNullOrWhiteSpace($keyword)) { $keyword = '警告' }

$forceAfter = $true
$forceRaw = $ForceAfterArg
if ([string]::IsNullOrWhiteSpace($forceRaw)) {
  $forceRaw = [string]$env:STOP_FORCE_AFTER_GRACEFUL
}
if (-not [string]::IsNullOrWhiteSpace($forceRaw)) {
  $forceAfter = $forceRaw -match '^(?i:true|1|yes)$'
}

$verifyName = $VerifyNameArg
if ([string]::IsNullOrWhiteSpace($verifyName)) {
  $verifyName = [string]$env:STOP_VERIFY_PROCESS_NAME
}
if ([string]::IsNullOrWhiteSpace($verifyName)) { $verifyName = $procName }

if (-not ([System.Management.Automation.PSTypeName]'SemaphoreWindowHelper').Type) {
  Add-Type @"
using System;
using System.Runtime.InteropServices;
public class SemaphoreWindowHelper {
    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
    public const int SW_RESTORE = 9;
    public static void RestoreMainWindow(IntPtr hWnd) {
        if (hWnd != IntPtr.Zero) { ShowWindow(hWnd, SW_RESTORE); }
    }
}
"@
}

$procs = @(Get-Process -Name $procName -ErrorAction SilentlyContinue)
if ($procs.Count -eq 0) {
  Write-StopLine 'ALREADY_STOPPED'
  exit 0
}

$closed = 0
$restored = 0
foreach ($p in $procs) {
  try {
    $h = $p.MainWindowHandle
    if ($h -ne [IntPtr]::Zero) {
      [SemaphoreWindowHelper]::RestoreMainWindow($h)
      $restored++
      Start-Sleep -Milliseconds 200
    }
    if ($p.CloseMainWindow()) { $closed++ }
  } catch {
    Write-StopLine "CLOSE_MAIN_WINDOW_ERROR:$($p.Id)|$($_.Exception.Message)"
  }
}
Write-StopLine "CLOSE_MAIN_WINDOW_SENT|count=$closed|restored=$restored|process=$procName"

Start-Sleep -Seconds $waitSec

if (-not ([System.Management.Automation.PSTypeName]'SemaphoreEnterKeySender').Type) {
  Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
using System.Collections.Generic;

public class SemaphoreEnterKeySender {
    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);

    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsIconic(IntPtr hWnd);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    public const uint WM_KEYDOWN = 0x0100;
    public const int VK_RETURN = 0x0D;
    public const int SW_RESTORE = 9;

    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

    public static List<string> SendEnterByKeyword(string keyword, bool includeHidden) {
        var matchedTitles = new List<string>();
        EnumWindows((hWnd, lParam) => {
            StringBuilder sb = new StringBuilder(256);
            GetWindowText(hWnd, sb, sb.Capacity);
            string title = sb.ToString();
            if (string.IsNullOrEmpty(title) || !title.Contains(keyword)) {
                return true;
            }
            bool visible = IsWindowVisible(hWnd);
            bool iconic = IsIconic(hWnd);
            if (!visible && !includeHidden) {
                return true;
            }
            if (iconic || !visible) {
                ShowWindow(hWnd, SW_RESTORE);
            }
            PostMessage(hWnd, WM_KEYDOWN, (IntPtr)VK_RETURN, IntPtr.Zero);
            matchedTitles.Add(title + (iconic ? "|minimized=1" : ""));
            return true;
        }, IntPtr.Zero);
        return matchedTitles;
    }
}
"@
}

$closedList = [SemaphoreEnterKeySender]::SendEnterByKeyword($keyword, $false)
if ($closedList.Count -eq 0) {
  $closedList = [SemaphoreEnterKeySender]::SendEnterByKeyword($keyword, $true)
}
if ($closedList.Count -gt 0) {
  foreach ($t in $closedList) {
    Write-StopLine "POPUP_CONFIRMED|title=$t"
  }
} else {
  Write-StopLine "POPUP_NOT_FOUND|keyword=$keyword|hint=minimized_or_hidden_popup_may_need_manual_confirm"
}

Start-Sleep -Seconds 1

$still = @(Get-Process -Name $verifyName -ErrorAction SilentlyContinue)
if ($still.Count -eq 0) {
  Write-StopLine 'NOT_RUNNING'
  exit 0
}

if ($forceAfter) {
  $still | Stop-Process -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 1
  $after = @(Get-Process -Name $verifyName -ErrorAction SilentlyContinue)
  if ($after.Count -eq 0) {
    Write-StopLine 'FORCE_STOPPED'
    exit 0
  }
  Write-StopLine 'STILL_RUNNING'
  exit 1
}

Write-StopLine 'STILL_RUNNING'
exit 1
