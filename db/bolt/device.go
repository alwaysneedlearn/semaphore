package bolt

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *BoltDb) GetDevice(projectID int, deviceID int) (device db.Device, err error) {
	err = d.getObject(projectID, db.DeviceProps, intObjectID(deviceID), &device)
	return
}

func (d *BoltDb) GetDeviceStatusCallbackLogs(projectID int, deviceID int, limit int) ([]db.DeviceStatusCallbackLog, error) {
	if limit <= 0 {
		limit = 20
	}
	// Deprecated backend: no dedicated callback log table. Return empty list.
	return []db.DeviceStatusCallbackLog{}, nil
}

func (d *BoltDb) CreateDeviceStatusCallbackLog(l db.DeviceStatusCallbackLog) (db.DeviceStatusCallbackLog, error) {
	// Deprecated backend: persist callback payload into abnormal_reason fallback.
	if l.DeviceID != nil {
		dev, err := d.GetDevice(l.ProjectID, *l.DeviceID)
		if err == nil {
			msg := l.Payload
			if msg == "" {
				b, _ := json.Marshal(l)
				msg = string(b)
			}
			dev.AbnormalReason = &msg
			_ = d.UpdateDevice(dev)
		}
	}
	return l, nil
}

func filterDevicesInMemory(devices []db.Device, filter *db.DeviceListFilter) []db.Device {
	if filter == nil {
		return devices
	}
	hostSub := strings.TrimSpace(filter.HostnameSubstring)
	ipSub := strings.TrimSpace(filter.IPSubstring)
	ds := strings.TrimSpace(filter.DeviceStatus)
	rs := strings.TrimSpace(filter.RDPStatus)
	ws := strings.TrimSpace(filter.WinRMStatus)
	as := strings.TrimSpace(filter.APIStatus)

	out := make([]db.Device, 0, len(devices))
	for _, dev := range devices {
		if hostSub != "" && !strings.Contains(strings.ToLower(dev.Hostname), strings.ToLower(hostSub)) {
			continue
		}
		if ipSub != "" && !strings.Contains(strings.ToLower(dev.IPAddress), strings.ToLower(ipSub)) {
			continue
		}
		if ds != "" && string(dev.DeviceStatus) != ds {
			continue
		}
		if rs != "" && string(dev.RDPStatus) != rs {
			continue
		}
		if ws != "" && string(dev.WinRMStatus) != ws {
			continue
		}
		if as != "" && string(dev.APIStatus) != as {
			continue
		}
		if filter.DeviceProfileID > 0 && dev.DeviceProfileID != filter.DeviceProfileID {
			continue
		}
		out = append(out, dev)
	}
	return out
}

func sliceDevicesPaged(all []db.Device, params db.RetrieveQueryParams) []db.Device {
	if params.Count <= 0 && params.Offset <= 0 {
		return all
	}
	start := params.Offset
	if start > len(all) {
		return []db.Device{}
	}
	end := len(all)
	if params.Count > 0 {
		end = start + params.Count
		if end > len(all) {
			end = len(all)
		}
	}
	return all[start:end]
}

func (d *BoltDb) GetDevices(projectID int, params db.RetrieveQueryParams, filter *db.DeviceListFilter) (devices []db.Device, err error) {
	var all []db.Device
	err = d.getObjects(projectID, db.DeviceProps, db.RetrieveQueryParams{}, nil, &all)
	if err != nil {
		return
	}
	all = filterDevicesInMemory(all, filter)

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = db.DeviceProps.DefaultSortingColumn
	}
	validSort := false
	for _, col := range db.DeviceProps.SortableColumns {
		if col == sortBy {
			validSort = true
			break
		}
	}
	if !validSort {
		sortBy = db.DeviceProps.DefaultSortingColumn
	}
	if err = sortObjects(&all, sortBy, params.SortInverted); err != nil {
		return
	}
	devices = sliceDevicesPaged(all, params)
	return
}

