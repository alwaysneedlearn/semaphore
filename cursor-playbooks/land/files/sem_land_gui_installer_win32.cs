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

    public const uint BM_CLICK = 0x00F5;
    public const uint WM_GETTEXT = 0x000D;
    public const uint WM_GETTEXTLENGTH = 0x000E;
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

    public static string GetControlText(IntPtr hWnd) {
        if (hWnd == IntPtr.Zero) { return string.Empty; }
        StringBuilder sb = new StringBuilder(256);
        GetWindowText(hWnd, sb, sb.Capacity);
        string text = sb.ToString();
        if (!string.IsNullOrEmpty(text)) { return text; }
        int len = (int)SendMessage(hWnd, WM_GETTEXTLENGTH, IntPtr.Zero, IntPtr.Zero);
        if (len <= 0) { return string.Empty; }
        StringBuilder sb2 = new StringBuilder(len + 2);
        SendMessage(hWnd, WM_GETTEXT, (IntPtr)(len + 1), sb2);
        return sb2.ToString();
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
        IntPtr found = IntPtr.Zero;
        EnumWindows((hWnd, lParam) => {
            if (!IsWindowVisible(hWnd)) { return true; }
            string title = GetControlText(hWnd);
            if (TitleMatches(title, titlePart)) {
                found = hWnd;
                return false;
            }
            return true;
        }, IntPtr.Zero);
        return found;
    }

    public static IntPtr FindButtonByTextDeep(IntPtr parent, string buttonText) {
        if (parent == IntPtr.Zero || string.IsNullOrEmpty(buttonText)) { return IntPtr.Zero; }
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0) {
            IntPtr hwnd = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(hwnd)) {
                string className = GetClassNameText(child);
                string label = GetControlText(child);
                if (IsButtonClass(className) && TextMatchesButton(label, buttonText)) {
                    return child;
                }
                queue.Enqueue(child);
            }
        }
        return IntPtr.Zero;
    }

    public static IntPtr FindLastVisibleTNewButton(IntPtr parent) {
        if (parent == IntPtr.Zero) { return IntPtr.Zero; }
        IntPtr last = IntPtr.Zero;
        var queue = new Queue<IntPtr>();
        queue.Enqueue(parent);
        while (queue.Count > 0) {
            IntPtr hwnd = queue.Dequeue();
            foreach (IntPtr child in GetChildWindows(hwnd)) {
                string className = GetClassNameText(child);
                if (string.Equals(className, "TNewButton", StringComparison.OrdinalIgnoreCase) && IsWindowVisible(child)) {
                    last = child;
                }
                queue.Enqueue(child);
            }
        }
        return last;
    }

    public static string ScanClickableControls(IntPtr parent, int maxItems) {
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
                if (!string.IsNullOrEmpty(label) && (IsButtonClass(className) || label.Length <= 32)) {
                    if (sb.Length > 0) { sb.Append(';'); }
                    sb.Append(label);
                    sb.Append('@');
                    sb.Append(className);
                    count++;
                }
                queue.Enqueue(child);
            }
        }
        return sb.ToString();
    }

    public static bool ClickDialogButton(string titlePart, string buttonText, out string detail) {
        detail = string.Empty;
        try {
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
            string btnClass = GetClassNameText(btn);
            string btnLabel = GetControlText(btn);
            SendMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
            PostMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
            string title = GetControlText(hwnd);
            detail = "title=" + title + "|btn=" + btnLabel + "|class=" + btnClass + "|mode=" + clickMode;
            return true;
        } catch (Exception ex) {
            detail = "exception=" + ex.GetType().Name + "|msg=" + ex.Message;
            return false;
        }
    }
}
