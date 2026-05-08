package projects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/server"
)

const (
	deviceAutoInventoryName  = "windows_hosts (auto)"
	deviceAutoInventoryGroup = "windows_hosts"
)

func normalizeDeviceConnection(device *db.Device, settings db.ProjectDeviceSettings) {
	if device.AnsibleUser == "" {
		device.AnsibleUser = settings.DefaultAnsibleUser
	}
	if device.AnsiblePassword == "" {
		device.AnsiblePassword = settings.DefaultAnsiblePassword
	}
	if device.AnsibleConnection == "" {
		device.AnsibleConnection = settings.DefaultAnsibleConnection
	}
	if device.AnsibleConnection == "" {
		device.AnsibleConnection = "winrm"
	}
	if device.AnsibleWinRMTransport == "" {
		device.AnsibleWinRMTransport = settings.DefaultAnsibleWinRMTransport
	}
	if device.AnsibleWinRMTransport == "" {
		device.AnsibleWinRMTransport = "basic"
	}
	if device.AnsibleWinRMScheme == "" {
		device.AnsibleWinRMScheme = settings.DefaultAnsibleWinRMScheme
	}
	if device.AnsibleWinRMScheme == "" {
		device.AnsibleWinRMScheme = "http"
	}
	if device.AnsiblePort == 0 {
		device.AnsiblePort = settings.DefaultAnsiblePort
	}
	if device.AnsiblePort == 0 {
		device.AnsiblePort = 5985
	}
	if device.AnsibleWinRMServerCertValidation == "" {
		device.AnsibleWinRMServerCertValidation = settings.DefaultAnsibleWinRMServerCertValidation
	}
	if device.AnsibleWinRMServerCertValidation == "" {
		device.AnsibleWinRMServerCertValidation = "ignore"
	}
}

func buildInventoryLine(dev db.Device, settings db.ProjectDeviceSettings) string {
	user := dev.AnsibleUser
	if user == "" {
		user = settings.DefaultAnsibleUser
	}
	password := dev.AnsiblePassword
	if password == "" {
		password = settings.DefaultAnsiblePassword
	}
	connection := dev.AnsibleConnection
	if connection == "" {
		connection = settings.DefaultAnsibleConnection
	}
	if connection == "" {
		connection = "winrm"
	}
	transport := dev.AnsibleWinRMTransport
	if transport == "" {
		transport = settings.DefaultAnsibleWinRMTransport
	}
	if transport == "" {
		transport = "basic"
	}
	scheme := dev.AnsibleWinRMScheme
	if scheme == "" {
		scheme = settings.DefaultAnsibleWinRMScheme
	}
	if scheme == "" {
		scheme = "http"
	}
	port := dev.AnsiblePort
	if port == 0 {
		port = settings.DefaultAnsiblePort
	}
	if port == 0 {
		port = 5985
	}
	certValidation := dev.AnsibleWinRMServerCertValidation
	if certValidation == "" {
		certValidation = settings.DefaultAnsibleWinRMServerCertValidation
	}
	if certValidation == "" {
		certValidation = "ignore"
	}

	parts := []string{dev.Hostname}
	if dev.IPAddress != "" {
		parts = append(parts, "ansible_host="+dev.IPAddress)
	}
	if user != "" {
		parts = append(parts, "ansible_user="+user)
	}
	if password != "" {
		parts = append(parts, "ansible_password="+password)
	}
	parts = append(parts, "ansible_connection="+connection)
	parts = append(parts, "ansible_winrm_transport="+transport)
	parts = append(parts, "ansible_winrm_scheme="+scheme)
	parts = append(parts, "ansible_port="+strconv.Itoa(port))
	parts = append(parts, "ansible_winrm_server_cert_validation="+certValidation)
	return strings.Join(parts, " ")
}

func renderWindowsInventory(devices []db.Device, settings db.ProjectDeviceSettings) string {
	var b strings.Builder
	b.WriteString("[" + deviceAutoInventoryGroup + "]\n")
	for _, dev := range devices {
		if strings.TrimSpace(dev.Hostname) == "" {
			continue
		}
		b.WriteString(buildInventoryLine(dev, settings))
		b.WriteString("\n")
	}
	return b.String()
}

