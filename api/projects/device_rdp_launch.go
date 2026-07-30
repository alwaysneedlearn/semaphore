package projects

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/random"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

const (
	rdpLaunchTokenTTL       = 90 * time.Second
	rdpLaunchTokenLength    = 48
	rdpLifecycleTokenTTL    = 12 * time.Hour
	rdpLifecycleTokenLength = 48

	rdpLaunchEventMstscStarted = "mstsc_started"
	rdpLaunchEventMstscExited  = "mstsc_exited"
)

// rdpLaunchTokenEntry is a one-time launch grant bound to user + project + device.
type rdpLaunchTokenEntry struct {
	UserID    int
	ProjectID int
	DeviceID  int
	LogID     int
	ExpiresAt time.Time
}

// rdpLifecycleTokenEntry authorizes Helper callbacks for mstsc start/exit on one launch log.
type rdpLifecycleTokenEntry struct {
	UserID    int
	ProjectID int
	DeviceID  int
	LogID     int
	ExpiresAt time.Time
}

var rdpLaunchTokenStore sync.Map     // token string → rdpLaunchTokenEntry
var rdpLifecycleTokenStore sync.Map // token string → rdpLifecycleTokenEntry

func createRDPLaunchToken(userID, projectID, deviceID, logID int) (token string, expiresIn int) {
	token = random.String(rdpLaunchTokenLength)
	rdpLaunchTokenStore.Store(token, rdpLaunchTokenEntry{
		UserID:    userID,
		ProjectID: projectID,
		DeviceID:  deviceID,
		LogID:     logID,
		ExpiresAt: time.Now().Add(rdpLaunchTokenTTL),
	})
	return token, int(rdpLaunchTokenTTL.Seconds())
}

// consumeRDPLaunchToken loads and deletes a one-time token. Returns false if missing, already used, or expired.
func consumeRDPLaunchToken(token string) (rdpLaunchTokenEntry, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return rdpLaunchTokenEntry{}, false
	}
	v, ok := rdpLaunchTokenStore.LoadAndDelete(token)
	if !ok {
		return rdpLaunchTokenEntry{}, false
	}
	entry := v.(rdpLaunchTokenEntry)
	if !time.Now().Before(entry.ExpiresAt) {
		return rdpLaunchTokenEntry{}, false
	}
	return entry, true
}

func createRDPLifecycleToken(userID, projectID, deviceID, logID int) (token string, expiresIn int) {
	token = random.String(rdpLifecycleTokenLength)
	rdpLifecycleTokenStore.Store(token, rdpLifecycleTokenEntry{
		UserID:    userID,
		ProjectID: projectID,
		DeviceID:  deviceID,
		LogID:     logID,
		ExpiresAt: time.Now().Add(rdpLifecycleTokenTTL),
	})
	return token, int(rdpLifecycleTokenTTL.Seconds())
}

func lookupRDPLifecycleToken(token string) (rdpLifecycleTokenEntry, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return rdpLifecycleTokenEntry{}, false
	}
	v, ok := rdpLifecycleTokenStore.Load(token)
	if !ok {
		return rdpLifecycleTokenEntry{}, false
	}
	entry := v.(rdpLifecycleTokenEntry)
	if !time.Now().Before(entry.ExpiresAt) {
		rdpLifecycleTokenStore.Delete(token)
		return rdpLifecycleTokenEntry{}, false
	}
	return entry, true
}

func deleteRDPLifecycleToken(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	rdpLifecycleTokenStore.Delete(token)
}

func resetRDPLaunchTokenStoreForTest() {
	rdpLaunchTokenStore.Range(func(key, _ any) bool {
		rdpLaunchTokenStore.Delete(key)
		return true
	})
	rdpLifecycleTokenStore.Range(func(key, _ any) bool {
		rdpLifecycleTokenStore.Delete(key)
		return true
	})
}

func requestClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// LaunchDeviceRDP issues a one-time token for the Native RDP Helper protocol.
// POST /api/project/{project_id}/devices/{device_id}/rdp/launch
func LaunchDeviceRDP(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	device := helpers.GetFromContext(r, "device").(db.Device)
	user := helpers.UserFromContext(r)

	port := device.RDPPort
	if port == 0 {
		port = db.DefaultDeviceRDPPort
	}

	logRow, err := helpers.Store(r).CreateDeviceRDPLaunchLog(db.DeviceRDPLaunchLog{
		ProjectID: project.ID,
		DeviceID:  device.ID,
		UserID:    user.ID,
		Username:  user.Username,
		Phase:     db.DeviceRDPLaunchPhaseRequested,
		Host:      device.IPAddress,
		RDPPort:   port,
		RDPUser:   device.RDPUser,
		ClientIP:  requestClientIP(r),
		Created:   tz.Now(),
	})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogCreate, helpers.EventLogItem{
		UserID:      user.ID,
		ProjectID:   project.ID,
		ObjectType:  db.EventDevice,
		ObjectID:    device.ID,
		Description: fmt.Sprintf("RDP launch requested for device %s (%s)", device.Hostname, device.IPAddress),
	})

	token, expiresIn := createRDPLaunchToken(user.ID, project.ID, device.ID, logRow.ID)
	helperURL := "semaphore-rdp://connect?token=" + url.QueryEscape(token)

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_in": expiresIn,
		"helper_url": helperURL,
		"log_id":     logRow.ID,
		"user_id":    user.ID,
		"username":   user.Username,
		"host":       device.IPAddress,
		"hostname":   device.Hostname,
		"rdp_port":   port,
		"rdp_user":   device.RDPUser,
	})
}

