package bolt

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *BoltDb) GetDevice(projectID int, deviceID int) (device db.Device, err error) {
	err = d.getObject(projectID, db.DeviceProps, intObjectID(deviceID), &device)
	return
}

func (d *BoltDb) GetDevices(projectID int, params db.RetrieveQueryParams) (devices []db.Device, err error) {
	err = d.getObjects(projectID, db.DeviceProps, params, nil, &devices)
	return
}

func (d *BoltDb) DeleteDevice(projectID int, deviceID int) error {
	return d.deleteObject(projectID, db.DeviceProps, intObjectID(deviceID), nil)
}

func (d *BoltDb) UpdateDevice(device db.Device) error {
	return d.updateObject(device.ProjectID, db.DeviceProps, device)
}

func (d *BoltDb) CreateDevice(device db.Device) (db.Device, error) {
	if device.RDPStatus == "" {
		device.RDPStatus = db.DeviceStatusUnknown
	}
	if device.WinRMStatus == "" {
		device.WinRMStatus = db.DeviceStatusUnknown
	}
	device.Created = tz.Now()

	res, err := d.createObject(device.ProjectID, db.DeviceProps, device)
	if err != nil {
		return db.Device{}, err
	}
	return res.(db.Device), nil
}

func (d *BoltDb) UpdateDeviceStatus(projectID, deviceID int, rdp, winrm db.DeviceStatus, refreshed time.Time) error {
	device, err := d.GetDevice(projectID, deviceID)
	if err != nil {
		return err
	}
	device.RDPStatus = rdp
	device.WinRMStatus = winrm
	t := refreshed
	device.LastUpdated = &t
	return d.updateObject(projectID, db.DeviceProps, device)
}

func (d *BoltDb) GetDeviceStats(projectID int) (stats db.DeviceStats, err error) {
	devices, err := d.GetDevices(projectID, db.RetrieveQueryParams{})
	if err != nil {
		return
	}
	for _, dev := range devices {
		stats.Total++
		switch dev.RDPStatus {
		case db.DeviceStatusOnline:
			stats.RDPOnline++
		case db.DeviceStatusOffline:
			stats.RDPOffline++
		}
		switch dev.WinRMStatus {
		case db.DeviceStatusOnline:
			stats.WinRMOnline++
		case db.DeviceStatusOffline:
			stats.WinRMOffline++
		}
		if dev.RDPStatus == db.DeviceStatusUnknown && dev.WinRMStatus == db.DeviceStatusUnknown {
			stats.Unknown++
		}
	}
	return
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
