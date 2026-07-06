# LAND GUI installer: launch installer exe and click through wizard buttons (interactive session).
param(
  [string]$ConfigFileArg = '',
  [string]$InstallerPathArg = '',
  [string]$LogFileArg = '',
  [string]$Dlg1TitleArg = 'LHBTS 安装',
  [string]$Dlg1ButtonArg = '升级',
  [string]$Dlg2TitleArg = '提示',
  [string]$Dlg2ButtonArg = '确定',
  [string]$Dlg3TitleArg = 'LHBTS 安装',
  [string]$Dlg3ButtonArg = '确定',
  [string]$StepTimeoutArg = '90',
  [string]$Dlg1CoordXPct = '32',
  [string]$Dlg1CoordYPct = '93',
  [string]$Dlg2CoordXPct = '0',
  [string]$Dlg2CoordYPct = '0',
  [string]$Dlg3CoordXPct = '83',
  [string]$Dlg3CoordYPct = '93'
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
  if ($null -ne $cfg.dlg1_coord_x_pct) { $script:Dlg1CoordXPct = [string]$cfg.dlg1_coord_x_pct }
  if ($null -ne $cfg.dlg1_coord_y_pct) { $script:Dlg1CoordYPct = [string]$cfg.dlg1_coord_y_pct }
  if ($null -ne $cfg.dlg2_coord_x_pct) { $script:Dlg2CoordXPct = [string]$cfg.dlg2_coord_x_pct }
  if ($null -ne $cfg.dlg2_coord_y_pct) { $script:Dlg2CoordYPct = [string]$cfg.dlg2_coord_y_pct }
  if ($null -ne $cfg.dlg3_coord_x_pct) { $script:Dlg3CoordXPct = [string]$cfg.dlg3_coord_x_pct }
  if ($null -ne $cfg.dlg3_coord_y_pct) { $script:Dlg3CoordYPct = [string]$cfg.dlg3_coord_y_pct }
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
    $Dlg1CoordXPct = $script:Dlg1CoordXPct
    $Dlg1CoordYPct = $script:Dlg1CoordYPct
    $Dlg2CoordXPct = $script:Dlg2CoordXPct
    $Dlg2CoordYPct = $script:Dlg2CoordYPct
    $Dlg3CoordXPct = $script:Dlg3CoordXPct
    $Dlg3CoordYPct = $script:Dlg3CoordYPct
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

$dlg1Title = $Dlg1TitleArg
$dlg1Button = $Dlg1ButtonArg
$dlg2Title = $Dlg2TitleArg
$dlg2Button = $Dlg2ButtonArg
$dlg3Title = $Dlg3TitleArg
$dlg3Button = $Dlg3ButtonArg

[int]$dlg1CoordX = 32
[int]$dlg1CoordY = 93
[int]$dlg2CoordX = 0
[int]$dlg2CoordY = 0
[int]$dlg3CoordX = 83
[int]$dlg3CoordY = 93
[int]::TryParse($Dlg1CoordXPct, [ref]$dlg1CoordX) | Out-Null
[int]::TryParse($Dlg1CoordYPct, [ref]$dlg1CoordY) | Out-Null
[int]::TryParse($Dlg2CoordXPct, [ref]$dlg2CoordX) | Out-Null
[int]::TryParse($Dlg2CoordYPct, [ref]$dlg2CoordY) | Out-Null
[int]::TryParse($Dlg3CoordXPct, [ref]$dlg3CoordX) | Out-Null
[int]::TryParse($Dlg3CoordYPct, [ref]$dlg3CoordY) | Out-Null

if (-not ([System.Management.Automation.PSTypeName]'SemaphoreLandGuiInstaller').Type) {
  try {
    Add-Type -TypeDefinition @"
using System;
using System.Collections.Generic;
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
    public static extern IntPtr SendMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);

    [DllImport("user32.dll")]
    public static extern bool GetClientRect(IntPtr hWnd, out RECT lpRect);

    [DllImport("user32.dll")]
    public static extern bool ClientToScreen(IntPtr hWnd, ref POINT lpPoint);

    [DllImport("user32.dll")]
    public static extern bool SetCursorPos(int X, int Y);

    [DllImport("user32.dll")]
    public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint lpdwProcessId);

    [DllImport("user32.dll", SetLastError = true)]
    public static extern IntPtr GetDlgItem(IntPtr hDlg, int nIDDlgItem);

    [StructLayout(LayoutKind.Sequential)]
    public struct RECT {
        public int Left;
        public int Top;
        public int Right;
        public int Bottom;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct POINT {
        public int X;
        public int Y;
    }

    public const uint BM_CLICK = 0x00F5;
    public const uint WM_KEYDOWN = 0x0100;
    public const uint WM_KEYUP = 0x0101;
    public const uint MOUSEEVENTF_LEFTDOWN = 0x0002;
    public const uint MOUSEEVENTF_LEFTUP = 0x0004;
    public const int VK_RETURN = 0x0D;
    public const int SW_RESTORE = 9;
    public const int IDOK = 1;
    public const int IDCANCEL = 2;

    public static uint InstallerProcessId = 0;
    public static int FallbackCoordXPct = 0;
    public static int FallbackCoordYPct = 0;

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

    public static string GetControlText(IntPtr hWnd) {
        if (hWnd == IntPtr.Zero) { return string.Empty; }
        StringBuilder sb = new StringBuilder(512);
        GetWindowText(hWnd, sb, sb.Capacity);
        return sb.ToString();
    }

    public static void ClickHwndCenter(IntPtr hWnd) {
        RECT r;
        if (!GetWindowRect(hWnd, out r)) { return; }
        int x = (r.Left + r.Right) / 2;
        int y = (r.Top + r.Bottom) / 2;
        SetCursorPos(x, y);
        mouse_event(MOUSEEVENTF_LEFTDOWN, 0, 0, 0, UIntPtr.Zero);
        mouse_event(MOUSEEVENTF_LEFTUP, 0, 0, 0, UIntPtr.Zero);
    }

    public static string ClickWindowAtPercent(IntPtr hWnd, int xPct, int yPct) {
        RECT cr;
        if (!GetClientRect(hWnd, out cr)) { return "coord_failed=no_client_rect"; }
        POINT pt = new POINT();
        pt.X = cr.Left + (cr.Right - cr.Left) * xPct / 100;
        pt.Y = cr.Top + (cr.Bottom - cr.Top) * yPct / 100;
        if (!ClientToScreen(hWnd, ref pt)) { return "coord_failed=client_to_screen"; }
        SetForegroundWindow(hWnd);
        SetCursorPos(pt.X, pt.Y);
        mouse_event(MOUSEEVENTF_LEFTDOWN, 0, 0, 0, UIntPtr.Zero);
        mouse_event(MOUSEEVENTF_LEFTUP, 0, 0, 0, UIntPtr.Zero);
        return "screen_x=" + pt.X + "|screen_y=" + pt.Y + "|client_x_pct=" + xPct + "|client_y_pct=" + yPct;
    }

    public static string GetClassNameText(IntPtr hWnd) {
        StringBuilder cls = new StringBuilder(64);
        GetClassName(hWnd, cls, cls.Capacity);
        return cls.ToString();
    }

    public static List<IntPtr> GetChildWindows(IntPtr parent) {
        var children = new List<IntPtr>();
        if (parent == IntPtr.Zero) { return children; }
        EnumChildWindows(parent, (child, lParam) => {
            children.Add(child);
            return true;
        }, IntPtr.Zero);
        return children;
    }

    public static IntPtr FindTopLevelWindowByTitle(string titlePart) {
        return FindWizardWindow(titlePart);
    }

    public static int CountDescendants(IntPtr parent) {
        int count = 0;
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0) {
            IntPtr hwnd = queue.Dequeue();
            List<IntPtr> kids = GetChildWindows(hwnd);
            count += kids.Count;
            foreach (IntPtr child in kids) {
                queue.Enqueue(child);
            }
        }
        return count;
    }

    public static IntPtr FindWizardWindow(string titlePart) {
        IntPtr best = IntPtr.Zero;
        int bestScore = -1;
        EnumWindows((hWnd, lParam) => {
            if (!IsWindowVisible(hWnd)) { return true; }
            string title = GetControlText(hWnd);
            if (!TitleMatches(title, titlePart)) { return true; }
            if (InstallerProcessId != 0) {
                uint pid;
                GetWindowThreadProcessId(hWnd, out pid);
                if (pid != InstallerProcessId) { return true; }
            }
            int score = CountDescendants(hWnd);
            if (score > bestScore) {
                bestScore = score;
                best = hWnd;
            }
            return true;
        }, IntPtr.Zero);
        if (best != IntPtr.Zero) { return best; }
        if (InstallerProcessId == 0) { return IntPtr.Zero; }
        bestScore = -1;
        EnumWindows((hWnd, lParam) => {
            if (!IsWindowVisible(hWnd)) { return true; }
            uint pid;
            GetWindowThreadProcessId(hWnd, out pid);
            if (pid != InstallerProcessId) { return true; }
            int score = CountDescendants(hWnd);
            if (score > bestScore) {
                bestScore = score;
                best = hWnd;
            }
            return true;
        }, IntPtr.Zero);
        return best;
    }

    public static bool IsCancelButtonText(string label) {
        if (string.IsNullOrEmpty(label)) { return false; }
        string norm = NormalizeButtonText(label);
        return norm == "取消" || norm.IndexOf("取消", StringComparison.Ordinal) >= 0
            || norm.Equals("Cancel", StringComparison.OrdinalIgnoreCase);
    }

    public static bool IsWarningDialog(IntPtr hwnd) {
        if (hwnd == IntPtr.Zero) { return false; }
        string title = GetControlText(hwnd);
        if (!string.IsNullOrEmpty(title) && title.IndexOf("警告", StringComparison.Ordinal) >= 0) {
            return true;
        }
        var queue = new Queue<IntPtr>();
        queue.Enqueue(hwnd);
        while (queue.Count > 0) {
            IntPtr node = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(node)) {
                string className = GetClassNameText(child);
                if (string.Equals(className, "Static", StringComparison.OrdinalIgnoreCase)) {
                    string body = GetControlText(child);
                    if (!string.IsNullOrEmpty(body) && body.IndexOf("警告", StringComparison.Ordinal) >= 0) {
                        return true;
                    }
                }
                queue.Enqueue(child);
            }
        }
        return false;
    }

    public static IntPtr FindStandardDialogButton(IntPtr hwnd, string buttonText) {
        if (hwnd == IntPtr.Zero || string.IsNullOrEmpty(buttonText)) { return IntPtr.Zero; }
        string norm = NormalizeButtonText(buttonText);
        int[] ids;
        if (norm == "确定" || norm == "确认" || norm.Equals("OK", StringComparison.OrdinalIgnoreCase)) {
            ids = new int[] { IDOK };
        } else if (IsCancelButtonText(buttonText)) {
            ids = new int[] { IDCANCEL };
        } else {
            return IntPtr.Zero;
        }
        foreach (int id in ids) {
            IntPtr item = GetDlgItem(hwnd, id);
            if (item == IntPtr.Zero || !IsWindowVisible(item)) { continue; }
            string label = GetControlText(item);
            if (string.IsNullOrEmpty(label) || TextMatchesButton(label, buttonText)) {
                return item;
            }
        }
        return IntPtr.Zero;
    }

    public static IntPtr FindButtonByTextDeep(IntPtr parent, string buttonText) {
        if (parent == IntPtr.Zero || string.IsNullOrEmpty(buttonText)) { return IntPtr.Zero; }
        IntPtr stdBtn = FindStandardDialogButton(parent, buttonText);
        if (stdBtn != IntPtr.Zero) { return stdBtn; }
        IntPtr exact = IntPtr.Zero;
        IntPtr partial = IntPtr.Zero;
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0) {
            IntPtr hwnd = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(hwnd)) {
                string className = GetClassNameText(child);
                string label = GetControlText(child);
                if (IsButtonClass(className) && IsWindowVisible(child)) {
                    string normLabel = NormalizeButtonText(label);
                    string normBtn = NormalizeButtonText(buttonText);
                    if (normLabel == normBtn) {
                        exact = child;
                    } else if (partial == IntPtr.Zero && TextMatchesButton(label, buttonText)) {
                        partial = child;
                    }
                }
                queue.Enqueue(child);
            }
        }
        if (exact != IntPtr.Zero) { return exact; }
        return partial;
    }

    public static IntPtr FindLastVisibleButton(IntPtr parent) {
        if (parent == IntPtr.Zero) { return IntPtr.Zero; }
        IntPtr last = IntPtr.Zero;
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0) {
            IntPtr hwnd = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(hwnd)) {
                string className = GetClassNameText(child);
                if (IsButtonClass(className) && IsWindowVisible(child)) {
                    last = child;
                }
                queue.Enqueue(child);
            }
        }
        return last;
    }

    public static IntPtr FindLastVisibleTNewButton(IntPtr parent) {
        return FindLastVisibleButton(parent);
    }

    public static string ScanWindowTree(IntPtr parent, int maxItems) {
        if (parent == IntPtr.Zero || maxItems <= 0) { return string.Empty; }
        StringBuilder sb = new StringBuilder();
        int count = 0;
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0 && count < maxItems) {
            IntPtr hwnd = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(hwnd)) {
                if (count >= maxItems) { break; }
                string className = GetClassNameText(child);
                string label = GetControlText(child);
                if (sb.Length > 0) { sb.Append(';'); }
                sb.Append(string.IsNullOrEmpty(label) ? "(empty)" : label);
                sb.Append('@');
                sb.Append(className);
                count++;
                queue.Enqueue(child);
            }
        }
        return sb.ToString();
    }

    public static string ScanClickableControls(IntPtr parent, int maxItems) {
        return ScanWindowTree(parent, maxItems);
    }

    public static void TryKeyboardReturn(IntPtr hwnd) {
        if (hwnd == IntPtr.Zero) { return; }
        SetForegroundWindow(hwnd);
        PostMessage(hwnd, WM_KEYDOWN, (IntPtr)VK_RETURN, IntPtr.Zero);
        PostMessage(hwnd, WM_KEYUP, (IntPtr)VK_RETURN, IntPtr.Zero);
    }

    public static bool ClickDialogButton(string titlePart, string buttonText, out string detail) {
        detail = string.Empty;
        try {
            IntPtr hwnd = FindWizardWindow(titlePart);
            if (hwnd == IntPtr.Zero) {
                detail = "window_not_found|installer_pid=" + InstallerProcessId;
                return false;
            }
            if (IsWarningDialog(hwnd)) {
                string warnTitle = GetControlText(hwnd);
                detail = "warning_dialog_abort|title=" + warnTitle;
                return false;
            }
            if (IsIconic(hwnd) || !IsWindowVisible(hwnd)) {
                ShowWindow(hwnd, SW_RESTORE);
            }
            SetForegroundWindow(hwnd);
            string wizardClass = GetClassNameText(hwnd);
            IntPtr btn = FindButtonByTextDeep(hwnd, buttonText);
            string clickMode = "text_match";
            if (btn == IntPtr.Zero) {
                string scan = ScanWindowTree(hwnd, 20);
                if (FallbackCoordXPct > 0 && FallbackCoordYPct > 0) {
                    string coordDetail = ClickWindowAtPercent(hwnd, FallbackCoordXPct, FallbackCoordYPct);
                    detail = "coord_click|wizard_class=" + wizardClass + "|" + coordDetail + "|scan=" + (string.IsNullOrEmpty(scan) ? "(empty_tree)" : scan);
                    return true;
                }
                TryKeyboardReturn(hwnd);
                detail = "button_not_found|wizard_class=" + wizardClass + "|installer_pid=" + InstallerProcessId + "|scan=" + (string.IsNullOrEmpty(scan) ? "(empty_tree)" : scan) + "|keyboard=VK_RETURN";
                return false;
            }
            string btnClass = GetClassNameText(btn);
            string btnLabel = GetControlText(btn);
            string normWanted = NormalizeButtonText(buttonText);
            string normGot = NormalizeButtonText(btnLabel);
            if ((normWanted == "确定" || normWanted == "确认") && IsCancelButtonText(btnLabel)) {
                detail = "wrong_button_abort|wanted=" + buttonText + "|got=" + btnLabel + "|wizard_class=" + wizardClass;
                return false;
            }
            if (normWanted.IndexOf("升级", StringComparison.Ordinal) >= 0 && IsCancelButtonText(btnLabel)) {
                detail = "wrong_button_abort|wanted=" + buttonText + "|got=" + btnLabel + "|wizard_class=" + wizardClass;
                return false;
            }
            SendMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
            PostMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
            ClickHwndCenter(btn);
            string title = GetControlText(hwnd);
            detail = "title=" + title + "|wizard_class=" + wizardClass + "|btn=" + btnLabel + "|class=" + btnClass + "|mode=" + clickMode + "|click=bm+mouse";
            return true;
        } catch (Exception ex) {
            detail = "exception=" + ex.GetType().Name + "|msg=" + ex.Message;
            return false;
        }
    }
}
"@ -ErrorAction Stop
  } catch {
    Write-InstallLine "INSTALL_ERROR|reason=add_type_failed|msg=$($_.Exception.Message)"
    exit 1
  }
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
    try {
      $hwnd = [SemaphoreLandGuiInstaller]::FindTopLevelWindowByTitle($TitlePart)
    } catch {
      Write-InstallLine "DIALOG_ERROR|phase=wait_visible|title_part=$TitlePart|msg=$($_.Exception.Message)"
      return $false
    }
    if ($hwnd -ne [IntPtr]::Zero) {
      Write-InstallLine "DIALOG_VISIBLE|title_part=$TitlePart|attempt=$attempt"
      return $true
    }
    if (($attempt % 50) -eq 0) {
      Write-InstallLine "DIALOG_POLL|title_part=$TitlePart|attempt=$attempt"
    }
    [System.Threading.Thread]::Sleep(0)
  } while ((Get-Date) -lt $deadline)
  Write-InstallLine "DIALOG_WAIT_TIMEOUT|title_part=$TitlePart|attempts=$attempt"
  return $false
}

