package projects

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
	"github.com/semaphoreui/semaphore/services/server"
)

type deviceWinRMExecBody struct {
	CredentialMode string `json:"credential_mode"`
	Command        string `json:"command"`
	Shell          string `json:"shell"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	ForceOffline   bool   `json:"force_offline"`
}

type deviceWinRMExecLogsDeleteBody struct {
	IDs []int `json:"ids"`
}

// GetDeviceWinRMConnectionPreview returns resolved connection summary without secrets.
func GetDeviceWinRMConnectionPreview(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(device.ProjectID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	mode := strings.TrimSpace(r.URL.Query().Get("credential_mode"))
	if mode == "" {
		mode = db.DeviceWinRMCredentialModeWinRM
	}
	creds, err := server.ResolveDeviceWinRMExecCredentials(device, settings, mode)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, server.PreviewDeviceWinRMConnection(creds))
}

// ExecDeviceWinRMCommand runs one WinRM command and persists an audit log row.
func ExecDeviceWinRMCommand(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	user := helpers.UserFromContext(r)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(device.ProjectID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if !db.DeviceUsesWinRM(device, settings) {
		helpers.WriteErrorStatus(w, "WinRM console is only available for Windows (winrm) devices", http.StatusBadRequest)
		return
	}

	var body deviceWinRMExecBody
	if !helpers.Bind(w, r, &body) {
		return
	}

	if !body.ForceOffline {
		if _, err := runDeviceProbeAndReload(r, &device, settings); err != nil {
			helpers.WriteError(w, err)
			return
		}
		if device.WinRMStatus == db.DeviceStatusOffline {
			helpers.WriteJSON(w, http.StatusConflict, map[string]any{
				"ok":    false,
				"error": "winrm_unreachable",
				"message": "WinRM port is offline; probe again or set force_offline to retry",
			})
			return
		}
	}

	creds, err := server.ResolveDeviceWinRMExecCredentials(device, settings, body.CredentialMode)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	shell := strings.TrimSpace(body.Shell)
	if shell == "" {
		shell = db.DeviceWinRMShellPowerShell
	}

	execResult := server.ExecDeviceWinRMCommand(r.Context(), creds, server.DeviceWinRMExecRequest{
		Command:        body.Command,
		Shell:          shell,
		TimeoutSeconds: body.TimeoutSeconds,
	})

	now := tz.Now()
	logRow := server.BuildDeviceWinRMExecLog(
		device.ProjectID, device.ID, user.ID, user.Username,
		creds.Mode, shell, strings.TrimSpace(body.Command),
		execResult, now,
	)
	saved, err := helpers.Store(r).CreateDeviceWinRMExecLog(logRow)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	out := map[string]any{
		"log_id":           saved.ID,
		"ok":               execResult.OK,
		"exit_code":        execResult.ExitCode,
		"stdout":           execResult.Stdout,
		"stderr":           execResult.Stderr,
		"duration_ms":      execResult.DurationMS,
		"output_truncated": execResult.OutputTruncated,
		"resolved_user":    execResult.ResolvedUser,
		"resolved_host":    execResult.ResolvedHost,
		"resolved_port":    execResult.ResolvedPort,
	}
	if execResult.ErrorCode != "" {
		out["error"] = execResult.ErrorCode
		out["message"] = execResult.ErrorMessage
	}
	status := http.StatusOK
	if execResult.ErrorCode != "" && !execResult.OK {
		status = http.StatusBadGateway
	}
	helpers.WriteJSON(w, status, out)
}

func runDeviceProbeAndReload(r *http.Request, device *db.Device, settings db.ProjectDeviceSettings) (db.Device, error) {
	rdp, winrm, api, refreshed := server.ProbeDevice(*device, settings)
	if err := helpers.Store(r).UpdateDevicePortProbeStatuses(
		device.ProjectID, device.ID, rdp, winrm, api, refreshed,
	); err != nil {
		return *device, err
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	device.APIStatus = api
	normalizeDeviceStatuses(device)
	device.LastUpdated = &refreshed
	return *device, nil
}

// GetDeviceWinRMExecLogs lists persisted execution history for one device.
func GetDeviceWinRMExecLogs(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := helpers.Store(r).GetDeviceWinRMExecLogs(device.ProjectID, device.ID, limit, offset)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, list)
}

// DeleteDeviceWinRMExecLog removes one execution log row.
func DeleteDeviceWinRMExecLog(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	logID, err := helpers.GetIntParam("log_id", w, r)
	if err != nil {
		return
	}
	if err := helpers.Store(r).DeleteDeviceWinRMExecLog(device.ProjectID, device.ID, logID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteDeviceWinRMExecLogsBatch removes multiple log rows for one device.
func DeleteDeviceWinRMExecLogsBatch(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	var body deviceWinRMExecLogsDeleteBody
	if !helpers.Bind(w, r, &body) {
		return
	}
	n, err := helpers.Store(r).DeleteDeviceWinRMExecLogs(device.ProjectID, device.ID, body.IDs)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"deleted": n})
}

// ClearDeviceWinRMExecLogs removes all execution logs for one device.
func ClearDeviceWinRMExecLogs(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	n, err := helpers.Store(r).ClearDeviceWinRMExecLogs(device.ProjectID, device.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"deleted": n})
}
