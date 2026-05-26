package projects

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func normalizeProtocolStatus(status db.DeviceStatus) db.DeviceStatus {
	if status == db.DeviceStatusOnline {
		return db.DeviceStatusOnline
	}
	return db.DeviceStatusOffline
}

func normalizeDeviceHealthStatus(status db.DeviceStatus) db.DeviceStatus {
	switch status {
	case db.DeviceStatusHealthy, db.DeviceStatusChecking:
		return status
	default:
		return db.DeviceStatusUnhealthy
	}
}

func normalizeDeviceStatuses(device *db.Device) {
	device.RDPStatus = normalizeProtocolStatus(device.RDPStatus)
	device.WinRMStatus = normalizeProtocolStatus(device.WinRMStatus)
	device.APIStatus = normalizeProtocolStatus(device.APIStatus)
	device.DeviceStatus = normalizeDeviceHealthStatus(device.DeviceStatus)
}

// normalizeDeviceConnection fills connection-related fields from project defaults when the
// device leaves them blank. ansible_user / ansible_password are intentionally omitted here so
// empty values stay empty in the database; buildInventoryLine still applies project defaults
// when generating Ansible inventory for tasks.
func normalizeDeviceConnection(device *db.Device, settings db.ProjectDeviceSettings) {
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
	if device.RDPPort <= 0 || device.RDPPort > 65535 {
		device.RDPPort = db.DefaultDeviceRDPPort
	}
	if device.APIPort <= 0 || device.APIPort > 65535 {
		device.APIPort = db.DefaultDeviceAPIPort
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
	port := db.EffectiveDeviceAnsiblePort(dev, settings)
	certValidation := dev.AnsibleWinRMServerCertValidation
	if certValidation == "" {
		certValidation = settings.DefaultAnsibleWinRMServerCertValidation
	}
	if certValidation == "" {
		certValidation = "ignore"
	}

	inventoryHost := strings.TrimSpace(dev.IPAddress)
	parts := []string{inventoryHost}
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
	parts = append(parts, "rdp_port="+strconv.Itoa(db.EffectiveDeviceRDPPort(dev)))
	if ap := db.EffectiveDeviceAPIPortForInventory(dev); ap > 0 {
		parts = append(parts, "api_port="+strconv.Itoa(ap))
	}
	return strings.Join(parts, " ")
}

func projectHasDeviceWithIP(r *http.Request, projectID int, ip string, exceptID int) (bool, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false, nil
	}

	devices, err := helpers.Store(r).GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return false, err
	}

	for _, d := range devices {
		if strings.TrimSpace(d.IPAddress) == ip && d.ID != exceptID {
			return true, nil
		}
	}

	return false, nil
}

func renderWindowsInventory(devices []db.Device, settings db.ProjectDeviceSettings) string {
	var b strings.Builder
	b.WriteString("[" + deviceAutoInventoryGroup + "]\n")
	for _, dev := range devices {
		if strings.TrimSpace(dev.IPAddress) == "" {
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
	devices, err := helpers.Store(r).GetDevices(projectID, db.RetrieveQueryParams{}, nil)
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

func parseDeviceListFilter(q url.Values) *db.DeviceListFilter {
	f := &db.DeviceListFilter{
		HostnameSubstring: strings.TrimSpace(q.Get("hostname")),
		IPSubstring:       strings.TrimSpace(q.Get("ip")),
		DeviceStatus:      strings.TrimSpace(q.Get("device_status")),
		RDPStatus:         strings.TrimSpace(q.Get("rdp_status")),
		WinRMStatus:       strings.TrimSpace(q.Get("winrm_status")),
		APIStatus:         strings.TrimSpace(q.Get("api_status")),
	}
	if f.DeviceStatus == string(db.DeviceStatusUnknown) {
		f.DeviceStatus = string(db.DeviceStatusUnhealthy)
	}
	if f.RDPStatus == string(db.DeviceStatusUnknown) {
		f.RDPStatus = string(db.DeviceStatusOffline)
	}
	if f.WinRMStatus == string(db.DeviceStatusUnknown) {
		f.WinRMStatus = string(db.DeviceStatusOffline)
	}
	if f.APIStatus == string(db.DeviceStatusUnknown) {
		f.APIStatus = string(db.DeviceStatusOffline)
	}
	if v := strings.TrimSpace(q.Get("device_profile_id")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.DeviceProfileID = n
		}
	}
	if f.HostnameSubstring == "" && f.IPSubstring == "" && f.DeviceStatus == "" &&
		f.RDPStatus == "" && f.WinRMStatus == "" && f.APIStatus == "" && f.DeviceProfileID <= 0 {
		return nil
	}
	return f
}

func deviceListRetrieveParams(r *http.Request) (db.RetrieveQueryParams, error) {
	params := helpers.QueryParamsForProps(r.URL, db.DeviceProps)
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return params, err
		}
		if n > 0 {
			params.Count = n
		}
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return params, err
		}
		if n >= 0 {
			params.Offset = n
		}
	}
	return params.Validate(db.DeviceProps)
}

