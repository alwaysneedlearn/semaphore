# LAND GUI installer: launch installer exe and click through wizard buttons (interactive session).
param(
  [string]$ConfigFileArg = '',
  [string]$InstallerPathArg = '',
  [string]$LogFileArg = '',
  [string]$Dlg1TitleArg = 'LHBTS 安装',
  [string]$Dlg1ButtonArg = '升级',
  [string]$Dlg2TitleArg = '提示',
  [string]$Dlg2ButtonArg = '确认',
  [string]$Dlg3TitleArg = 'LHBTS 安装',
  [string]$Dlg3ButtonArg = '确定',
  [string]$StepTimeoutArg = '90',
  [string]$ClickSettleMsArg = '400'
)

function Write-InstallLine {
  param([string]$Line)
  $ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
  $out = "[$ts] $Line"
  Write-Output $out
  if (-not [string]::IsNullOrWhiteSpace($script:LogFileArg)) {
    try {
      Add-Content -LiteralPath $script:LogFileArg -Value $out -Encoding UTF8 -ErrorAction Stop
    } catch {
      Write-Output "INSTALL_LOG_APPEND_FAILED|path=$script:LogFileArg|msg=$($_.Exception.Message)"
    }
  }
}

function Initialize-LandInstallFromConfig {
  param([string]$Path)
  if ([string]::IsNullOrWhiteSpace($Path)) { return $false }
  if (-not (Test-Path -LiteralPath $Path)) { return $false }
  try {
    $raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8 -ErrorAction Stop
    $cfg = $raw | ConvertFrom-Json -ErrorAction Stop
  } catch {
    return $false
  }
  if ($cfg.log_file) { $script:LogFileArg = [string]$cfg.log_file }
  if ($cfg.installer_path) { $script:InstallerPathArg = [string]$cfg.installer_path }
  if ($cfg.dlg1_title) { $script:Dlg1TitleArg = [string]$cfg.dlg1_title }
  if ($cfg.dlg1_button) { $script:Dlg1ButtonArg = [string]$cfg.dlg1_button }
  if ($cfg.dlg2_title) { $script:Dlg2TitleArg = [string]$cfg.dlg2_title }
  if ($cfg.dlg2_button) { $script:Dlg2ButtonArg = [string]$cfg.dlg2_button }
  if ($cfg.dlg3_title) { $script:Dlg3TitleArg = [string]$cfg.dlg3_title }
  if ($cfg.dlg3_button) { $script:Dlg3ButtonArg = [string]$cfg.dlg3_button }
  if ($cfg.step_timeout_seconds) { $script:StepTimeoutArg = [string]$cfg.step_timeout_seconds }
  if ($cfg.click_settle_ms) { $script:ClickSettleMsArg = [string]$cfg.click_settle_ms }
  return $true
}

if (-not [string]::IsNullOrWhiteSpace($ConfigFileArg)) {
  if (Initialize-LandInstallFromConfig -Path $ConfigFileArg) {
    $InstallerPathArg = $script:InstallerPathArg
    $LogFileArg = $script:LogFileArg
    $Dlg1TitleArg = $script:Dlg1TitleArg
    $Dlg1ButtonArg = $script:Dlg1ButtonArg
    $Dlg2TitleArg = $script:Dlg2TitleArg
    $Dlg2ButtonArg = $script:Dlg2ButtonArg
    $Dlg3TitleArg = $script:Dlg3TitleArg
    $Dlg3ButtonArg = $script:Dlg3ButtonArg
    $StepTimeoutArg = $script:StepTimeoutArg
    $ClickSettleMsArg = $script:ClickSettleMsArg
  }
}

if (-not [string]::IsNullOrWhiteSpace($script:LogFileArg)) {
  try {
    $boot = "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] INSTALL_WORKER_BOOT|config=$ConfigFileArg|user=$([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"
    if (Test-Path -LiteralPath $script:LogFileArg) {
      Add-Content -LiteralPath $script:LogFileArg -Value $boot -Encoding UTF8 -ErrorAction Stop
    } else {
      $boot | Out-File -LiteralPath $script:LogFileArg -Encoding UTF8 -Force -ErrorAction Stop
    }
  } catch {
    Write-Output "INSTALL_WORKER_BOOT_FAILED|msg=$($_.Exception.Message)"
  }
}

$installerPath = $InstallerPathArg
if ([string]::IsNullOrWhiteSpace($installerPath)) {
  $installerPath = [string]$env:LAND_INSTALLER_EXE_PATH
}
if ($null -eq $installerPath) { $installerPath = '' }
$installerPath = $installerPath.Trim()

if ($installerPath.Length -eq 0 -or -not (Test-Path -LiteralPath $installerPath)) {
  Write-InstallLine "INSTALL_ERROR|reason=installer_exe_missing|path=$installerPath"
  exit 1
}

