package projects

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/server"
)

// DeviceMiddleware ensures the device exists, belongs to the current project,
// and loads it into the request context under the key "device".
func DeviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project := helpers.GetFromContext(r, "project").(db.Project)
		deviceID, err := helpers.GetIntParam("device_id", w, r)
		if err != nil {
			return
		}

		device, err := helpers.Store(r).GetDevice(project.ID, deviceID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}

		r = helpers.SetContextValue(r, "device", device)
		next.ServeHTTP(w, r)
	})
}

// GetDevices returns the project's device list.
func GetDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	params := helpers.QueryParamsForProps(r.URL, db.DeviceProps)

	devices, err := helpers.Store(r).GetDevices(project.ID, params)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if devices == nil {
		devices = []db.Device{}
	}
	helpers.WriteJSON(w, http.StatusOK, devices)
}

// GetDevice returns one device (loaded by DeviceMiddleware).
func GetDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	helpers.WriteJSON(w, http.StatusOK, device)
}

// GetDeviceStatsHandler returns aggregate counts for the project's devices.
func GetDeviceStatsHandler(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	stats, err := helpers.Store(r).GetDeviceStats(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, stats)
}

// AddDevice creates a device under the current project.
func AddDevice(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)

	var device db.Device
	if !helpers.Bind(w, r, &device) {
		return
	}

	device.ProjectID = project.ID
	device.Name = strings.TrimSpace(device.Name)
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Hostname = strings.TrimSpace(device.Hostname)

	if err := device.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}

	newDevice, err := helpers.Store(r).CreateDevice(device)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogCreate, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   project.ID,
		ObjectType:  db.EventDevice,
		ObjectID:    newDevice.ID,
		Description: fmt.Sprintf("Device %s created", newDevice.Name),
	})

	helpers.WriteJSON(w, http.StatusCreated, newDevice)
}

// UpdateDevice persists changes to an existing device.
func UpdateDevice(w http.ResponseWriter, r *http.Request) {
	old := helpers.GetFromContext(r, "device").(db.Device)

	var device db.Device
	if !helpers.Bind(w, r, &device) {
		return
	}

	device.ID = old.ID
	device.ProjectID = old.ProjectID
	device.Name = strings.TrimSpace(device.Name)
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Hostname = strings.TrimSpace(device.Hostname)

	if err := device.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}

	if err := helpers.Store(r).UpdateDevice(device); err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   old.ProjectID,
		ObjectType:  db.EventDevice,
		ObjectID:    old.ID,
		Description: fmt.Sprintf("Device %s updated", device.Name),
	})

	w.WriteHeader(http.StatusNoContent)
}

// RemoveDevice deletes a device and (cascaded) its config items.
func RemoveDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	if err := helpers.Store(r).DeleteDevice(device.ProjectID, device.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogDelete, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   device.ProjectID,
		ObjectType:  db.EventDevice,
		ObjectID:    device.ID,
		Description: fmt.Sprintf("Device %s deleted", device.Name),
	})

	w.WriteHeader(http.StatusNoContent)
}

// GetDeviceConfig returns the categorized config items for a device.
func GetDeviceConfig(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	items, err := helpers.Store(r).GetDeviceConfigItems(device.ProjectID, device.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if items == nil {
		items = []db.DeviceConfigItem{}
	}
	helpers.WriteJSON(w, http.StatusOK, items)
}

// PutDeviceConfig replaces the device config items in a single transaction.
func PutDeviceConfig(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)

	var items []db.DeviceConfigItem
	if !helpers.Bind(w, r, &items) {
		return
	}

	for i := range items {
		items[i].DeviceID = device.ID
		items[i].Category = strings.TrimSpace(items[i].Category)
		items[i].Key = strings.TrimSpace(items[i].Key)
	}

	if err := helpers.Store(r).SetDeviceConfigItems(device.ProjectID, device.ID, items); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetDeviceSettings returns the per-project device action template bindings
// and the periodic refresh interval.
func GetDeviceSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	s, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, s)
}