// GetDevices returns the project's device list with optional pagination and filters.
// Response is always JSON object { "devices": [...], "total": N }.
func GetDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	_, _ = server.EnsureDefaultDeviceProfile(helpers.Store(r), project.ID)
	params, err := deviceListRetrieveParams(r)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	filter := parseDeviceListFilter(r.URL.Query())

	devices, err := helpers.Store(r).GetDevices(project.ID, params, filter)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if devices == nil {
		devices = []db.Device{}
	}
	for i := range devices {
		normalizeDeviceStatuses(&devices[i])
	}

	total := len(devices)
	if params.Count > 0 {
		total, err = helpers.Store(r).CountDevices(project.ID, filter)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"devices": devices,
		"total":   total,
	})
}

// GetDevice returns one device (loaded by DeviceMiddleware).
func GetDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	normalizeDeviceStatuses(&device)
	helpers.WriteJSON(w, http.StatusOK, device)
}

func GetDeviceStatusReason(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	normalizeDeviceStatuses(&device)
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
	device.AnsibleUser = strings.TrimSpace(device.AnsibleUser)
	device.AnsiblePassword = strings.TrimSpace(device.AnsiblePassword)
	device.RDPUser = strings.TrimSpace(device.RDPUser)
	device.RDPPassword = strings.TrimSpace(device.RDPPassword)
	normalizeDeviceStatuses(&device)
	normalizeDeviceConnection(&device, settings)

	defaultProf, err := server.EnsureDefaultDeviceProfile(helpers.Store(r), project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if device.DeviceProfileID <= 0 {
		device.DeviceProfileID = defaultProf.ID
	}

	if err := device.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}
	duplicateIP, err := projectHasDeviceWithIP(r, project.ID, device.IPAddress, 0)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if duplicateIP {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Device ip_address already exists in this project",
		})
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
	if device.DeviceProfileID <= 0 {
		device.DeviceProfileID = old.DeviceProfileID
	}
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Hostname = strings.TrimSpace(device.Hostname)
	device.Name = device.Hostname
	device.AnsibleUser = strings.TrimSpace(device.AnsibleUser)
	device.AnsiblePassword = strings.TrimSpace(device.AnsiblePassword)
	device.RDPUser = strings.TrimSpace(device.RDPUser)
	device.RDPPassword = strings.TrimSpace(device.RDPPassword)
	device.RDPStatus = old.RDPStatus
	device.WinRMStatus = old.WinRMStatus
	device.APIStatus = old.APIStatus
	if device.DeviceStatus == "" {
		device.DeviceStatus = old.DeviceStatus
	}
	normalizeDeviceStatuses(&device)
	device.AbnormalReason = old.AbnormalReason
	device.LastUpdated = old.LastUpdated
	normalizeDeviceConnection(&device, settings)

	if err := device.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}
	duplicateIP, err := projectHasDeviceWithIP(r, old.ProjectID, device.IPAddress, old.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if duplicateIP {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Device ip_address already exists in this project",
		})
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

// UpdateDeviceSettings is deprecated; use per-profile settings under /devices/profiles/{id}/settings.
func UpdateDeviceSettings(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusGone, map[string]string{
		"error": "Project-level device settings are removed. Configure Devices → Device types instead.",
	})
}

// deviceConnectionSettingsPayload is project-level WinRM defaults for windows_hosts inventory generation.
type deviceConnectionSettingsPayload struct {
	DefaultAnsibleUser                      string `json:"default_ansible_user"`
	DefaultAnsiblePassword                  string `json:"default_ansible_password"`
	DefaultAnsibleConnection                string `json:"default_ansible_connection"`
	DefaultAnsibleWinRMTransport            string `json:"default_ansible_winrm_transport"`
	DefaultAnsibleWinRMScheme               string `json:"default_ansible_winrm_scheme"`
	DefaultAnsiblePort                      int    `json:"default_ansible_port"`
	DefaultAnsibleWinRMServerCertValidation string `json:"default_ansible_winrm_server_cert_validation"`
}

func applyConnectionPayload(s *db.ProjectDeviceSettings, body deviceConnectionSettingsPayload) {
	s.DefaultAnsibleUser = strings.TrimSpace(body.DefaultAnsibleUser)
	s.DefaultAnsiblePassword = body.DefaultAnsiblePassword
	s.DefaultAnsibleConnection = strings.TrimSpace(body.DefaultAnsibleConnection)
	s.DefaultAnsibleWinRMTransport = strings.TrimSpace(body.DefaultAnsibleWinRMTransport)
	s.DefaultAnsibleWinRMScheme = strings.TrimSpace(body.DefaultAnsibleWinRMScheme)
	s.DefaultAnsiblePort = body.DefaultAnsiblePort
	if s.DefaultAnsiblePort <= 0 {
		s.DefaultAnsiblePort = db.DefaultDeviceAnsiblePort
	}
	s.DefaultAnsibleWinRMServerCertValidation = strings.TrimSpace(body.DefaultAnsibleWinRMServerCertValidation)
}