func (d *BoltDb) CountDevices(projectID int, filter *db.DeviceListFilter) (int, error) {
	var all []db.Device
	err := d.getObjects(projectID, db.DeviceProps, db.RetrieveQueryParams{}, nil, &all)
	if err != nil {
		return 0, err
	}
	all = filterDevicesInMemory(all, filter)
	return len(all), nil
}

func (d *BoltDb) DeleteDevice(projectID int, deviceID int) error {
	return d.deleteObject(projectID, db.DeviceProps, intObjectID(deviceID), nil)
}

func (d *BoltDb) UpdateDevice(device db.Device) error {
	return d.updateObject(device.ProjectID, db.DeviceProps, device)
}

func (d *BoltDb) CreateDevice(device db.Device) (db.Device, error) {
	if device.Name == "" {
		device.Name = device.Hostname
	}
	if device.DeviceStatus == "" {
		device.DeviceStatus = db.DeviceStatusUnhealthy
	}
	if device.RDPStatus == "" {
		device.RDPStatus = db.DeviceStatusOffline
	}
	if device.WinRMStatus == "" {
		device.WinRMStatus = db.DeviceStatusOffline
	}
	if device.AnsiblePort <= 0 || device.AnsiblePort > 65535 {
		device.AnsiblePort = db.DefaultDeviceAnsiblePort
	}
	if device.RDPPort <= 0 || device.RDPPort > 65535 {
		device.RDPPort = db.DefaultDeviceRDPPort
	}
	if device.APIPort <= 0 || device.APIPort > 65535 {
		device.APIPort = db.DefaultDeviceAPIPort
	}
	if device.APIStatus == "" {
		device.APIStatus = db.DeviceStatusOffline
	}
	device.Created = tz.Now()

	res, err := d.createObject(device.ProjectID, db.DeviceProps, device)
	if err != nil {
		return db.Device{}, err
	}
	return res.(db.Device), nil
}

func (d *BoltDb) UpdateDeviceStatus(projectID, deviceID int, rdp, winrm, api db.DeviceStatus, refreshed time.Time) error {
	device, err := d.GetDevice(projectID, deviceID)
	if err != nil {
		return err
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	device.APIStatus = api
	device.DeviceStatus = db.DeviceStatusFromChannelProbes(rdp, winrm, api)
	t := refreshed
	device.LastUpdated = &t
	return d.updateObject(projectID, db.DeviceProps, device)
}

func (d *BoltDb) UpdateDevicePortProbeStatuses(projectID, deviceID int, rdp, winrm, api db.DeviceStatus, refreshed time.Time) error {
	device, err := d.GetDevice(projectID, deviceID)
	if err != nil {
		return err
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	device.APIStatus = api
	t := refreshed
	device.LastUpdated = &t
	return d.updateObject(projectID, db.DeviceProps, device)
}

func (d *BoltDb) GetDeviceStats(projectID int) (stats db.DeviceStats, err error) {
	devices, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return
	}
	for _, dev := range devices {
		stats.Total++
		switch dev.DeviceStatus {
		case db.DeviceStatusHealthy:
			stats.Healthy++
		case db.DeviceStatusUnhealthy:
			stats.Unhealthy++
		case db.DeviceStatusChecking:
			stats.Checking++
		default:
			stats.Unhealthy++
		}
	}
	return
}

func (d *BoltDb) UpdateDeviceStatusByHostname(projectID int, hostname string, status db.DeviceStatus, refreshed time.Time) error {
	devices, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return err
	}
	for _, dev := range devices {
		if dev.Hostname == hostname {
			dev.DeviceStatus = status
			t := refreshed
			dev.LastUpdated = &t
			return d.UpdateDevice(dev)
		}
	}
	return db.ErrNotFound
}

