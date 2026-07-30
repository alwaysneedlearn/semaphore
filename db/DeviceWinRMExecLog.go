package db

import (
	"strings"
	"time"
)

const (
	DeviceWinRMCredentialModeWinRM = "winrm"
	DeviceWinRMCredentialModeRDP   = "rdp"

	DeviceWinRMShellPowerShell = "powershell"
	DeviceWinRMShellCmd        = "cmd"

	DeviceWinRMExecMaxCommandLen   = 8192
	DeviceWinRMExecMaxStoredOutput = 65536
	DeviceWinRMExecMaxResponseOut  = 262144

	// DeviceAuditLogRetainLimit is how many newest WinRM / RDP launch rows to keep per device.
	DeviceAuditLogRetainLimit = 10
)

// DeviceWinRMExecLog stores one WinRM command execution for audit and UI history.
type DeviceWinRMExecLog struct {
	ID               int       `db:"id" json:"id"`
	ProjectID        int       `db:"project_id" json:"project_id"`
	DeviceID         int       `db:"device_id" json:"device_id"`
	UserID           int       `db:"user_id" json:"user_id"`
	Username         string    `db:"username" json:"username"`
	CredentialMode   string    `db:"credential_mode" json:"credential_mode"`
	Shell            string    `db:"shell" json:"shell"`
	Command          string    `db:"command" json:"command"`
	OK               bool      `db:"ok" json:"ok"`
	ExitCode         *int      `db:"exit_code" json:"exit_code,omitempty"`
	ErrorCode        *string   `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage     *string   `db:"error_message" json:"error_message,omitempty"`
	Stdout           string    `db:"stdout" json:"stdout"`
	Stderr           string    `db:"stderr" json:"stderr"`
	OutputTruncated  bool      `db:"output_truncated" json:"output_truncated"`
	DurationMS       int       `db:"duration_ms" json:"duration_ms"`
	ResolvedHost     string    `db:"resolved_host" json:"resolved_host"`
	ResolvedPort     int       `db:"resolved_port" json:"resolved_port"`
	ResolvedUser     string    `db:"resolved_user" json:"resolved_user"`
	Created          time.Time `db:"created" json:"created"`
}

// DeviceWinRMExecLogList is a paginated list of execution logs for one device.
type DeviceWinRMExecLogList struct {
	Logs  []DeviceWinRMExecLog `json:"logs"`
	Total int                  `json:"total"`
}

// DeviceUsesWinRM reports whether the device is configured for WinRM automation (Windows hosts).
func DeviceUsesWinRM(device Device, settings ProjectDeviceSettings) bool {
	conn := device.AnsibleConnection
	if conn == "" {
		conn = settings.DefaultAnsibleConnection
	}
	if conn == "" {
		conn = "winrm"
	}
	return strings.EqualFold(strings.TrimSpace(conn), "winrm")
}

// TruncateDeviceWinRMExecOutput trims stored command output for persistence.
func TruncateDeviceWinRMExecOutput(stdout, stderr string) (outStdout, outStderr string, truncated bool) {
	outStdout, t1 := truncateUTF8At(stdout, DeviceWinRMExecMaxStoredOutput)
	outStderr, t2 := truncateUTF8At(stderr, DeviceWinRMExecMaxStoredOutput)
	return outStdout, outStderr, t1 || t2
}

func truncateUTF8At(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	return s[:max] + "\n…[truncated]", true
}