func connectionPayloadFromSettings(s db.ProjectDeviceSettings) deviceConnectionSettingsPayload {
	return deviceConnectionSettingsPayload{
		DefaultAnsibleUser:                      s.DefaultAnsibleUser,
		DefaultAnsiblePassword:                  s.DefaultAnsiblePassword,
		DefaultAnsibleConnection:                s.DefaultAnsibleConnection,
		DefaultAnsibleWinRMTransport:            s.DefaultAnsibleWinRMTransport,
		DefaultAnsibleWinRMScheme:               s.DefaultAnsibleWinRMScheme,
		DefaultAnsiblePort:                      s.DefaultAnsiblePort,
		DefaultAnsibleWinRMServerCertValidation: s.DefaultAnsibleWinRMServerCertValidation,
	}
}

// GetDeviceConnectionSettings returns project-level inventory connection defaults.
func GetDeviceConnectionSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	s, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, connectionPayloadFromSettings(s))
}

// UpdateDeviceConnectionSettings updates project-level inventory connection defaults only.
func UpdateDeviceConnectionSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	var body deviceConnectionSettingsPayload
	if !helpers.Bind(w, r, &body) {
		return
	}
	store := helpers.Store(r)
	s, err := store.GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	s.ProjectID = project.ID
	applyConnectionPayload(&s, body)
	if err := store.UpdateProjectDeviceSettings(s); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deviceDiscoverySettingsResponse is the project-level discovery template binding (no device type).
type deviceDiscoverySettingsResponse struct {
	DiscoverTemplateID *int `json:"discover_template_id,omitempty"`
	DefaultInventoryID *int `json:"default_inventory_id,omitempty"`
}

// GetDeviceDiscoverySettings returns the generic discovery template for the project.
func GetDeviceDiscoverySettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	s, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, deviceDiscoverySettingsResponse{
		DiscoverTemplateID: s.DiscoverTemplateID,
		DefaultInventoryID: s.DefaultInventoryID,
	})
}

// UpdateDeviceDiscoverySettings updates only discovery template bindings on project__device_settings.
func UpdateDeviceDiscoverySettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	var body deviceDiscoverySettingsResponse
	if !helpers.Bind(w, r, &body) {
		return
	}
	store := helpers.Store(r)
	s, err := store.GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	s.ProjectID = project.ID
	s.DiscoverTemplateID = body.DiscoverTemplateID
	s.DefaultInventoryID = body.DefaultInventoryID
	if err := store.UpdateProjectDeviceSettings(s); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func patchTaskEnvironmentJSON(store db.Store, task *db.Task, patch map[string]any) error {
	merged := map[string]any{}
	if strings.TrimSpace(task.Environment) != "" {
		if err := json.Unmarshal([]byte(task.Environment), &merged); err != nil {
			return err
		}
	}
	for k, v := range patch {
		merged[k] = v
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	task.Environment = string(b)
	return store.UpdateTask(*task)
}

func normalizeDiscoveredDeviceRows(rows []db.DiscoveredDeviceRow) []db.DiscoveredDeviceRow {
	out := make([]db.DiscoveredDeviceRow, 0, len(rows))
	for _, row := range rows {
		ip := strings.TrimSpace(row.IPAddress)
		if ip == "" {
			ip = strings.TrimSpace(row.IP)
		}
		if ip == "" {
			continue
		}
		host := strings.TrimSpace(row.Hostname)
		if host == "" {
			host = ip
		}
		rdp := strings.TrimSpace(row.RDPStatus)
		if rdp == "" {
			rdp = string(db.DeviceStatusOffline)
		}
		winrm := strings.TrimSpace(row.WinRMStatus)
		if winrm == "" {
			winrm = string(db.DeviceStatusOffline)
		}
		api := strings.TrimSpace(row.APIStatus)
		if api == "" {
			api = string(db.DeviceStatusOffline)
		}
		devStatus := strings.TrimSpace(row.DeviceStatus)
		if devStatus == "" {
			devStatus = strings.TrimSpace(row.Status)
		}
		if devStatus == "" {
			devStatus = string(db.DeviceStatusFromChannelProbes(
				db.DeviceStatus(rdp),
				db.DeviceStatus(winrm),
				db.DeviceStatus(api),
			))
		}
		out = append(out, db.DiscoveredDeviceRow{
			Hostname:       host,
			IPAddress:      ip,
			DeviceStatus:   devStatus,
			RDPStatus:      rdp,
			WinRMStatus:    winrm,
			APIStatus:      api,
			APIPort:        row.APIPort,
			AbnormalReason: strings.TrimSpace(row.AbnormalReason),
		})
	}
	return out
}

// GetDeviceDiscoveryResults returns persisted discovery hosts (by project; optional task_id filter).
func GetDeviceDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	taskID, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("task_id")))
	syncLog := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("sync")), "1") ||
		strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("sync")), "true")
	store := helpers.Store(r)
	hosts, err := store.ListDiscoveredHosts(project.ID, taskID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if taskID > 0 && (syncLog || len(hosts) == 0) {
		if _, syncErr := SyncDiscoveryResultsFromTaskOutput(store, project.ID, taskID); syncErr == nil {
			hosts, err = store.ListDiscoveredHosts(project.ID, taskID)
			if err != nil {
				helpers.WriteError(w, err)
				return
			}
		}
	}
	devices := make([]db.DiscoveredDeviceRow, 0, len(hosts))
	for _, h := range hosts {
		devices = append(devices, h.ToDiscoveredDeviceRow())
	}
	out := map[string]any{
		"devices": devices,
		"total":   len(devices),
	}
	if taskID > 0 {
		if run, runErr := store.GetDeviceDiscoveryRun(project.ID, taskID); runErr == nil {
			out["task_id"] = run.TaskID
			out["subnet"] = run.Subnet
			out["status"] = run.Status
		}
		if len(devices) == 0 && !server.HasPlaybookCallbackToken(nil) {
			out["callback_hint"] = "missing_semaphore_api_token"
		}
	}
	helpers.WriteJSON(w, http.StatusOK, out)
}

