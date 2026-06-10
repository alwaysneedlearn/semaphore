package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/masterzen/winrm"
	"github.com/semaphoreui/semaphore/db"
)

// DeviceWinRMExecRequest is input for a single remote command.
type DeviceWinRMExecRequest struct {
	Command        string
	Shell          string
	TimeoutSeconds int
}

// DeviceWinRMExecResult is the outcome of one remote command (API + audit).
type DeviceWinRMExecResult struct {
	OK              bool   `json:"ok"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	DurationMS      int    `json:"duration_ms"`
	ErrorCode       string `json:"error_code,omitempty"`
	ErrorMessage    string `json:"message,omitempty"`
	OutputTruncated bool   `json:"output_truncated"`
	ResolvedUser    string `json:"resolved_user"`
	ResolvedHost    string `json:"resolved_host"`
	ResolvedPort    int    `json:"resolved_port"`
}

// ExecDeviceWinRMCommand runs one command over WinRM with timeout and output limits.
func ExecDeviceWinRMCommand(ctx context.Context, creds DeviceWinRMExecCredentials, req DeviceWinRMExecRequest) DeviceWinRMExecResult {
	res := DeviceWinRMExecResult{
		ResolvedUser: creds.User,
		ResolvedHost: creds.Host,
		ResolvedPort: creds.Port,
	}

	command := strings.TrimSpace(req.Command)
	if command == "" {
		res.ErrorCode = "command_too_long"
		res.ErrorMessage = "command is required"
		return res
	}
	if len(command) > db.DeviceWinRMExecMaxCommandLen {
		res.ErrorCode = "command_too_long"
		res.ErrorMessage = fmt.Sprintf("command exceeds %d bytes", db.DeviceWinRMExecMaxCommandLen)
		return res
	}

	shell := strings.ToLower(strings.TrimSpace(req.Shell))
	if shell == "" {
		shell = db.DeviceWinRMShellPowerShell
	}
	if shell != db.DeviceWinRMShellPowerShell && shell != db.DeviceWinRMShellCmd {
		res.ErrorCode = "invalid_shell"
		res.ErrorMessage = "shell must be powershell or cmd"
		return res
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	if timeout > 120 {
		timeout = 120
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()

	endpoint := winrm.NewEndpoint(
		creds.Host,
		creds.Port,
		creds.UseHTTPS,
		creds.InsecureSkipVerify,
		nil, nil, nil,
		time.Duration(timeout)*time.Second,
	)

	params := winrm.DefaultParameters
	transport := strings.ToLower(strings.TrimSpace(creds.Transport))
	switch transport {
	case "ntlm", "credssp", "kerberos":
		params = &winrm.Parameters{
			TransportDecorator: func() winrm.Transporter { return &winrm.ClientNTLM{} },
		}
	default:
		params = winrm.DefaultParameters
	}

	client, err := winrm.NewClientWithParameters(endpoint, creds.User, creds.Password, params)
	if err != nil {
		res.ErrorCode = "winrm_unreachable"
		res.ErrorMessage = err.Error()
		res.DurationMS = int(time.Since(start).Milliseconds())
		return res
	}

	var stdout, stderr string
	var exitCode int
	switch shell {
	case db.DeviceWinRMShellCmd:
		stdout, stderr, exitCode, err = client.RunCmdWithContext(ctx, command)
	default:
		stdout, stderr, exitCode, err = client.RunPSWithContextWithString(ctx, command, "")
	}

	res.DurationMS = int(time.Since(start).Milliseconds())
	res.Stdout, res.Stderr, res.OutputTruncated = truncateResponseOutput(stdout, stderr)
	code := exitCode
	res.ExitCode = &code

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			res.ErrorCode = "command_timeout"
			res.ErrorMessage = "command timed out"
		} else if isWinRMAuthError(err) {
			res.ErrorCode = "winrm_auth_failed"
			res.ErrorMessage = sanitizeWinRMError(err)
		} else {
			res.ErrorCode = "winrm_unreachable"
			res.ErrorMessage = sanitizeWinRMError(err)
		}
		if res.Stderr == "" && res.ErrorMessage != "" {
			res.Stderr = res.ErrorMessage
		}
		return res
	}

	res.OK = exitCode == 0
	if !res.OK && res.ErrorCode == "" {
		res.ErrorCode = "command_failed"
		res.ErrorMessage = fmt.Sprintf("exit code %d", exitCode)
	}
	return res
}

func truncateResponseOutput(stdout, stderr string) (string, string, bool) {
	max := db.DeviceWinRMExecMaxResponseOut
	t1, t2 := false, false
	if len(stdout) > max {
		stdout = stdout[:max] + "\n…[truncated]"
		t1 = true
	}
	if len(stderr) > max {
		stderr = stderr[:max] + "\n…[truncated]"
		t2 = true
	}
	return stdout, stderr, t1 || t2
}

func isWinRMAuthError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "authentication")
}

func sanitizeWinRMError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 512 {
		msg = msg[:512] + "…"
	}
	return msg
}

// BuildDeviceWinRMExecLog creates an audit row from an execution result.
func BuildDeviceWinRMExecLog(
	projectID, deviceID, userID int,
	username, credentialMode, shell, command string,
	exec DeviceWinRMExecResult,
	created time.Time,
) db.DeviceWinRMExecLog {
	storeStdout, storeStderr, storedTruncated := db.TruncateDeviceWinRMExecOutput(exec.Stdout, exec.Stderr)
	var errCode *string
	if exec.ErrorCode != "" {
		errCode = &exec.ErrorCode
	}
	var errMsg *string
	if exec.ErrorMessage != "" {
		errMsg = &exec.ErrorMessage
	}
	return db.DeviceWinRMExecLog{
		ProjectID:       projectID,
		DeviceID:        deviceID,
		UserID:          userID,
		Username:        username,
		CredentialMode:  credentialMode,
		Shell:           shell,
		Command:         command,
		OK:              exec.OK,
		ExitCode:        exec.ExitCode,
		ErrorCode:       errCode,
		ErrorMessage:    errMsg,
		Stdout:          storeStdout,
		Stderr:          storeStderr,
		OutputTruncated: exec.OutputTruncated || storedTruncated,
		DurationMS:      exec.DurationMS,
		ResolvedHost:    exec.ResolvedHost,
		ResolvedPort:    exec.ResolvedPort,
		ResolvedUser:    exec.ResolvedUser,
		Created:         created,
	}
}