// GetRDPLaunchParams exchanges a one-time token for RDP connection parameters.
// GET /api/rdp/launch-params?token=... — no session cookie; token-only auth.
func GetRDPLaunchParams(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	entry, ok := consumeRDPLaunchToken(token)
	if !ok {
		helpers.WriteErrorStatus(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	device, err := helpers.Store(r).GetDevice(entry.ProjectID, entry.DeviceID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	port := device.RDPPort
	if port == 0 {
		port = db.DefaultDeviceRDPPort
	}

	if entry.LogID > 0 {
		_ = helpers.Store(r).MarkDeviceRDPLaunchHelperFetched(
			entry.ProjectID, entry.DeviceID, entry.LogID, tz.Now(),
		)
		desc := fmt.Sprintf("RDP helper fetched params for device %s (%s)", device.Hostname, device.IPAddress)
		helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
			UserID:      entry.UserID,
			ProjectID:   entry.ProjectID,
			ObjectType:  db.EventDevice,
			ObjectID:    entry.DeviceID,
			Description: desc,
		})
	}

	password := strings.TrimSpace(device.RDPPassword)
	var rdpPassword *string
	passwordProvided := false
	if password != "" {
		rdpPassword = &password
		passwordProvided = true
	}

	lifecycleToken := ""
	lifecycleExpiresIn := 0
	if entry.LogID > 0 {
		lifecycleToken, lifecycleExpiresIn = createRDPLifecycleToken(
			entry.UserID, entry.ProjectID, entry.DeviceID, entry.LogID,
		)
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id":            entry.ProjectID,
		"device_id":             entry.DeviceID,
		"log_id":                entry.LogID,
		"host":                  device.IPAddress,
		"rdp_port":              port,
		"rdp_user":              device.RDPUser,
		"rdp_password":          rdpPassword,
		"password_provided":     passwordProvided,
		"lifecycle_token":       lifecycleToken,
		"lifecycle_expires_in":  lifecycleExpiresIn,
	})
}

type rdpLaunchEventRequest struct {
	Token string `json:"token"`
	Event string `json:"event"`
}

// PostRDPLaunchEvent records Helper mstsc lifecycle callbacks (start / exit).
// POST /api/rdp/launch-events — token-only auth (lifecycle_token from launch-params).
func PostRDPLaunchEvent(w http.ResponseWriter, r *http.Request) {
	var req rdpLaunchEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteErrorStatus(w, "invalid json body", http.StatusBadRequest)
		return
	}
	event := strings.TrimSpace(strings.ToLower(req.Event))
	if event != rdpLaunchEventMstscStarted && event != rdpLaunchEventMstscExited {
		helpers.WriteErrorStatus(w, "event must be mstsc_started or mstsc_exited", http.StatusBadRequest)
		return
	}

	entry, ok := lookupRDPLifecycleToken(req.Token)
	if !ok {
		helpers.WriteErrorStatus(w, "invalid or expired lifecycle token", http.StatusUnauthorized)
		return
	}
	if entry.LogID <= 0 {
		helpers.WriteErrorStatus(w, "lifecycle token missing log id", http.StatusBadRequest)
		return
	}

	store := helpers.Store(r)
	now := tz.Now()
	var err error
	switch event {
	case rdpLaunchEventMstscStarted:
		err = store.MarkDeviceRDPLaunchMstscStarted(entry.ProjectID, entry.DeviceID, entry.LogID, now)
	case rdpLaunchEventMstscExited:
		err = store.MarkDeviceRDPLaunchMstscExited(entry.ProjectID, entry.DeviceID, entry.LogID, now)
		deleteRDPLifecycleToken(req.Token)
	}
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	device, derr := store.GetDevice(entry.ProjectID, entry.DeviceID)
	hostLabel := fmt.Sprintf("device_id=%d", entry.DeviceID)
	if derr == nil {
		hostLabel = fmt.Sprintf("%s (%s)", device.Hostname, device.IPAddress)
	}
	desc := fmt.Sprintf("RDP %s for device %s", event, hostLabel)
	helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
		UserID:      entry.UserID,
		ProjectID:   entry.ProjectID,
		ObjectType:  db.EventDevice,
		ObjectID:    entry.DeviceID,
		Description: desc,
	})

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"event":  event,
		"log_id": entry.LogID,
	})
}

// GetDeviceRDPLaunchLogs lists remote-desktop launch history for one device.
func GetDeviceRDPLaunchLogs(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := helpers.Store(r).GetDeviceRDPLaunchLogs(device.ProjectID, device.ID, limit, offset)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, list)
}
