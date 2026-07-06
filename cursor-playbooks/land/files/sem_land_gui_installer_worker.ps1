# LAND GUI installer: launch installer exe and click through wizard buttons.
# Env:
#   LAND_INSTALLER_EXE_PATH (required)
#   LAND_INSTALL_DLG1_TITLE / LAND_INSTALL_DLG1_BUTTON (default LHBTS 安装 / 升级)
#   LAND_INSTALL_DLG2_TITLE / LAND_INSTALL_DLG2_BUTTON (default 提示 / 确认)
#   LAND_INSTALL_DLG3_TITLE / LAND_INSTALL_DLG3_BUTTON (default LHBTS 安装 / 确定)
#   LAND_INSTALL_STEP_TIMEOUT_SECONDS (default 90 per dialog; step 3 waits for dialog, no fixed sleep)
#   LogFileArg (optional)

param(
  [string]$LogFileArg = ''
)

function Write-InstallLine {
  param([string]$Line)
  $ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
  $out = "[$ts] $Line"
  Write-Output $out
  if (-not [string]::IsNullOrWhiteSpace($LogFileArg)) {
    Add-Content -LiteralPath $LogFileArg -Value $out -Encoding UTF8 -ErrorAction SilentlyContinue
  }
}

$installerPath = [string]$env:LAND_INSTALLER_EXE_PATH
if ($null -eq $installerPath) { $installerPath = '' }
$installerPath = $installerPath.Trim()
if ($installerPath.Length -eq 0 -or -not (Test-Path -LiteralPath $installerPath)) {
  Write-InstallLine "INSTALL_ERROR|reason=installer_exe_missing|path=$installerPath"
  exit 1
}

[int]$stepTimeout = 90
$stepTimeoutRaw = [string]$env:LAND_INSTALL_STEP_TIMEOUT_SECONDS
if (-not [string]::IsNullOrWhiteSpace($stepTimeoutRaw)) {
  [int]::TryParse($stepTimeoutRaw, [ref]$stepTimeout) | Out-Null
}
if ($stepTimeout -lt 10) { $stepTimeout = 10 }

[int]$settleMs = 400
$settleRaw = [string]$env:LAND_INSTALL_CLICK_SETTLE_MS
if (-not [string]::IsNullOrWhiteSpace($settleRaw)) {
  [int]::TryParse($settleRaw, [ref]$settleMs) | Out-Null
}
if ($settleMs -lt 0) { $settleMs = 0 }