// PutDeviceDiscoveryResults is the playbook callback that stores discovery scan rows.
func PutDeviceDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	var body struct {
		TaskID  int                     `json:"task_id"`
		Devices []db.DiscoveredDeviceRow `json:"devices"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	if body.TaskID <= 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "task_id is required",
		})
		return
	}
	store := helpers.Store(r)
	if _, err := store.GetTask(project.ID, body.TaskID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	normalized := normalizeDiscoveredDeviceRows(body.Devices)
	upserted, err := store.UpsertDiscoveredHostsByIP(project.ID, body.TaskID, normalized)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	existing, err := store.GetDeviceDiscoveryRun(project.ID, body.TaskID)
	subnet := ""
	if err == nil {
		subnet = existing.Subnet
	} else if !errors.Is(err, db.ErrNotFound) {
		helpers.WriteError(w, err)
		return
	}
	run := db.DeviceDiscoveryRun{
		TaskID:      body.TaskID,
		ProjectID:   project.ID,
		Subnet:      subnet,
		Status:      db.DeviceDiscoveryRunReady,
		DevicesJSON: "[]",
		Updated:     time.Now(),
	}
	if err := store.UpsertDeviceDiscoveryRun(run); err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"task_id":  body.TaskID,
		"count":    upserted,
		"received": len(normalized),
	})
}

// runDeviceTemplate enqueues the template configured for the given action (project-level fallback).
func runDeviceTemplate(r *http.Request, project db.Project, action db.DeviceAction, extraVars map[string]any, inventoryID *int) (db.Task, error) {
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		return db.Task{}, err
	}
	ps := db.ProjectDeviceProfileSettings{
		ProjectID:            project.ID,
		DiscoverTemplateID:   settings.DiscoverTemplateID,
		StartTemplateID:      settings.StartTemplateID,
		StopTemplateID:       settings.StopTemplateID,
		RestartTemplateID:    settings.RestartTemplateID,
		StatusTemplateID:     settings.StatusTemplateID,
		ConfigTemplateID:     settings.ConfigTemplateID,
		DefaultInventoryID:   settings.DefaultInventoryID,
		DefaultConfigJSON:    settings.DefaultConfigJSON,
		StatusRefreshIntervalMin: settings.StatusRefreshIntervalMin,
	}
	return runDeviceTemplateWithProfileSettings(r, project, ps, action, extraVars, inventoryID)
}

// runDeviceTemplateWithProfileSettings uses per-profile template bindings.
func runDeviceTemplateWithProfileSettings(r *http.Request, project db.Project, ps db.ProjectDeviceProfileSettings, action db.DeviceAction, extraVars map[string]any, inventoryID *int) (db.Task, error) {
	tplID := ps.TemplateIDForAction(action)
	if tplID == nil || *tplID == 0 {
		return db.Task{}, &db.ValidationError{
			Message: fmt.Sprintf("No template configured for action %q on this device profile", action),
		}
	}
	if inventoryID == nil || *inventoryID == 0 {
		inventoryID = ps.DefaultInventoryID
	}

	tpl, err := helpers.Store(r).GetTemplate(project.ID, *tplID)
	if err != nil {
		return db.Task{}, err
	}

	mergedVars := map[string]any{}
	for k, v := range extraVars {
		mergedVars[k] = v
	}
	// Lets playbooks call PUT /devices/status/bulk without requiring SEMAPHORE_PROJECT_ID in template env.
	mergedVars["semaphore_project_id"] = project.ID
	server.InjectPlaybookCallbackVars(mergedVars)

	env := ""
	if len(mergedVars) > 0 {
		b, err := json.Marshal(mergedVars)
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
	if subnet == "" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "subnet is required (CIDR or single IP)",
		})
		return
	}
	// Keep both keys for compatibility with existing templates.
	extraVars["subnet"] = subnet
	extraVars["network_cidr"] = subnet

	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	invID := settings.DefaultInventoryID

	task, err := runDeviceTemplate(r, project, db.DeviceActionDiscover, extraVars, invID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	store := helpers.Store(r)
	if err := patchTaskEnvironmentJSON(store, &task, map[string]any{
		"semaphore_task_id": task.ID,
	}); err != nil {
		helpers.WriteError(w, err)
		return
	}
	if err := store.UpsertDeviceDiscoveryRun(db.DeviceDiscoveryRun{
		TaskID:      task.ID,
		ProjectID:   project.ID,
		Subnet:      subnet,
		Status:      db.DeviceDiscoveryRunPending,
		DevicesJSON: "[]",
		Updated:     time.Now(),
	}); err != nil {
		helpers.WriteError(w, err)
		return
	}
	resp := map[string]any{}
	taskBytes, _ := json.Marshal(task)
	_ = json.Unmarshal(taskBytes, &resp)
	warnCheck := map[string]any{}
	for k, v := range extraVars {
		warnCheck[k] = v
	}
	server.InjectPlaybookCallbackVars(warnCheck)
	if warn := discoveryCallbackWarning(warnCheck); warn != "" {
		resp["discovery_warning"] = warn
	}
	helpers.WriteJSON(w, http.StatusCreated, resp)
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
		SelectedIPs       []string    `json:"selected_ips"`
		SelectedHostnames []string    `json:"selected_hostnames"` // legacy: used only if selected_ips is empty
		DeviceProfileID   int         `json:"device_profile_id"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	if body.DeviceProfileID <= 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "device_profile_id is required to import discovered devices",
		})
		return
	}
	store := helpers.Store(r)
	if _, err := store.GetDeviceProfile(project.ID, body.DeviceProfileID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	selectedByIP := map[string]bool{}
	for _, ip := range body.SelectedIPs {
		if t := strings.TrimSpace(ip); t != "" {
			selectedByIP[t] = true
		}
	}
	selectedByHostname := map[string]bool{}
	for _, h := range body.SelectedHostnames {
		if t := strings.TrimSpace(h); t != "" {
			selectedByHostname[t] = true
		}
	}
	useIPFilter := len(selectedByIP) > 0
	useHostnameFilter := !useIPFilter && len(selectedByHostname) > 0

	byIP := map[string]db.Device{}
	for _, dev := range body.Devices {
		dev.Hostname = strings.TrimSpace(dev.Hostname)
		dev.IPAddress = strings.TrimSpace(dev.IPAddress)
		if dev.IPAddress == "" {
			continue
		}
		if dev.Hostname == "" {
			dev.Hostname = dev.IPAddress
		}
		dev.Name = dev.Hostname
		if dev.DeviceStatus == "" {
			dev.DeviceStatus = db.DeviceStatusFromChannelProbes(dev.RDPStatus, dev.WinRMStatus, dev.APIStatus)
		}
		normalizeDeviceStatuses(&dev)
		normalizeDeviceConnection(&dev, settings)
		// Upsert merges ports only when non-zero; avoid normalized defaults overwriting
		// manually configured ports when discovery JSON does not specify ports.
		dev.AnsiblePort = 0
		dev.RDPPort = 0
		dev.APIPort = 0
		if useIPFilter && !selectedByIP[dev.IPAddress] {
			continue
		}
		if useHostnameFilter && !selectedByHostname[dev.Hostname] {
			continue
		}
		byIP[dev.IPAddress] = dev
	}
	toUpsert := make([]db.Device, 0, len(byIP))
	for _, dev := range byIP {
		dev.DeviceProfileID = body.DeviceProfileID
		toUpsert = append(toUpsert, dev)
	}
	saved, err := helpers.Store(r).UpsertDevicesByIPAddress(project.ID, toUpsert)
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
	devices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	byHostname := map[string]db.Device{}
	byIP := map[string]db.Device{}
	for _, dev := range devices {
		byHostname[dev.Hostname] = dev
		if dev.IPAddress != "" {
			byIP[strings.TrimSpace(dev.IPAddress)] = dev
		}
	}
	for _, u := range updates {
		ipKey := strings.TrimSpace(u.IPAddress)
		hostKey := strings.TrimSpace(u.Hostname)
		if ipKey == "" && hostKey == "" {
			continue
		}
		var dev db.Device
		var ok bool
		if ipKey != "" {
			dev, ok = byIP[ipKey]
		}
		if !ok && hostKey != "" {
			if d, ok2 := byHostname[hostKey]; ok2 {
				dev, ok = d, true
			} else if d, ok2 := byIP[hostKey]; ok2 {
				dev, ok = d, true
			}
		}
		if !ok {
			continue
		}
		switch u.Status {
		case db.DeviceStatusHealthy, db.DeviceStatusUnhealthy, db.DeviceStatusChecking:
		default:
			continue
		}
		refreshed := time.Now()
		if u.CheckedAt != nil {
			refreshed = *u.CheckedAt
		}
		if u.RDPStatus != "" {
			dev.RDPStatus = u.RDPStatus
		}
		if u.WinRMStatus != "" {
			dev.WinRMStatus = u.WinRMStatus
		}
		if u.APIStatus != "" {
			dev.APIStatus = u.APIStatus
		}
		// Trust playbook-supplied aggregate device_status. Templates may report healthy when the
		// log/upload path proves OK while api_status stays offline (HTTP != 200 per column rules).
		// Do not downgrade healthy→unhealthy here — that caused Patrol "NORMAL" vs UI mismatch.
		dev.DeviceStatus = u.Status
		normalizeDeviceStatuses(&dev)
		dev.AbnormalReason = u.AbnormalReason
		dev.LastUpdated = &refreshed
		if err := helpers.Store(r).UpdateDevice(dev); err != nil {
			continue
		}
		if shouldRecordAbnormalLog(dev.DeviceStatus, u) {
			payloadBytes, _ := json.Marshal(u)
			_, _ = helpers.Store(r).CreateDeviceStatusCallbackLog(db.DeviceStatusCallbackLog{
				ProjectID:      project.ID,
				DeviceID:       &dev.ID,
				Hostname:       dev.Hostname,
				Status:         dev.DeviceStatus,
				RDPStatus:      dev.RDPStatus,
				WinRMStatus:    dev.WinRMStatus,
				APIStatus:      dev.APIStatus,
				AbnormalReason: u.AbnormalReason,
				Payload:        string(payloadBytes),
				Created:        refreshed,
			})
		}
		updated++
	}
	store := helpers.Store(r)
	go server.PublishProjectStatusSnapshots(store, project.ID)
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"updated": updated})
}

