package projects

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
	"github.com/semaphoreui/semaphore/services/server"
)

func resolveMonitorWinRM(w http.ResponseWriter, r *http.Request, forceOffline bool) (db.Device, server.DeviceWinRMExecCredentials, bool) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(device.ProjectID)
	if err != nil {
		helpers.WriteError(w, err)
		return device, server.DeviceWinRMExecCredentials{}, false
	}
	if !db.DeviceUsesWinRM(device, settings) {
		helpers.WriteErrorStatus(w, "Device monitor is only available for Windows (winrm) devices", http.StatusBadRequest)
		return device, server.DeviceWinRMExecCredentials{}, false
	}
	if !forceOffline {
		if _, err := runDeviceProbeAndReload(r, &device, settings); err != nil {
			helpers.WriteError(w, err)
			return device, server.DeviceWinRMExecCredentials{}, false
		}
		if device.WinRMStatus == db.DeviceStatusOffline {
			helpers.WriteJSON(w, http.StatusConflict, map[string]any{
				"ok":      false,
				"error":   "winrm_unreachable",
				"message": "WinRM port is offline; probe again or set force_offline=1 to retry",
			})
			return device, server.DeviceWinRMExecCredentials{}, false
		}
	}
	mode := strings.TrimSpace(r.URL.Query().Get("credential_mode"))
	if mode == "" {
		mode = db.DeviceWinRMCredentialModeWinRM
	}
	creds, err := server.ResolveDeviceWinRMExecCredentials(device, settings, mode)
	if err != nil {
		helpers.WriteError(w, err)
		return device, server.DeviceWinRMExecCredentials{}, false
	}
	return device, creds, true
}

func queryForceOffline(r *http.Request) bool {
	v := strings.TrimSpace(r.URL.Query().Get("force_offline"))
	return v == "1" || strings.EqualFold(v, "true")
}

func auditMonitorAction(r *http.Request, device db.Device, creds server.DeviceWinRMExecCredentials, command string, ok bool, durationMS int, errCode, errMsg string) {
	user := helpers.UserFromContext(r)
	now := tz.Now()
	execRes := server.DeviceWinRMExecResult{
		OK:           ok,
		DurationMS:   durationMS,
		ErrorCode:    errCode,
		ErrorMessage: errMsg,
		ResolvedUser: creds.User,
		ResolvedHost: creds.Host,
		ResolvedPort: creds.Port,
	}
	if !ok && errCode == "" {
		execRes.ErrorCode = "monitor_failed"
	}
	logRow := server.BuildDeviceWinRMExecLog(
		device.ProjectID, device.ID, user.ID, user.Username,
		creds.Mode, db.DeviceWinRMShellPowerShell, command,
		execRes, now,
	)
	_, _ = helpers.Store(r).CreateDeviceWinRMExecLog(logRow)
}

// GetDeviceMonitorMetrics returns CPU / memory / disk utilization via WinRM.
func GetDeviceMonitorMetrics(w http.ResponseWriter, r *http.Request) {
	device, creds, ok := resolveMonitorWinRM(w, r, queryForceOffline(r))
	if !ok {
		return
	}
	metrics := server.GetDeviceMonitorMetrics(r.Context(), creds)
	auditMonitorAction(r, device, creds, "[monitor] metrics", metrics.OK, metrics.DurationMS, metrics.ErrorCode, metrics.ErrorMessage)
	status := http.StatusOK
	if !metrics.OK {
		status = http.StatusBadGateway
	}
	helpers.WriteJSON(w, status, metrics)
}

// GetDeviceMonitorFSList lists a directory (or C:/D: roots) with page size 10.
func GetDeviceMonitorFSList(w http.ResponseWriter, r *http.Request) {
	device, creds, ok := resolveMonitorWinRM(w, r, queryForceOffline(r))
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	rawPath := r.URL.Query().Get("path")
	list := server.ListDeviceMonitorFS(r.Context(), creds, rawPath, page)
	cmd := fmt.Sprintf("[monitor] list path=%q page=%d", list.Path, page)
	auditMonitorAction(r, device, creds, cmd, list.OK, list.DurationMS, list.ErrorCode, list.ErrorMessage)

	status := http.StatusOK
	if !list.OK {
		switch list.ErrorCode {
		case "invalid_path", "not_directory", "network_drive":
			status = http.StatusBadRequest
		case "not_found":
			status = http.StatusNotFound
		default:
			status = http.StatusBadGateway
		}
	}
	helpers.WriteJSON(w, status, list)
}

// flushResponseWriter flushes after each Write so browsers can report download progress.
type flushResponseWriter struct {
	http.ResponseWriter
}

func (f flushResponseWriter) Write(p []byte) (int, error) {
	n, err := f.ResponseWriter.Write(p)
	if fl, ok := f.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
	return n, err
}

// DownloadDeviceMonitorFile streams a single file (≤20MB) to the client.
func DownloadDeviceMonitorFile(w http.ResponseWriter, r *http.Request) {
	device, creds, ok := resolveMonitorWinRM(w, r, queryForceOffline(r))
	if !ok {
		return
	}
	rawPath := r.URL.Query().Get("path")
	started := tz.Now()
	headersSent := false
	meta, err := server.DownloadDeviceMonitorFile(
		r.Context(),
		creds,
		rawPath,
		func(m server.DeviceMonitorDownloadMeta) error {
			name := server.SafeDownloadFilename(m.Name)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Length", strconv.FormatInt(m.SizeBytes, 10))
			w.Header().Set("Content-Disposition", fmt.Sprintf(
				"attachment; filename=%q; filename*=UTF-8''%s",
				name, url.PathEscape(name),
			))
			w.WriteHeader(http.StatusOK)
			headersSent = true
			if fl, ok := w.(http.Flusher); ok {
				fl.Flush()
			}
			return nil
		},
		flushResponseWriter{ResponseWriter: w},
	)
	durationMS := int(tz.Now().Sub(started).Milliseconds())
	if err != nil {
		msg := err.Error()
		code := "download_failed"
		status := http.StatusBadGateway
		if ve, ok := err.(*db.ValidationError); ok {
			msg = ve.Message
			code = "invalid_request"
			status = http.StatusBadRequest
			lower := strings.ToLower(msg)
			if strings.Contains(lower, "exceed") {
				code = "too_large"
			}
			if strings.Contains(lower, "not found") {
				status = http.StatusNotFound
				code = "not_found"
			}
			if strings.Contains(lower, "network") || strings.Contains(lower, "shared") {
				code = "network_drive"
			}
		} else if strings.Contains(strings.ToLower(msg), "network/shared") {
			code = "network_drive"
			status = http.StatusBadRequest
		}
		auditMonitorAction(r, device, creds, fmt.Sprintf("[monitor] download path=%q", rawPath), false, durationMS, code, msg)
		if !headersSent {
			helpers.WriteJSON(w, status, map[string]any{
				"ok":      false,
				"error":   code,
				"message": msg,
			})
		}
		return
	}
	auditMonitorAction(r, device, creds,
		fmt.Sprintf("[monitor] download path=%q size=%d name=%q", meta.Path, meta.SizeBytes, meta.Name),
		true, durationMS, "", "")
}
