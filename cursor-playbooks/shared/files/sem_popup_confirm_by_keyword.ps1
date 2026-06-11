# Confirm a desktop popup by sending Enter to windows whose title contains a keyword.
# Used from interactive scheduled tasks (start/stop helpers).
param(
  [string]$PopupKeywordArg = '',
  [int]$PopupWaitSecondsArg = -1,
  [string]$LogFileArg = ''
)

function Write-PopupLine {
  param([string]$Line)
  Write-Output $Line
  if (-not [string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
    Add-Content -LiteralPath $script:PopupLogPath -Value $Line -Encoding UTF8 -ErrorAction SilentlyContinue
  }
}

$script:PopupLogPath = $LogFileArg
if ([string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
  $script:PopupLogPath = [string]$env:POPUP_LOG_FILE
}
if (-not [string]::IsNullOrWhiteSpace($script:PopupLogPath)) {
  $logDir = Split-Path -Parent $script:PopupLogPath
  if ($logDir -and -not (Test-Path -LiteralPath $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
  }
  Set-Content -LiteralPath $script:PopupLogPath -Value '' -Encoding UTF8 -ErrorAction SilentlyContinue
}

$waitSec = $PopupWaitSecondsArg
if ($waitSec -lt 0) {
  $waitSec = 2
  if (-not [string]::IsNullOrWhiteSpace($env:POPUP_WAIT_SECONDS)) {
    [int]::TryParse($env:POPUP_WAIT_SECONDS, [ref]$waitSec) | Out-Null
  }
}
if ($waitSec -lt 0) { $waitSec = 0 }

$keyword = $PopupKeywordArg
if ([string]::IsNullOrWhiteSpace($keyword)) {
  $keyword = [string]$env:POPUP_KEYWORD
}
if ([string]::IsNullOrWhiteSpace($keyword)) {
  Write-PopupLine 'POPUP_SKIP|reason=empty_keyword'
  exit 0
}

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

$matched = [SemaphoreEnterKeySender]::SendEnterByKeyword($keyword, $false)
if ($matched.Count -eq 0) {
  $matched = [SemaphoreEnterKeySender]::SendEnterByKeyword($keyword, $true)
}
if ($matched.Count -gt 0) {
  foreach ($t in $matched) {
    Write-PopupLine "POPUP_CONFIRMED|keyword=$keyword|title=$t"
  }
} else {
  Write-PopupLine "POPUP_NOT_FOUND|keyword=$keyword|hint=minimized_or_hidden_popup_may_need_manual_confirm"
}

exit 0