func shouldRecordAbnormalLog(finalStatus db.DeviceStatus, u db.DeviceStatusUpdate) bool {
	if finalStatus == db.DeviceStatusUnhealthy {
		return true
	}
	return u.AbnormalReason != nil && strings.TrimSpace(*u.AbnormalReason) != ""
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
		updates[i].IPAddress = strings.TrimSpace(updates[i].IPAddress)
		updates[i].Status = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].Status))))
		if updates[i].RDPStatus != "" {
			updates[i].RDPStatus = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].RDPStatus))))
		}
		if updates[i].WinRMStatus != "" {
			updates[i].WinRMStatus = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].WinRMStatus))))
		}
		if updates[i].APIStatus != "" {
			updates[i].APIStatus = db.DeviceStatus(strings.ToLower(strings.TrimSpace(string(updates[i].APIStatus))))
		}
		updates[i].Status = normalizeDeviceHealthStatus(updates[i].Status)
		if updates[i].RDPStatus != "" {
			updates[i].RDPStatus = normalizeProtocolStatus(updates[i].RDPStatus)
		}
		if updates[i].WinRMStatus != "" {
			updates[i].WinRMStatus = normalizeProtocolStatus(updates[i].WinRMStatus)
		}
		if updates[i].APIStatus != "" {
			updates[i].APIStatus = normalizeProtocolStatus(updates[i].APIStatus)
		}
	}
}