func ensureProjectAutoInventory(r *http.Request, projectID int, devices []db.Device, settings db.ProjectDeviceSettings) (db.Inventory, error) {
	inventories, err := helpers.Store(r).GetInventories(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return db.Inventory{}, err
	}
	content := renderWindowsInventory(devices, settings)
	for _, inv := range inventories {
		if !inv.IsDeviceDefaultAuto {
			continue
		}
		inv.Name = deviceAutoInventoryName
		inv.Type = db.InventoryStatic
		inv.Inventory = content
		if err = helpers.Store(r).UpdateInventory(inv); err != nil {
			return db.Inventory{}, err
		}
		return inv, nil
	}

	inv, err := helpers.Store(r).CreateInventory(db.Inventory{
		ProjectID:           projectID,
		Name:                deviceAutoInventoryName,
		Type:                db.InventoryStatic,
		Inventory:           content,
		IsDeviceDefaultAuto: true,
	})
	if err != nil {
		return db.Inventory{}, err
	}
	return inv, nil
}

func syncProjectAutoInventory(r *http.Request, projectID int) error {
	settings, err := helpers.Store(r).GetProjectDeviceSettings(projectID)
	if err != nil {
		return err
	}
	devices, err := helpers.Store(r).GetDevices(projectID, db.RetrieveQueryParams{})
	if err != nil {
		return err
	}
	inv, err := ensureProjectAutoInventory(r, projectID, devices, settings)
	if err != nil {
		return err
	}
	if settings.DefaultInventoryID == nil || *settings.DefaultInventoryID != inv.ID {
		settings.DefaultInventoryID = &inv.ID
		return helpers.Store(r).UpdateProjectDeviceSettings(settings)
	}
	return nil
}

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

func GetDeviceStatusReason(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	logs, err := helpers.Store(r).GetDeviceStatusCallbackLogs(device.ProjectID, device.ID, 20)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"hostname":        device.Hostname,
		"device_status":   device.DeviceStatus,
		"abnormal_reason": device.AbnormalReason,
		"last_updated":    device.LastUpdated,
		"logs":            logs,
	})
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
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	var device db.Device
	if !helpers.Bind(w, r, &device) {
		return
	}

	device.ProjectID = project.ID
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Hostname = strings.TrimSpace(device.Hostname)
	device.Name = device.Hostname
	if device.DeviceStatus == "" {
		device.DeviceStatus = db.DeviceStatusUnknown
	}
	if device.RDPStatus == "" {
		device.RDPStatus = db.DeviceStatusUnknown
	}
	if device.WinRMStatus == "" {
		device.WinRMStatus = db.DeviceStatusUnknown
	}
	normalizeDeviceConnection(&device, settings)

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
		Description: fmt.Sprintf("Device %s created", newDevice.Hostname),
	})
	if err = syncProjectAutoInventory(r, project.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, newDevice)
}