function Wait-Click-LandDialog {
  param(
    [string]$TitlePart,
    [string]$ButtonText,
    [int]$TimeoutSec,
    [int]$CoordXPct = 0,
    [int]$CoordYPct = 0
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSec)
  $attempt = 0
  do {
    $attempt++
    $detail = ''
    $ok = $false
    [SemaphoreLandGuiInstaller]::FallbackCoordXPct = $CoordXPct
    [SemaphoreLandGuiInstaller]::FallbackCoordYPct = $CoordYPct
    try {
      $ok = [SemaphoreLandGuiInstaller]::ClickDialogButton($TitlePart, $ButtonText, [ref]$detail)
    } catch {
      $detail = "ps_exception=$($_.Exception.Message)"
      $ok = $false
    }
    if ($ok) {
      Write-InstallLine "DIALOG_CLICKED|title_part=$TitlePart|button=$ButtonText|attempt=$attempt|$detail"
      return $true
    }
    if ($detail -match 'warning_dialog_abort|wrong_button_abort') {
      Write-InstallLine "DIALOG_ABORT|title_part=$TitlePart|button=$ButtonText|attempt=$attempt|$detail"
      return $false
    }
    if (($attempt % 50) -eq 0) {
      Write-InstallLine "DIALOG_POLL|title_part=$TitlePart|button=$ButtonText|attempt=$attempt|last=$detail"
    }
    [System.Threading.Thread]::Sleep(0)
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
    $script:landInstallerPid = if ($proc) { [int]$proc.Id } else { 0 }
    if ($script:landInstallerPid -eq 0) {
      $procName = [System.IO.Path]::GetFileName($LiteralPath)
      $pidDeadline = (Get-Date).AddSeconds(10)
      do {
        $cim = Get-CimInstance Win32_Process -Filter "Name='$procName'" -ErrorAction SilentlyContinue |
          Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $LiteralPath) } |
          Sort-Object CreationDate -Descending |
          Select-Object -First 1
        if ($cim) {
          $script:landInstallerPid = [int]$cim.ProcessId
          break
        }
        [System.Threading.Thread]::Sleep(0)
      } while ((Get-Date) -lt $pidDeadline)
    }
    [SemaphoreLandGuiInstaller]::InstallerProcessId = [uint32]$script:landInstallerPid
    Write-InstallLine "INSTALLER_LAUNCHED|exe=$LiteralPath|pid=$($script:landInstallerPid)|mode=UseShellExecute"
    return $true
  } catch {
    Write-InstallLine "INSTALL_ERROR|reason=launch_failed|msg=$($_.Exception.Message)"
    return $false
  }
}