[int]$stepTimeout = 90
[int]::TryParse($StepTimeoutArg, [ref]$stepTimeout) | Out-Null
if ($stepTimeout -lt 10) { $stepTimeout = 10 }

[int]$settleMs = 400
[int]::TryParse($ClickSettleMsArg, [ref]$settleMs) | Out-Null
if ($settleMs -lt 0) { $settleMs = 0 }

$dlg1Title = $Dlg1TitleArg
$dlg1Button = $Dlg1ButtonArg
$dlg2Title = $Dlg2TitleArg
$dlg2Button = $Dlg2ButtonArg
$dlg3Title = $Dlg3TitleArg
$dlg3Button = $Dlg3ButtonArg

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

    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    public const uint BM_CLICK = 0x00F5;
    public const int SW_RESTORE = 9;

    public static bool TitleMatches(string title, string titlePart) {
        if (string.IsNullOrEmpty(title) || string.IsNullOrEmpty(titlePart)) { return false; }
        return title.IndexOf(titlePart, StringComparison.Ordinal) >= 0;
    }

    public static string NormalizeButtonText(string text) {
        if (string.IsNullOrEmpty(text)) { return string.Empty; }
        return text.Replace("&", string.Empty).Trim();
    }

    public static bool TextMatchesButton(string label, string buttonText) {
        if (string.IsNullOrEmpty(buttonText)) { return false; }
        if (string.IsNullOrEmpty(label)) { return false; }
        string normLabel = NormalizeButtonText(label);
        string normBtn = NormalizeButtonText(buttonText);
        if (normLabel == normBtn) { return true; }
        if (normLabel.IndexOf(normBtn, StringComparison.Ordinal) >= 0) { return true; }
        if (label == buttonText) { return true; }
        if (label.IndexOf(buttonText, StringComparison.Ordinal) >= 0) { return true; }
        return false;
    }

    public static bool IsButtonClass(string className) {
        if (string.IsNullOrEmpty(className)) { return false; }
        if (string.Equals(className, "Button", StringComparison.OrdinalIgnoreCase)) { return true; }
        if (string.Equals(className, "SysButton", StringComparison.OrdinalIgnoreCase)) { return true; }
        if (string.Equals(className, "TButton", StringComparison.OrdinalIgnoreCase)) { return true; }
        if (string.Equals(className, "TNewButton", StringComparison.OrdinalIgnoreCase)) { return true; }
        if (className.EndsWith("Button", StringComparison.OrdinalIgnoreCase)) { return true; }
        return false;
    }

    public static IntPtr FindTopLevelWindowByTitle(string titlePart) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((hWnd, lParam) => {
            if (!IsWindowVisible(hWnd)) { return true; }
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

    private static void FindButtonDeepWorker(IntPtr hwnd, string buttonText, ref IntPtr found) {
        if (found != IntPtr.Zero || hwnd == IntPtr.Zero) { return; }
        EnumChildWindows(hwnd, (child, lParam) => {
            if (found != IntPtr.Zero) { return false; }
            StringBuilder cls = new StringBuilder(64);
            GetClassName(child, cls, cls.Capacity);
            string className = cls.ToString();
            StringBuilder txt = new StringBuilder(256);
            GetWindowText(child, txt, txt.Capacity);
            string label = txt.ToString();
            if (IsButtonClass(className) && TextMatchesButton(label, buttonText)) {
                found = child;
                return false;
            }
            FindButtonDeepWorker(child, buttonText, ref found);
            return found == IntPtr.Zero;
        }, IntPtr.Zero);
    }

    public static IntPtr FindButtonByTextDeep(IntPtr parent, string buttonText) {
        IntPtr found = IntPtr.Zero;
        FindButtonDeepWorker(parent, buttonText, ref found);
        return found;
    }

    private static void FindLastTNewButtonWorker(IntPtr hwnd, ref IntPtr last) {
        if (hwnd == IntPtr.Zero) { return; }
        EnumChildWindows(hwnd, (child, lParam) => {
            StringBuilder cls = new StringBuilder(64);
            GetClassName(child, cls, cls.Capacity);
            string className = cls.ToString();
            if (string.Equals(className, "TNewButton", StringComparison.OrdinalIgnoreCase) && IsWindowVisible(child)) {
                last = child;
            }
            FindLastTNewButtonWorker(child, ref last);
            return true;
        }, IntPtr.Zero);
    }

    public static IntPtr FindLastVisibleTNewButton(IntPtr parent) {
        IntPtr last = IntPtr.Zero;
        FindLastTNewButtonWorker(parent, ref last);
        return last;
    }

    public static string ScanClickableControls(IntPtr parent, int maxItems) {
        if (parent == IntPtr.Zero || maxItems <= 0) { return string.Empty; }
        StringBuilder sb = new StringBuilder();
        int count = 0;
        ScanClickableWorker(parent, sb, ref count, maxItems);
        return sb.ToString();
    }

    private static void ScanClickableWorker(IntPtr hwnd, StringBuilder sb, ref int count, int maxItems) {
        if (hwnd == IntPtr.Zero || count >= maxItems) { return; }
        EnumChildWindows(hwnd, (child, lParam) => {
            if (count >= maxItems) { return false; }
            StringBuilder cls = new StringBuilder(64);
            GetClassName(child, cls, cls.Capacity);
            string className = cls.ToString();
            StringBuilder txt = new StringBuilder(256);
            GetWindowText(child, txt, txt.Capacity);
            string label = txt.ToString();
            if (!string.IsNullOrEmpty(label) && (IsButtonClass(className) || label.Length <= 32)) {
                if (sb.Length > 0) { sb.Append(';'); }
                sb.Append(label);
                sb.Append('@');
                sb.Append(className);
                count++;
            }
            ScanClickableWorker(child, sb, ref count, maxItems);
            return true;
        }, IntPtr.Zero);
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
        IntPtr btn = FindButtonByTextDeep(hwnd, buttonText);
        string clickMode = "text_match";
        if (btn == IntPtr.Zero) {
            btn = FindLastVisibleTNewButton(hwnd);
            if (btn != IntPtr.Zero) {
                clickMode = "fallback_last_TNewButton";
            }
        }
        if (btn == IntPtr.Zero) {
            string scan = ScanClickableControls(hwnd, 12);
            detail = string.IsNullOrEmpty(scan) ? "button_not_found" : ("button_not_found|scan=" + scan);
            return false;
        }
        StringBuilder bcls = new StringBuilder(64);
        GetClassName(btn, bcls, bcls.Capacity);
        StringBuilder blbl = new StringBuilder(256);
        GetWindowText(btn, blbl, blbl.Capacity);
        SendMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
        PostMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
        StringBuilder tsb = new StringBuilder(512);
        GetWindowText(hwnd, tsb, tsb.Capacity);
        detail = "title=" + tsb.ToString() + "|btn=" + blbl.ToString() + "|class=" + bcls.ToString() + "|mode=" + clickMode;
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
    if (($attempt % 10) -eq 0) {
      Write-InstallLine "DIALOG_POLL|title_part=$TitlePart|attempt=$attempt"
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
    if (($attempt % 10) -eq 0) {
      Write-InstallLine "DIALOG_POLL|title_part=$TitlePart|button=$ButtonText|attempt=$attempt|last=$detail"
    }
    Start-Sleep -Milliseconds 500
  } while ((Get-Date) -lt $deadline)
  Write-InstallLine "DIALOG_TIMEOUT|title_part=$TitlePart|button=$ButtonText|attempts=$attempt|last=$detail"
  return $false
}

function Start-LandInstallerGui {
  param([string]$LiteralPath, [string]$WorkDir)
  try {
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $LiteralPath
    $psi.WorkingDirectory = $WorkDir
    $psi.UseShellExecute = $true
    $psi.WindowStyle = [System.Diagnostics.ProcessWindowStyle]::Normal
    $proc = [System.Diagnostics.Process]::Start($psi)
    $procId = if ($proc) { $proc.Id } else { 0 }
    Write-InstallLine "INSTALLER_LAUNCHED|exe=$LiteralPath|pid=$procId|mode=UseShellExecute"
    return $true
  } catch {
    Write-InstallLine "INSTALL_ERROR|reason=launch_failed|msg=$($_.Exception.Message)"
    return $false
  }
}

function Test-InstallerProcessRunning {
  param([string]$LiteralPath)
  $name = [System.IO.Path]::GetFileName($LiteralPath)
  $cim = Get-CimInstance Win32_Process -Filter "Name='$name'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $LiteralPath) }
  return @($cim).Count -gt 0
}

$workDir = [System.IO.Path]::GetDirectoryName($installerPath)
if ([string]::IsNullOrWhiteSpace($workDir)) {
  $workDir = (Get-Location).Path
}

Write-InstallLine "INSTALL_START|path=$installerPath|workdir=$workDir|session=interactive"

if (-not (Test-InstallerProcessRunning -LiteralPath $installerPath)) {
  if (-not (Start-LandInstallerGui -LiteralPath $installerPath -WorkDir $workDir)) {
    exit 1
  }
  if (-not (Wait-For-LandDialog -TitlePart $dlg1Title -TimeoutSec $stepTimeout)) {
    Write-InstallLine 'INSTALL_FAILED|step=launch_wait_dlg1'
    exit 1
  }
} else {
  Write-InstallLine "INSTALLER_ALREADY_RUNNING|path=$installerPath"
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