// UpdateDevice persists changes to an existing device.
func UpdateDevice(w http.ResponseWriter, r *http.Request) {
	old := helpers.GetFromContext(r, "device").(db.Device)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(old.ProjectID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	var device db.Device
	if !helpers.Bind(w, r, &device) {
		return
	}

	device.ID = old.ID
	device.ProjectID = old.ProjectID
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Hostname = strings.TrimSpace(device.Hostname)
	device.Name = device.Hostname
	device.RDPStatus = old.RDPStatus
	device.WinRMStatus = old.WinRMStatus
	if device.DeviceStatus == "" {
		device.DeviceStatus = old.DeviceStatus
	}
	device.AbnormalReason = old.AbnormalReason
	device.LastUpdated = old.LastUpdated
	normalizeDeviceConnection(&device, settings)

	if err := device.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}

	if err = helpers.Store(r).UpdateDevice(device); err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   old.ProjectID,
		ObjectType:  db.EventDevice,
		ObjectID:    old.ID,
		Description: fmt.Sprintf("Device %s updated", device.Hostname),
	})
	if err = syncProjectAutoInventory(r, old.ProjectID); err != nil {
		helpers.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveDevice deletes a device and (cascaded) its config items.
func RemoveDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	err := helpers.Store(r).DeleteDevice(device.ProjectID, device.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	helpers.EventLog(r, helpers.EventLogDelete, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   device.ProjectID,
		ObjectType:  db.EventDevice,
		ObjectID:    device.ID,
		Description: fmt.Sprintf("Device %s deleted", device.Hostname),
	})
	if err = syncProjectAutoInventory(r, device.ProjectID); err != nil {
		helpers.WriteError(w, err)
		return
	}

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
	if s.DefaultAnsibleConnection == "" {
		s.DefaultAnsibleConnection = "winrm"
	}
	if s.DefaultAnsibleWinRMTransport == "" {
		s.DefaultAnsibleWinRMTransport = "basic"
	}
	if s.DefaultAnsibleWinRMScheme == "" {
		s.DefaultAnsibleWinRMScheme = "http"
	}
	if s.DefaultAnsiblePort == 0 {
		s.DefaultAnsiblePort = 5985
	}
	if s.DefaultAnsibleWinRMServerCertValidation == "" {
		s.DefaultAnsibleWinRMServerCertValidation = "ignore"
	}

	if s.StatusRefreshIntervalMin < 0 {
		s.StatusRefreshIntervalMin = 0
	}

	if err := helpers.Store(r).UpdateProjectDeviceSettings(s); err != nil {
		helpers.WriteError(w, err)
		return
	}
	if err := syncProjectAutoInventory(r, project.ID); err != nil {
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
func runDeviceTemplate(r *http.Request, project db.Project, action db.DeviceAction, extraVars map[string]any, inventoryID *int) (db.Task, error) {
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
		InventoryID: inventoryID,
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
	var body struct {
		Subnet      string                 `json:"subnet"`
		NetworkCIDR string                 `json:"network_cidr"`
		ExtraVars   map[string]interface{} `json:"extra_vars"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil && err != io.EOF {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	extraVars := map[string]interface{}{}
	for k, v := range body.ExtraVars {
		extraVars[k] = v
	}

	subnet := strings.TrimSpace(body.Subnet)
	if subnet == "" {
		subnet = strings.TrimSpace(body.NetworkCIDR)
	}
	if subnet != "" {
		// Keep both keys for compatibility with existing templates.
		extraVars["subnet"] = subnet
		extraVars["network_cidr"] = subnet
	}

	task, err := runDeviceTemplate(r, project, db.DeviceActionDiscover, extraVars, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, task)
}

func ImportDiscoveredDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	var body struct {
		Devices           []db.Device `json:"devices"`
		SelectedHostnames []string    `json:"selected_hostnames"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	selected := map[string]bool{}
	for _, h := range body.SelectedHostnames {
		selected[strings.TrimSpace(h)] = true
	}
	var toUpsert []db.Device
	for _, dev := range body.Devices {
		dev.Hostname = strings.TrimSpace(dev.Hostname)
		dev.IPAddress = strings.TrimSpace(dev.IPAddress)
		dev.Name = dev.Hostname
		if dev.DeviceStatus == "" {
			if dev.RDPStatus == db.DeviceStatusOnline && dev.WinRMStatus == db.DeviceStatusOnline {
				dev.DeviceStatus = db.DeviceStatusHealthy
			} else if dev.RDPStatus == db.DeviceStatusOffline && dev.WinRMStatus == db.DeviceStatusOffline {
				dev.DeviceStatus = db.DeviceStatusUnhealthy
			} else {
				dev.DeviceStatus = db.DeviceStatusUnknown
			}
		}
		if dev.RDPStatus == "" {
			dev.RDPStatus = db.DeviceStatusUnknown
		}
		if dev.WinRMStatus == "" {
			dev.WinRMStatus = db.DeviceStatusUnknown
		}
		if dev.Hostname == "" {
			continue
		}
		normalizeDeviceConnection(&dev, settings)
		if len(selected) > 0 && !selected[dev.Hostname] {
			continue
		}
		toUpsert = append(toUpsert, dev)
	}
	saved, err := helpers.Store(r).UpsertDevicesByHostname(project.ID, toUpsert)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if err = syncProjectAutoInventory(r, project.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"saved_count": len(saved),
		"devices":     saved,
	})
}

func BulkUpdateDeviceStatus(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	updates, err := parseBulkDeviceStatusUpdates(r.Body)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if len(updates) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No status updates provided",
		})
		return
	}
	updated := 0
	devices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	byHostname := map[string]db.Device{}
	for _, dev := range devices {
		byHostname[dev.Hostname] = dev
	}
	for _, u := range updates {
		hostname := strings.TrimSpace(u.Hostname)
		if hostname == "" {
			continue
		}
		dev, ok := byHostname[hostname]
		if !ok {
			continue
		}
		switch u.Status {
		case db.DeviceStatusHealthy, db.DeviceStatusUnhealthy, db.DeviceStatusChecking, db.DeviceStatusUnknown:
		default:
			continue
		}
		refreshed := time.Now()
		if u.CheckedAt != nil {
			refreshed = *u.CheckedAt
		}
		dev.DeviceStatus = u.Status
		if u.RDPStatus != "" {
			dev.RDPStatus = u.RDPStatus
		}
		if u.WinRMStatus != "" {
			dev.WinRMStatus = u.WinRMStatus
		}
		dev.AbnormalReason = u.AbnormalReason
		dev.LastUpdated = &refreshed
		if err := helpers.Store(r).UpdateDevice(dev); err != nil {
			continue
		}
		payloadBytes, _ := json.Marshal(u)
		_, _ = helpers.Store(r).CreateDeviceStatusCallbackLog(db.DeviceStatusCallbackLog{
			ProjectID:      project.ID,
			DeviceID:       &dev.ID,
			Hostname:       hostname,
			Status:         u.Status,
			RDPStatus:      dev.RDPStatus,
			WinRMStatus:    dev.WinRMStatus,
			AbnormalReason: u.AbnormalReason,
			Payload:        string(payloadBytes),
			Created:        refreshed,
		})
		updated++
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"updated": updated})
}