func (d *BoltDb) UpsertDevicesFromDiscoveryImport(projectID int, devices []db.Device) ([]db.Device, error) {
	existing, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return nil, err
	}
	byIP := map[string]db.Device{}
	for _, dev := range existing {
		if ip := strings.TrimSpace(dev.IPAddress); ip != "" {
			byIP[ip] = dev
		}
	}
	var saved []db.Device
	for _, dev := range devices {
		ip := strings.TrimSpace(dev.IPAddress)
		if ip == "" {
			continue
		}
		dev.IPAddress = ip
		if old, ok := byIP[ip]; ok {
			old.IPAddress = ip
			if strings.TrimSpace(dev.Hostname) != "" {
				old.Hostname = strings.TrimSpace(dev.Hostname)
				old.Name = old.Hostname
			}
			db.MergeDeviceCredentialsOnUpsert(&old, dev)
			db.MergeDevicePortsOnUpsert(&old, dev)
			old.AnsibleConnection = dev.AnsibleConnection
			old.AnsibleWinRMTransport = dev.AnsibleWinRMTransport
			old.AnsibleWinRMScheme = dev.AnsibleWinRMScheme
			old.AnsibleWinRMServerCertValidation = dev.AnsibleWinRMServerCertValidation
			if dev.RDPStatus != "" {
				old.RDPStatus = dev.RDPStatus
			}
			if dev.WinRMStatus != "" {
				old.WinRMStatus = dev.WinRMStatus
			}
			if dev.APIStatus != "" {
				old.APIStatus = dev.APIStatus
			}
			now := tz.Now()
			old.LastUpdated = &now
			if err = d.UpdateDevice(old); err != nil {
				return nil, err
			}
			byIP[ip] = old
			saved = append(saved, old)
		} else {
			dev.ProjectID = projectID
			if strings.TrimSpace(dev.Hostname) == "" {
				dev.Hostname = ip
			}
			dev.Name = dev.Hostname
			if dev.DeviceStatus == "" {
				dev.DeviceStatus = db.DeviceStatusUnknown
			}
			created, cErr := d.CreateDevice(dev)
			if cErr != nil {
				return nil, cErr
			}
			byIP[ip] = created
			saved = append(saved, created)
		}
	}
	return saved, nil
}

func (d *BoltDb) UpsertDevicesByIPAddress(projectID int, devices []db.Device) ([]db.Device, error) {
	existing, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return nil, err
	}
	byIP := map[string]db.Device{}
	for _, dev := range existing {
		if ip := strings.TrimSpace(dev.IPAddress); ip != "" {
			byIP[ip] = dev
		}
	}
	var saved []db.Device
	for _, dev := range devices {
		ip := strings.TrimSpace(dev.IPAddress)
		if ip == "" {
			continue
		}
		dev.IPAddress = ip
		if old, ok := byIP[ip]; ok {
			old.IPAddress = ip
			if strings.TrimSpace(dev.Hostname) != "" {
				old.Hostname = strings.TrimSpace(dev.Hostname)
				old.Name = old.Hostname
			}
			db.MergeDeviceCredentialsOnUpsert(&old, dev)
			db.MergeDevicePortsOnUpsert(&old, dev)
			old.AnsibleConnection = dev.AnsibleConnection
			old.AnsibleWinRMTransport = dev.AnsibleWinRMTransport
			old.AnsibleWinRMScheme = dev.AnsibleWinRMScheme
			old.AnsibleWinRMServerCertValidation = dev.AnsibleWinRMServerCertValidation
			if dev.DeviceStatus != "" {
				old.DeviceStatus = dev.DeviceStatus
			}
			if dev.RDPStatus != "" {
				old.RDPStatus = dev.RDPStatus
			}
			if dev.WinRMStatus != "" {
				old.WinRMStatus = dev.WinRMStatus
			}
			if dev.APIStatus != "" {
				old.APIStatus = dev.APIStatus
			}
			old.AbnormalReason = dev.AbnormalReason
			now := tz.Now()
			old.LastUpdated = &now
			if err = d.UpdateDevice(old); err != nil {
				return nil, err
			}
			byIP[ip] = old
			saved = append(saved, old)
		} else {
			dev.ProjectID = projectID
			if strings.TrimSpace(dev.Hostname) == "" {
				dev.Hostname = ip
			}
			dev.Name = dev.Hostname
			created, cErr := d.CreateDevice(dev)
			if cErr != nil {
				return nil, cErr
			}
			byIP[ip] = created
			saved = append(saved, created)
		}
	}
	return saved, nil
}