// UpdateDeviceSettings persists per-project device action template bindings.
func UpdateDeviceSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)

	var s db.ProjectDeviceSettings
	if !helpers.Bind(w, r, &s) {
		return
	}
	s.ProjectID = project.ID

	if s.StatusRefreshIntervalMin < 0 {
		s.StatusRefreshIntervalMin = 0
	}

	if err := helpers.Store(r).UpdateProjectDeviceSettings(s); err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   project.ID,
		ObjectType:  db.EventProject,
		ObjectID:    project.ID,
		Description: "Device settings updated",
	})

	w.WriteHeader(http.StatusNoContent)
}

// runDeviceTemplate enqueues the template configured for the given action.
// extraVars is merged into the task's environment override (extra-vars JSON).
func runDeviceTemplate(r *http.Request, project db.Project, action db.DeviceAction, extraVars map[string]any) (db.Task, error) {
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		return db.Task{}, err
	}

	tplID := settings.TemplateIDForAction(action)
	if tplID == nil || *tplID == 0 {
		return db.Task{}, &db.ValidationError{
			Message: fmt.Sprintf("No template configured for action %q", action),
		}
	}

	tpl, err := helpers.Store(r).GetTemplate(project.ID, *tplID)
	if err != nil {
		return db.Task{}, err
	}

	env := ""
	if len(extraVars) > 0 {
		b, err := json.Marshal(extraVars)
		if err != nil {
			return db.Task{}, err
		}
		env = string(b)
	}

	task := db.Task{
		TemplateID:  tpl.ID,
		ProjectID:   project.ID,
		Environment: env,
	}

	user := helpers.UserFromContext(r)
	var userID *int
	username := ""
	if user != nil {
		userID = &user.ID
		username = user.Username
	}

	return taskPool(r).AddTask(task, userID, username, project.ID, tpl.App.NeedTaskAlias())
}

// DiscoverDevices triggers the project's discover template (if configured).
func DiscoverDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)

	task, err := runDeviceTemplate(r, project, db.DeviceActionDiscover, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, task)
}

// RunDeviceAction triggers the configured template for {start, stop, restart,
// status, config} on a specific device.
func RunDeviceAction(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	device := helpers.GetFromContext(r, "device").(db.Device)

	var body struct {
		Action db.DeviceAction `json:"action"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}

	switch body.Action {
	case db.DeviceActionStart, db.DeviceActionStop, db.DeviceActionRestart,
		db.DeviceActionStatus, db.DeviceActionConfig:
	default:
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Unsupported device action",
		})
		return
	}

	extraVars := map[string]any{
		"device": map[string]any{
			"id":       device.ID,
			"name":     device.Name,
			"ip":       device.IPAddress,
			"hostname": device.Hostname,
		},
	}

	if body.Action == db.DeviceActionConfig {
		items, err := helpers.Store(r).GetDeviceConfigItems(project.ID, device.ID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		categorized := map[string]map[string]string{}
		for _, it := range items {
			cat := it.Category
			if cat == "" {
				cat = "default"
			}
			if categorized[cat] == nil {
				categorized[cat] = map[string]string{}
			}
			categorized[cat][it.Key] = it.Value
		}
		extraVars["config"] = categorized
	}

	task, err := runDeviceTemplate(r, project, body.Action, extraVars)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, task)
}

// ProbeDevice runs an immediate server-side TCP port probe of RDP and WinRM
// for one device and persists the result. Useful for instant feedback when
// no status template is configured.
func ProbeDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	rdp, winrm, refreshed := server.ProbeDevice(device)
	if err := helpers.Store(r).UpdateDeviceStatus(
		device.ProjectID, device.ID, rdp, winrm, refreshed,
	); err != nil {
		helpers.WriteError(w, err)
		return
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	device.LastUpdated = &refreshed
	helpers.WriteJSON(w, http.StatusOK, device)
}