func parseBulkDeviceStatusUpdates(body io.ReadCloser) ([]db.DeviceStatusUpdate, error) {
	defer body.Close()

	rawBody, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var updates []db.DeviceStatusUpdate
	if err := json.Unmarshal(rawBody, &updates); err == nil && len(updates) > 0 {
		normalizeBulkUpdates(updates)
		return updates, nil
	}

	type bulkPayload struct {
		Updates []bulkDeviceStatusUpdate `json:"updates"`
		Results []bulkDeviceStatusUpdate `json:"results"`
		Devices []bulkDeviceStatusUpdate `json:"devices"`
	}

	var payload bulkPayload
	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	var source []bulkDeviceStatusUpdate
	switch {
	case len(payload.Updates) > 0:
		source = payload.Updates
	case len(payload.Results) > 0:
		source = payload.Results
	case len(payload.Devices) > 0:
		source = payload.Devices
	default:
		return []db.DeviceStatusUpdate{}, nil
	}

	updates = make([]db.DeviceStatusUpdate, 0, len(source))
	for _, item := range source {
		u := item.toDeviceStatusUpdate()
		updates = append(updates, u)
	}
	normalizeBulkUpdates(updates)
	return updates, nil
}

func normalizeBulkUpdates(updates []db.DeviceStatusUpdate) {
	for i := range updates {
		updates[i].Hostname = strings.TrimSpace(updates[i].Hostname)
		updates[i].Status = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].Status))))
		if updates[i].RDPStatus != "" {
			updates[i].RDPStatus = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].RDPStatus))))
		}
		if updates[i].WinRMStatus != "" {
			updates[i].WinRMStatus = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].WinRMStatus))))
		}
	}
}