function Get-EnvOrDefault {
  param([string]$Name, [string]$Default)
  $v = [string]$env:$Name
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

if (-not ([System.Management.Automation.PSTypeName]'SemaphoreLandGuiInstaller').Type) {
  Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;

public class SemaphoreLandGuiInstaller {
    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumChildWindows(IntPtr hWndParent, EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern int GetClassName(IntPtr hWnd, StringBuilder lpClassName, int nMaxCount);

    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsIconic(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool SendMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    public const uint BM_CLICK = 0x00F5;
    public const int SW_RESTORE = 9;

    public static bool TitleMatches(string title, string titlePart) {
        if (string.IsNullOrEmpty(title) || string.IsNullOrEmpty(titlePart)) { return false; }
        return title.IndexOf(titlePart, StringComparison.Ordinal) >= 0;
    }

    public static IntPtr FindTopLevelWindowByTitle(string titlePart) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((hWnd, lParam) => {
            StringBuilder sb = new StringBuilder(512);
            GetWindowText(hWnd, sb, sb.Capacity);
            string title = sb.ToString();
            if (TitleMatches(title, titlePart)) {
                found = hWnd;
                return false;
            }
            return true;
        }, IntPtr.Zero);
        return found;
    }

    public static IntPtr FindButtonByText(IntPtr parent, string buttonText) {
        if (parent == IntPtr.Zero || string.IsNullOrEmpty(buttonText)) { return IntPtr.Zero; }
        IntPtr found = IntPtr.Zero;
        EnumChildWindows(parent, (hWnd, lParam) => {
            StringBuilder cls = new StringBuilder(64);
            GetClassName(hWnd, cls, cls.Capacity);
            string className = cls.ToString();
            if (!string.Equals(className, "Button", StringComparison.OrdinalIgnoreCase)) {
                return true;
            }
            StringBuilder txt = new StringBuilder(256);
            GetWindowText(hWnd, txt, txt.Capacity);
            string label = txt.ToString();
            if (label == buttonText || label.IndexOf(buttonText, StringComparison.Ordinal) >= 0) {
                found = hWnd;
                return false;
            }
            return true;
        }, IntPtr.Zero);
        return found;
    }

    public static bool ClickDialogButton(string titlePart, string buttonText, out string detail) {
        detail = string.Empty;
        IntPtr hwnd = FindTopLevelWindowByTitle(titlePart);
        if (hwnd == IntPtr.Zero) {
            detail = "window_not_found";
            return false;
        }
        if (IsIconic(hwnd) || !IsWindowVisible(hwnd)) {
            ShowWindow(hwnd, SW_RESTORE);
        }
        SetForegroundWindow(hwnd);
        IntPtr btn = FindButtonByText(hwnd, buttonText);
        if (btn == IntPtr.Zero) {
            detail = "button_not_found";
            return false;
        }
        SendMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
        StringBuilder tsb = new StringBuilder(512);
        GetWindowText(hwnd, tsb, tsb.Capacity);
        detail = "title=" + tsb.ToString();
        return true;
    }
}
"@
}

function Wait-For-LandDialog {
  param(
    [string]$TitlePart,
    [int]$TimeoutSec
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  $attempt = 0
  do {
    $attempt++
    $hwnd = [SemaphoreLandGuiInstaller]::FindTopLevelWindowByTitle($TitlePart)
    if ($hwnd -ne [IntPtr]::Zero) {
      Write-InstallLine "DIALOG_VISIBLE|title_part=$TitlePart|attempt=$attempt"
      return $true
    }
    Start-Sleep -Milliseconds 400
  } while ((Get-Date) -lt $deadline)
  Write-InstallLine "DIALOG_WAIT_TIMEOUT|title_part=$TitlePart|attempts=$attempt"
  return $false
}

function Wait-Click-LandDialog {
  param(
    [string]$TitlePart,
    [string]$ButtonText,
    [int]$TimeoutSec
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  $attempt = 0
  do {
    $attempt++
    $detail = ''
    $ok = [SemaphoreLandGuiInstaller]::ClickDialogButton($TitlePart, $ButtonText, [ref]$detail)
    if ($ok) {
      Write-InstallLine "DIALOG_CLICKED|title_part=$TitlePart|button=$ButtonText|attempt=$attempt|$detail"
      return $true
    }
    Start-Sleep -Milliseconds 500
  } while ((Get-Date) -lt $deadline)
  Write-InstallLine "DIALOG_TIMEOUT|title_part=$TitlePart|button=$ButtonText|attempts=$attempt|last=$detail"
  return $false
}

$workDir = [System.IO.Path]::GetDirectoryName($installerPath)
if ([string]::IsNullOrWhiteSpace($workDir)) {
  $workDir = (Get-Location).Path
}

Write-InstallLine "INSTALL_START|path=$installerPath|workdir=$workDir"

$procName = [System.IO.Path]::GetFileNameWithoutExtension($installerPath)
$running = @(Get-Process -Name $procName -ErrorAction SilentlyContinue)
if ($running.Count -eq 0) {
  try {
    Start-Process -FilePath $installerPath -WorkingDirectory $workDir | Out-Null
    Write-InstallLine "INSTALLER_LAUNCHED|exe=$installerPath"
    if (-not (Wait-For-LandDialog -TitlePart $dlg1Title -TimeoutSec $stepTimeout)) {
      Write-InstallLine 'INSTALL_FAILED|step=launch_wait_dlg1'
      exit 1
    }
  } catch {
    Write-InstallLine "INSTALL_ERROR|reason=launch_failed|msg=$($_.Exception.Message)"
    exit 1
  }
} else {
  Write-InstallLine "INSTALLER_ALREADY_RUNNING|proc=$procName|count=$($running.Count)"
}

if (-not (Wait-Click-LandDialog -TitlePart $dlg1Title -ButtonText $dlg1Button -TimeoutSec $stepTimeout)) {
  Write-InstallLine 'INSTALL_FAILED|step=1'
  exit 1
}
if ($settleMs -gt 0) { Start-Sleep -Milliseconds $settleMs }

if (-not (Wait-Click-LandDialog -TitlePart $dlg2Title -ButtonText $dlg2Button -TimeoutSec $stepTimeout)) {
  Write-InstallLine 'INSTALL_FAILED|step=2'
  exit 1
}
if ($settleMs -gt 0) { Start-Sleep -Milliseconds $settleMs }

Write-InstallLine "INSTALL_WAIT_DIALOG|title_part=$dlg3Title|mode=poll"
if (-not (Wait-Click-LandDialog -TitlePart $dlg3Title -ButtonText $dlg3Button -TimeoutSec $stepTimeout)) {
  Write-InstallLine 'INSTALL_FAILED|step=3'
  exit 1
}

Write-InstallLine 'INSTALL_COMPLETE'
exit 0