type bulkDeviceStatusUpdate struct {
	Hostname       string     `json:"hostname"`
	IP             string     `json:"ip"`
	IPAddress      string     `json:"ip_address"`
	Status         string     `json:"status"`
	DeviceStatus   string     `json:"device_status"`
	RDPStatus      string     `json:"rdp_status"`
	RDP            string     `json:"rdp"`
	WinRMStatus    string     `json:"winrm_status"`
	WINRM          string     `json:"winrm"`
	APIStatus      string     `json:"api_status"`
	API            string     `json:"api"`
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
	ipStr := strings.TrimSpace(u.IP)
	if ipStr == "" {
		ipStr = strings.TrimSpace(u.IPAddress)
	}
	rdp := u.RDPStatus
	if strings.TrimSpace(rdp) == "" {
		rdp = u.RDP
	}
	winrm := u.WinRMStatus
	if strings.TrimSpace(winrm) == "" {
		winrm = u.WINRM
	}
	api := u.APIStatus
	if strings.TrimSpace(api) == "" {
		api = u.API
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
		Hostname:       strings.TrimSpace(u.Hostname),
		IPAddress:      ipStr,
		Status:         db.DeviceStatus(status),
		RDPStatus:      db.DeviceStatus(rdp),
		WinRMStatus:    db.DeviceStatus(winrm),
		APIStatus:      db.DeviceStatus(api),
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
	_, _ = server.EnsureDefaultDeviceProfile(helpers.Store(r), project.ID)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	devices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	store := helpers.Store(r)
	now := time.Now()
	tasks := make([]db.Task, 0)
	for profileID, devs := range server.GroupDevicesByProfile(devices) {
		if profileID <= 0 {
			prof, err := server.EnsureDefaultDeviceProfile(store, project.ID)
			if err != nil {
				helpers.WriteError(w, err)
				return
			}
			for i := range devs {
				devs[i].DeviceProfileID = prof.ID
			}
			profileID = prof.ID
		}
		_, ps, err := server.ResolveDeviceProfileSettings(store, project.ID, devs[0])
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		payload := make([]map[string]any, 0, len(devs))
		for _, d := range devs {
			payload = append(payload, map[string]any{
				"id":           d.ID,
				"hostname":     d.Hostname,
				"ip":           d.IPAddress,
				"rdp_user":     d.RDPUser,
				"rdp_password": d.RDPPassword,
				"rdp_port":     db.EffectiveDeviceRDPPort(d),
				"ansible_port": db.EffectiveDeviceAnsiblePort(d, settings),
				"api_port":     db.EffectiveDeviceAPIPortForExtraVars(d),
			})
			_ = store.UpdateDeviceStatusByHostname(project.ID, d.Hostname, db.DeviceStatusChecking, now)
		}
		invID := ps.DefaultInventoryID
		if invID == nil {
			invID = settings.DefaultInventoryID
		}
		task, err := runDeviceTemplateWithProfileSettings(r, project, ps, db.DeviceActionStatus, map[string]any{"devices": payload}, invID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		tasks = append(tasks, task)
	}
	if len(tasks) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "No status template configured for any device profile",
		})
		return
	}
	if len(tasks) == 1 {
		helpers.WriteJSON(w, http.StatusCreated, tasks[0])
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, map[string]any{"tasks": tasks})
}