function Set-LandInstallerProcessId {
  param([string]$LiteralPath)
  $script:landInstallerPid = 0
  $procName = [System.IO.Path]::GetFileName($LiteralPath)
  $cim = Get-CimInstance Win32_Process -Filter "Name='$procName'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $LiteralPath) } |
    Sort-Object CreationDate -Descending |
    Select-Object -First 1
  if ($cim) { $script:landInstallerPid = [int]$cim.ProcessId }
  [SemaphoreLandGuiInstaller]::InstallerProcessId = [uint32]$script:landInstallerPid
  Write-InstallLine "INSTALLER_PID_RESOLVED|pid=$($script:landInstallerPid)|path=$LiteralPath"
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

$script:landInstallerPid = 0

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
  Set-LandInstallerProcessId -LiteralPath $installerPath
}

if (-not (Wait-Click-LandDialog -TitlePart $dlg1Title -ButtonText $dlg1Button -TimeoutSec $stepTimeout -CoordXPct $dlg1CoordX -CoordYPct $dlg1CoordY)) {
  Write-InstallLine 'INSTALL_FAILED|step=1'
  exit 1
}

if (-not (Wait-Click-LandDialog -TitlePart $dlg2Title -ButtonText $dlg2Button -TimeoutSec $stepTimeout -CoordXPct $dlg2CoordX -CoordYPct $dlg2CoordY)) {
  Write-InstallLine 'INSTALL_FAILED|step=2'
  exit 1
}

Write-InstallLine "INSTALL_WAIT_DIALOG|title_part=$dlg3Title|mode=poll"
if (-not (Wait-Click-LandDialog -TitlePart $dlg3Title -ButtonText $dlg3Button -TimeoutSec $stepTimeout -CoordXPct $dlg3CoordX -CoordYPct $dlg3CoordY)) {
  Write-InstallLine 'INSTALL_FAILED|step=3'
  exit 1
}

Write-InstallLine 'INSTALL_COMPLETE'
exit 0