type bulkDeviceStatusUpdate struct {
	Hostname       string     `json:"hostname"`
	Status         string     `json:"status"`
	DeviceStatus   string     `json:"device_status"`
	RDPStatus      string     `json:"rdp_status"`
	RDP            string     `json:"rdp"`
	WinRMStatus    string     `json:"winrm_status"`
	WINRM          string     `json:"winrm"`
	AbnormalReason *string    `json:"abnormal_reason,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	CheckedAt      *time.Time `json:"checked_at,omitempty"`
	LastUpdated    *time.Time `json:"last_updated,omitempty"`
	Timestamp      *time.Time `json:"timestamp,omitempty"`
}

func (u bulkDeviceStatusUpdate) toDeviceStatusUpdate() db.DeviceStatusUpdate {
	status := u.Status
	if strings.TrimSpace(status) == "" {
		status = u.DeviceStatus
	}
	rdp := u.RDPStatus
	if strings.TrimSpace(rdp) == "" {
		rdp = u.RDP
	}
	winrm := u.WinRMStatus
	if strings.TrimSpace(winrm) == "" {
		winrm = u.WINRM
	}
	reason := u.AbnormalReason
	if reason == nil {
		reason = u.Reason
	}
	checkedAt := u.CheckedAt
	if checkedAt == nil {
		checkedAt = u.LastUpdated
	}
	if checkedAt == nil {
		checkedAt = u.Timestamp
	}

	return db.DeviceStatusUpdate{
		Hostname:       u.Hostname,
		Status:         db.DeviceStatus(status),
		RDPStatus:      db.DeviceStatus(rdp),
		WinRMStatus:    db.DeviceStatus(winrm),
		AbnormalReason: reason,
		CheckedAt:      checkedAt,
	}
}

func RunPatrolForAllDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	if err := syncProjectAutoInventory(r, project.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	devices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	payload := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		payload = append(payload, map[string]any{
			"id":       d.ID,
			"hostname": d.Hostname,
			"ip":       d.IPAddress,
		})
	}
	task, err := runDeviceTemplate(r, project, db.DeviceActionStatus, map[string]any{"devices": payload}, settings.DefaultInventoryID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	now := time.Now()
	for _, d := range devices {
		_ = helpers.Store(r).UpdateDeviceStatusByHostname(project.ID, d.Hostname, db.DeviceStatusChecking, now)
	}
	helpers.WriteJSON(w, http.StatusCreated, task)
}

func createTemporaryInventoryForDevices(r *http.Request, projectID int, devices []db.Device) (*int, error) {
	settings, err := helpers.Store(r).GetProjectDeviceSettings(projectID)
	if err != nil {
		return nil, err
	}
	content := renderWindowsInventory(devices, settings)
	inv, err := helpers.Store(r).CreateInventory(db.Inventory{
		ProjectID: projectID,
		Name:      fmt.Sprintf("windows_hosts batch %d", time.Now().Unix()),
		Type:      db.InventoryStatic,
		Inventory: content,
	})
	if err != nil {
		return nil, err
	}
	return &inv.ID, nil
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

	tmpInventoryID, err := createTemporaryInventoryForDevices(r, project.ID, []db.Device{device})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	task, err := runDeviceTemplate(r, project, body.Action, extraVars, tmpInventoryID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, task)
}

func RunBulkDeviceAction(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	var body struct {
		Action    db.DeviceAction `json:"action"`
		DeviceIDs []int           `json:"device_ids"`
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
	if len(body.DeviceIDs) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No devices selected",
		})
		return
	}

	idSet := map[int]bool{}
	for _, id := range body.DeviceIDs {
		idSet[id] = true
	}
	allDevices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	selected := make([]db.Device, 0, len(body.DeviceIDs))
	payload := make([]map[string]any, 0, len(body.DeviceIDs))
	for _, d := range allDevices {
		if !idSet[d.ID] {
			continue
		}
		selected = append(selected, d)
		payload = append(payload, map[string]any{
			"id":       d.ID,
			"hostname": d.Hostname,
			"ip":       d.IPAddress,
		})
	}
	if len(selected) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Selected devices not found",
		})
		return
	}

	tmpInventoryID, err := createTemporaryInventoryForDevices(r, project.ID, selected)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	extraVars := map[string]any{"devices": payload}
	task, err := runDeviceTemplate(r, project, body.Action, extraVars, tmpInventoryID)
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
	if rdp == db.DeviceStatusOnline && winrm == db.DeviceStatusOnline {
		device.DeviceStatus = db.DeviceStatusHealthy
	} else if rdp == db.DeviceStatusOffline && winrm == db.DeviceStatusOffline {
		device.DeviceStatus = db.DeviceStatusUnhealthy
	} else {
		device.DeviceStatus = db.DeviceStatusUnknown
	}
	device.LastUpdated = &refreshed
	helpers.WriteJSON(w, http.StatusOK, device)
}
