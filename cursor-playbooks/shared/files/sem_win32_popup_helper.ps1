# Dot-source: Win32 popup confirm by window title OR child control text (content).
# Used by sem_stop_close_main_window_confirm.ps1 and sem_popup_confirm_by_keyword.ps1

if (-not ([System.Management.Automation.PSTypeName]'SemaphorePopupHelper').Type) {
  Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
using System.Collections.Generic;

public class SemaphorePopupHelper {
    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumChildWindows(IntPtr hWndParent, EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);

    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsIconic(IntPtr hWnd);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern IntPtr SendMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint lpdwProcessId);

    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

    public const uint WM_KEYDOWN = 0x0100;
    public const uint BM_CLICK = 0x00F5;
    public const int VK_RETURN = 0x0D;
    public const int SW_RESTORE = 9;

    public static string GetAggregatedWindowText(IntPtr hWnd) {
        StringBuilder sb = new StringBuilder(256);
        GetWindowText(hWnd, sb, sb.Capacity);
        StringBuilder all = new StringBuilder();
        string self = sb.ToString();
        if (!string.IsNullOrEmpty(self)) {
            all.Append(self);
        }
        EnumChildWindows(hWnd, (child, lParam) => {
            StringBuilder csb = new StringBuilder(512);
            GetWindowText(child, csb, csb.Capacity);
            string ct = csb.ToString();
            if (!string.IsNullOrEmpty(ct)) {
                if (all.Length > 0) { all.Append('|'); }
                all.Append(ct);
            }
            return true;
        }, IntPtr.Zero);
        return all.ToString();
    }

    public static bool WindowMatchesKeyword(IntPtr hWnd, string keyword, string matchMode) {
        if (string.IsNullOrEmpty(keyword)) { return false; }
        StringBuilder sb = new StringBuilder(256);
        GetWindowText(hWnd, sb, sb.Capacity);
        string title = sb.ToString();
        string mode = (matchMode ?? "title_or_content").ToLowerInvariant();
        if (mode == "title") {
            return !string.IsNullOrEmpty(title) && title.Contains(keyword);
        }
        if (mode == "content") {
            string blob = GetAggregatedWindowText(hWnd);
            return !string.IsNullOrEmpty(blob) && blob.Contains(keyword);
        }
        // title_or_content (default)
        if (!string.IsNullOrEmpty(title) && title.Contains(keyword)) { return true; }
        string all = GetAggregatedWindowText(hWnd);
        return !string.IsNullOrEmpty(all) && all.Contains(keyword);
    }

    public static bool TryClickConfirmButton(IntPtr hWnd, out string clickedText) {
        clickedText = null;
        IntPtr found = IntPtr.Zero;
        string foundText = null;
        EnumChildWindows(hWnd, (child, lParam) => {
            StringBuilder csb = new StringBuilder(256);
            GetWindowText(child, csb, csb.Capacity);
            string ct = (csb.ToString() ?? "").Trim();
            if (string.IsNullOrEmpty(ct)) { return true; }
            // Prefer Yes / 是; avoid 否 / No / 取消 / Cancel.
            bool isNo = ct.Contains("否") || ct.Equals("No", StringComparison.OrdinalIgnoreCase)
                || ct.StartsWith("No(", StringComparison.OrdinalIgnoreCase)
                || ct.StartsWith("No(&", StringComparison.OrdinalIgnoreCase);
            bool isCancel = ct.Contains("取消") || ct.StartsWith("Cancel", StringComparison.OrdinalIgnoreCase);
            if (isNo || isCancel) { return true; }
            bool isYes = ct == "是" || ct.StartsWith("是(") || ct.StartsWith("是(&")
                || ct.Equals("Yes", StringComparison.OrdinalIgnoreCase)
                || ct.StartsWith("Yes(", StringComparison.OrdinalIgnoreCase)
                || ct.StartsWith("Yes(&", StringComparison.OrdinalIgnoreCase);
            if (!isYes) { return true; }
            found = child;
            foundText = ct;
            return false;
        }, IntPtr.Zero);
        if (found == IntPtr.Zero) { return false; }
        SendMessage(found, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
        clickedText = foundText;
        return true;
    }

    public static List<string> ConfirmByKeyword(string keyword, bool includeHidden, uint[] ownerPids, string matchMode) {
        var matched = new List<string>();
        HashSet<uint> pidSet = null;
        if (ownerPids != null && ownerPids.Length > 0) {
            pidSet = new HashSet<uint>(ownerPids);
        }
        EnumWindows((hWnd, lParam) => {
            if (pidSet != null) {
                uint pid;
                GetWindowThreadProcessId(hWnd, out pid);
                if (!pidSet.Contains(pid)) { return true; }
            }
            if (!WindowMatchesKeyword(hWnd, keyword, matchMode)) { return true; }
            bool visible = IsWindowVisible(hWnd);
            bool iconic = IsIconic(hWnd);
            if (!visible && !includeHidden) { return true; }
            if (iconic || !visible) { ShowWindow(hWnd, SW_RESTORE); }
            string clickHow = "enter";
            string buttonText = "";
            string clicked;
            if (TryClickConfirmButton(hWnd, out clicked)) {
                clickHow = "button";
                buttonText = clicked ?? "";
            } else {
                PostMessage(hWnd, WM_KEYDOWN, (IntPtr)VK_RETURN, IntPtr.Zero);
            }
            StringBuilder tsb = new StringBuilder(256);
            GetWindowText(hWnd, tsb, tsb.Capacity);
            string title = tsb.ToString();
            string content = GetAggregatedWindowText(hWnd);
            string label = string.IsNullOrEmpty(title) ? "(no_title)" : title;
            matched.Add(label + "|content=" + content + "|confirm=" + clickHow
                + (string.IsNullOrEmpty(buttonText) ? "" : ("|button=" + buttonText))
                + (iconic ? "|minimized=1" : ""));
            return true;
        }, IntPtr.Zero);
        return matched;
    }
}
"@
}

function Get-SemaphorePopupOwnerPids {
  param([string]$ProcessName)
  if ([string]::IsNullOrWhiteSpace($ProcessName)) { return @() }
  $pids = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue | ForEach-Object { [uint32]$_.Id })
  return $pids
}

function Invoke-SemaphorePopupConfirm {
  param(
    [string]$Keyword,
    [bool]$IncludeHidden = $false,
    [string]$ProcessName = '',
    [string]$MatchMode = 'title_or_content'
  )
  $pids = @(Get-SemaphorePopupOwnerPids -ProcessName $ProcessName)
  $owner = if ($pids.Count -gt 0) { $pids } else { $null }
  $list = [SemaphorePopupHelper]::ConfirmByKeyword($Keyword, $IncludeHidden, $owner, $MatchMode)
  if ($list.Count -eq 0 -and -not $IncludeHidden) {
    $list = [SemaphorePopupHelper]::ConfirmByKeyword($Keyword, $true, $owner, $MatchMode)
  }
  return ,$list
}
