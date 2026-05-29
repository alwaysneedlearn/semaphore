# Graceful stop: CloseMainWindow -> wait -> confirm popup by title keyword (Enter).
# Env (Ansible): STOP_GRACEFUL_PROCESS_NAME, STOP_POPUP_WAIT_SECONDS, STOP_POPUP_KEYWORD,
#                STOP_FORCE_AFTER_GRACEFUL (true/false), optional STOP_VERIFY_PROCESS_NAME
$ErrorActionPreference = 'Continue'

$procName = [string]$env:STOP_GRACEFUL_PROCESS_NAME
if ([string]::IsNullOrWhiteSpace($procName)) { $procName = 'LHBTS' }

$waitSec = 2
if (-not [string]::IsNullOrWhiteSpace($env:STOP_POPUP_WAIT_SECONDS)) {
  [int]::TryParse($env:STOP_POPUP_WAIT_SECONDS, [ref]$waitSec) | Out-Null
}
if ($waitSec -lt 0) { $waitSec = 0 }

$keyword = [string]$env:STOP_POPUP_KEYWORD
if ([string]::IsNullOrWhiteSpace($keyword)) { $keyword = '警告' }

$forceAfter = $true
if (-not [string]::IsNullOrWhiteSpace($env:STOP_FORCE_AFTER_GRACEFUL)) {
  $forceAfter = $env:STOP_FORCE_AFTER_GRACEFUL -match '^(?i:true|1|yes)$'
}

$verifyName = [string]$env:STOP_VERIFY_PROCESS_NAME
if ([string]::IsNullOrWhiteSpace($verifyName)) { $verifyName = $procName }

$procs = @(Get-Process -Name $procName -ErrorAction SilentlyContinue)
if ($procs.Count -eq 0) {
  Write-Output 'ALREADY_STOPPED'
  exit 0
}

$closed = 0
foreach ($p in $procs) {
  try {
    if ($p.CloseMainWindow()) { $closed++ }
  } catch {
    Write-Output "CLOSE_MAIN_WINDOW_ERROR:$($p.Id)|$($_.Exception.Message)"
  }
}
Write-Output "CLOSE_MAIN_WINDOW_SENT|count=$closed|process=$procName"

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

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    public const uint WM_KEYDOWN = 0x0100;
    public const int VK_RETURN = 0x0D;

    public static List<string> SendEnterByKeyword(string keyword) {
        var matchedTitles = new List<string>();
        EnumWindows((hWnd, lParam) => {
            if (IsWindowVisible(hWnd)) {
                StringBuilder sb = new StringBuilder(256);
                GetWindowText(hWnd, sb, sb.Capacity);
                string title = sb.ToString();
                if (!string.IsNullOrEmpty(title) && title.Contains(keyword)) {
                    PostMessage(hWnd, WM_KEYDOWN, (IntPtr)VK_RETURN, IntPtr.Zero);
                    matchedTitles.Add(title);
                }
            }
            return true;
        }, IntPtr.Zero);
        return matchedTitles;
    }
}
"@
}

$closedList = [SemaphoreEnterKeySender]::SendEnterByKeyword($keyword)
if ($closedList.Count -gt 0) {
  foreach ($t in $closedList) {
    Write-Output "POPUP_CONFIRMED|title=$t"
  }
} else {
  Write-Output "POPUP_NOT_FOUND|keyword=$keyword"
}

Start-Sleep -Seconds 1

$still = @(Get-Process -Name $verifyName -ErrorAction SilentlyContinue)
if ($still.Count -eq 0) {
  Write-Output 'NOT_RUNNING'
  exit 0
}

if ($forceAfter) {
  $still | Stop-Process -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 1
  $after = @(Get-Process -Name $verifyName -ErrorAction SilentlyContinue)
  if ($after.Count -eq 0) {
    Write-Output 'FORCE_STOPPED'
    exit 0
  }
  Write-Output 'STILL_RUNNING'
  exit 1
}

Write-Output 'STILL_RUNNING'
exit 1