func createTemporaryInventoryForDevices(r *http.Request, projectID int, devices []db.Device) (*int, error) {
	settings, err := helpers.Store(r).GetProjectDeviceSettings(projectID)
	if err != nil {
		return nil, err
	}
	content := renderWindowsInventory(devices, settings)
	inv, err := helpers.Store(r).CreateInventory(db.Inventory{
		ProjectID: projectID,
		Name:      fmt.Sprintf("%s%d", db.DeviceEphemeralBatchInventoryPrefix, time.Now().Unix()),
		Type:      db.InventoryStatic,
		Inventory: content,
	})
	if err != nil {
		return nil, err
	}
	return &inv.ID, nil
}

func enqueueDeviceActionTask(
	r *http.Request,
	project db.Project,
	action db.DeviceAction,
	extraVars map[string]any,
	inventoryID *int,
) (db.Task, error) {
	return runDeviceTemplate(r, project, action, extraVars, inventoryID)
}

func parseDefaultDeviceConfigJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	return parsed
}

func buildCategorizedDeviceConfig(items []db.DeviceConfigItem) map[string]map[string]string {
	categorized := map[string]map[string]string{}
	for _, it := range items {
		cat := strings.TrimSpace(it.Category)
		if cat == "" {
			cat = "default"
		}
		if categorized[cat] == nil {
			categorized[cat] = map[string]string{}
		}
		categorized[cat][it.Key] = it.Value
	}
	return categorized
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

	if err := server.ValidateDeviceHasProfile(device); err != nil {
		helpers.WriteError(w, err)
		return
	}
	_, ps, err := server.ResolveDeviceProfileSettings(helpers.Store(r), project.ID, device)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	extraVars := map[string]any{
		"device": map[string]any{
			"id":           device.ID,
			"ip":           device.IPAddress,
			"hostname":     device.Hostname,
			"rdp_user":     device.RDPUser,
			"rdp_password": device.RDPPassword,
			"rdp_port":     db.EffectiveDeviceRDPPort(device),
			"ansible_port": db.EffectiveDeviceAnsiblePort(device, settings),
			"api_port":     db.EffectiveDeviceAPIPortForExtraVars(device),
		},
	}

	if defaultConfig := parseDefaultDeviceConfigJSON(settings.DefaultConfigJSON); defaultConfig != nil {
		extraVars["default_config"] = defaultConfig
	}

	if body.Action == db.DeviceActionConfig || body.Action == db.DeviceActionStart || body.Action == db.DeviceActionRestart {
		items, err := helpers.Store(r).GetDeviceConfigItems(project.ID, device.ID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		extraVars["config"] = buildCategorizedDeviceConfig(items)
	}

	tmpInventoryID, err := createTemporaryInventoryForDevices(r, project.ID, []db.Device{device})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	effectiveAction := body.Action
	if body.Action == db.DeviceActionConfig {
		effectiveAction = db.DeviceActionRestart
		extraVars["triggered_by"] = "config"
	}
	task, err := runDeviceTemplateWithProfileSettings(r, project, ps, effectiveAction, extraVars, tmpInventoryID)
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
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	allDevices, err := helpers.Store(r).GetDevices(project.ID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	selected := make([]db.Device, 0, len(body.DeviceIDs))
	for _, d := range allDevices {
		if !idSet[d.ID] {
			continue
		}
		selected = append(selected, d)
	}
	if len(selected) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Selected devices not found",
		})
		return
	}
	for _, d := range selected {
		if err := server.ValidateDeviceHasProfile(d); err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}
	}

	effectiveAction := body.Action
	if body.Action == db.DeviceActionConfig {
		effectiveAction = db.DeviceActionRestart
	}

	store := helpers.Store(r)
	tasks := make([]db.Task, 0)
	byProfile := server.GroupDevicesByProfile(selected)
	for profileID, devs := range byProfile {
		if profileID <= 0 {
			prof, err := server.EnsureDefaultDeviceProfile(store, project.ID)
			if err != nil {
				helpers.WriteError(w, err)
				return
			}
			for i := range devs {
				devs[i].DeviceProfileID = prof.ID
			}
			profileID = prof.ID
		}
		_, ps, err := server.ResolveDeviceProfileSettings(store, project.ID, devs[0])
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		profilePayload := make([]map[string]any, 0, len(devs))
		for _, d := range devs {
			profilePayload = append(profilePayload, map[string]any{
				"id":           d.ID,
				"hostname":     d.Hostname,
				"ip":           d.IPAddress,
				"rdp_user":     d.RDPUser,
				"rdp_password": d.RDPPassword,
				"rdp_port":     db.EffectiveDeviceRDPPort(d),
				"ansible_port": db.EffectiveDeviceAnsiblePort(d, settings),
				"api_port":     db.EffectiveDeviceAPIPortForExtraVars(d),
			})
		}
		tmpInventoryID, err := createTemporaryInventoryForDevices(r, project.ID, devs)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		extraVars := map[string]any{"devices": profilePayload}
		if defaultConfig := parseDefaultDeviceConfigJSON(ps.DefaultConfigJSON); defaultConfig != nil {
			extraVars["default_config"] = defaultConfig
		} else if defaultConfig := parseDefaultDeviceConfigJSON(settings.DefaultConfigJSON); defaultConfig != nil {
			extraVars["default_config"] = defaultConfig
		}
		if body.Action == db.DeviceActionConfig || body.Action == db.DeviceActionStart || body.Action == db.DeviceActionRestart {
			configByHostname := map[string]map[string]map[string]string{}
			configsByHost := map[string]map[string]map[string]string{}
			for _, d := range devs {
				items, itemErr := store.GetDeviceConfigItems(project.ID, d.ID)
				if itemErr != nil {
					helpers.WriteError(w, itemErr)
					return
				}
				cfg := buildCategorizedDeviceConfig(items)
				configByHostname[d.Hostname] = cfg
				if ip := strings.TrimSpace(d.IPAddress); ip != "" {
					configsByHost[ip] = cfg
				}
				if h := strings.TrimSpace(d.Hostname); h != "" {
					configsByHost[h] = cfg
				}
			}
			extraVars["configs_by_hostname"] = configByHostname
			extraVars["configs_by_host"] = configsByHost
		}
		if body.Action == db.DeviceActionConfig {
			extraVars["triggered_by"] = "config"
		}
		task, err := runDeviceTemplateWithProfileSettings(r, project, ps, effectiveAction, extraVars, tmpInventoryID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		tasks = append(tasks, task)
	}
	if len(tasks) == 1 {
		helpers.WriteJSON(w, http.StatusCreated, tasks[0])
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, map[string]any{"tasks": tasks})
}

// ProbeDevice runs an immediate server-side TCP port probe of RDP, WinRM,
// and the configured application API port for one device and persists
// rdp_status, winrm_status, api_status, and last_updated only
// (device_status is unchanged). Useful for instant feedback when no status
// template is configured.
func ProbeDevice(w http.ResponseWriter, r *http.Request) {
	device := helpers.GetFromContext(r, "device").(db.Device)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(device.ProjectID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	rdp, winrm, api, refreshed := server.ProbeDevice(device, settings)
	if err := helpers.Store(r).UpdateDevicePortProbeStatuses(
		device.ProjectID, device.ID, rdp, winrm, api, refreshed,
	); err != nil {
		helpers.WriteError(w, err)
		return
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	device.APIStatus = api
	normalizeDeviceStatuses(&device)
	device.LastUpdated = &refreshed
	helpers.WriteJSON(w, http.StatusOK, device)
}