// Device config items in Bolt are stored as a side-list keyed by device id.
// We piggyback on the device record by serializing items into a synthetic
// bucket scoped by the device id. To keep things simple for the deprecated
// Bolt backend, we use createObject/getObjects with DeviceConfigItemProps and
// the device id as the parent bucket id.

func (d *BoltDb) GetDeviceConfigItems(projectID, deviceID int) ([]db.DeviceConfigItem, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return nil, err
	}
	var items []db.DeviceConfigItem
	err := d.getObjects(deviceID, db.DeviceConfigItemProps, db.RetrieveQueryParams{}, nil, &items)
	return items, err
}

func (d *BoltDb) SetDeviceConfigItems(projectID, deviceID int, items []db.DeviceConfigItem) error {
	existing, err := d.GetDeviceConfigItems(projectID, deviceID)
	if err != nil {
		return err
	}
	for _, it := range existing {
		if err = d.deleteObject(deviceID, db.DeviceConfigItemProps, intObjectID(it.ID), nil); err != nil {
			return err
		}
	}
	for _, it := range items {
		if it.Key == "" {
			continue
		}
		it.DeviceID = deviceID
		if _, err = d.createObject(deviceID, db.DeviceConfigItemProps, it); err != nil {
			return err
		}
	}
	return nil
}

// ProjectDeviceSettings: stored as a single-row "object" in a bucket per project.
// We use intObjectID(0) as the well-known id for the singleton settings record.
const projectDeviceSettingsObjectID = intObjectID(1)

var projectDeviceSettingsProps = db.ObjectProps{
	TableName:         "project__device_settings",
	PrimaryColumnName: "project_id",
}

func (d *BoltDb) GetProjectDeviceSettings(projectID int) (settings db.ProjectDeviceSettings, err error) {
	err = d.getObject(projectID, projectDeviceSettingsProps, projectDeviceSettingsObjectID, &settings)
	if err == db.ErrNotFound {
		settings = db.ProjectDeviceSettings{ProjectID: projectID}
		err = nil
	}
	return
}

func (d *BoltDb) UpdateProjectDeviceSettings(s db.ProjectDeviceSettings) error {
	// updateObject requires the object to exist; createObject otherwise.
	existing, err := d.GetProjectDeviceSettings(s.ProjectID)
	if err != nil && err != db.ErrNotFound {
		return err
	}
	if existing.ProjectID == 0 || existing == (db.ProjectDeviceSettings{}) {
		_, err = d.createObject(s.ProjectID, projectDeviceSettingsProps, s)
		return err
	}
	return d.updateObject(s.ProjectID, projectDeviceSettingsProps, s)
}

func (d *BoltDb) MarkProjectStatusRefreshed(projectID int, refreshed time.Time) error {
	s, err := d.GetProjectDeviceSettings(projectID)
	if err != nil {
		return err
	}
	t := refreshed
	s.LastStatusRefreshAt = &t
	return d.UpdateProjectDeviceSettings(s)
}

// GetProjectsDueForStatusRefresh is an unsupported op for the deprecated Bolt
// backend; periodic device refresh is only available with SQL stores.
func (d *BoltDb) GetProjectsDueForStatusRefresh(_ time.Time) ([]db.ProjectDeviceSettings, error) {
	return nil, nil
}
