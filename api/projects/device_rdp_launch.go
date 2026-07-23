package projects

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/random"
)

const (
	rdpLaunchTokenTTL    = 90 * time.Second
	rdpLaunchTokenLength = 48
)

// rdpLaunchTokenEntry is a one-time launch grant bound to user + project + device.
type rdpLaunchTokenEntry struct {
	UserID    int
	ProjectID int
	DeviceID  int
	ExpiresAt time.Time
}

var rdpLaunchTokenStore sync.Map // token string → rdpLaunchTokenEntry

func createRDPLaunchToken(userID, projectID, deviceID int) (token string, expiresIn int) {
	token = random.String(rdpLaunchTokenLength)
	rdpLaunchTokenStore.Store(token, rdpLaunchTokenEntry{
		UserID:    userID,
		ProjectID: projectID,
		DeviceID:  deviceID,
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

func resetRDPLaunchTokenStoreForTest() {
	rdpLaunchTokenStore.Range(func(key, _ any) bool {
		rdpLaunchTokenStore.Delete(key)
		return true
	})
}

// LaunchDeviceRDP issues a one-time token for the Native RDP Helper protocol.
// POST /api/project/{project_id}/devices/{device_id}/rdp/launch
func LaunchDeviceRDP(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	device := helpers.GetFromContext(r, "device").(db.Device)
	user := helpers.UserFromContext(r)

	token, expiresIn := createRDPLaunchToken(user.ID, project.ID, device.ID)
	helperURL := "semaphore-rdp://connect?token=" + url.QueryEscape(token)

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_in": expiresIn,
		"helper_url": helperURL,
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

	password := strings.TrimSpace(device.RDPPassword)
	var rdpPassword *string
	passwordProvided := false
	if password != "" {
		rdpPassword = &password
		passwordProvided = true
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"project_id":        entry.ProjectID,
		"device_id":         entry.DeviceID,
		"host":              device.IPAddress,
		"rdp_port":          port,
		"rdp_user":          device.RDPUser,
		"rdp_password":      rdpPassword,
		"password_provided": passwordProvided,
	})
}
