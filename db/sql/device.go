package sql

import (
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *SqlDb) GetDevice(projectID int, deviceID int) (device db.Device, err error) {
	err = d.getObject(projectID, db.DeviceProps, deviceID, &device)
	return
}

func (d *SqlDb) GetDevices(projectID int, params db.RetrieveQueryParams) ([]db.Device, error) {
	var devices []db.Device
	err := d.getObjects(projectID, db.DeviceProps, params, nil, &devices)
	return devices, err
}

func (d *SqlDb) DeleteDevice(projectID int, deviceID int) error {
	return d.deleteObject(projectID, db.DeviceProps, deviceID)
}

func (d *SqlDb) CreateDevice(device db.Device) (newDevice db.Device, err error) {
	if device.RDPStatus == "" {
		device.RDPStatus = db.DeviceStatusUnknown
	}
	if device.WinRMStatus == "" {
		device.WinRMStatus = db.DeviceStatusUnknown
	}
	device.Created = tz.Now()

	insertID, err := d.insert(
		"id",
		"insert into project__device ("+
			"project_id, name, ip_address, hostname, "+
			"rdp_status, winrm_status, last_updated, created) values "+
			"(?, ?, ?, ?, ?, ?, ?, ?)",
		device.ProjectID,
		device.Name,
		device.IPAddress,
		device.Hostname,
		device.RDPStatus,
		device.WinRMStatus,
		device.LastUpdated,
		device.Created,
	)
	if err != nil {
		return
	}

	newDevice = device
	newDevice.ID = insertID
	return
}

func (d *SqlDb) UpdateDevice(device db.Device) error {
	_, err := d.exec(
		"update project__device set "+
			"name=?, ip_address=?, hostname=? "+
			"where id=? and project_id=?",
		device.Name,
		device.IPAddress,
		device.Hostname,
		device.ID,
		device.ProjectID,
	)
	return err
}

func (d *SqlDb) UpdateDeviceStatus(projectID, deviceID int, rdp, winrm db.DeviceStatus, refreshed time.Time) error {
	_, err := d.exec(
		"update project__device set rdp_status=?, winrm_status=?, last_updated=? "+
			"where id=? and project_id=?",
		rdp, winrm, refreshed, deviceID, projectID,
	)
	return err
}

func (d *SqlDb) GetDeviceStats(projectID int) (stats db.DeviceStats, err error) {
	type row struct {
		RDP     db.DeviceStatus `db:"rdp_status"`
		WinRM   db.DeviceStatus `db:"winrm_status"`
		Count   int             `db:"cnt"`
	}
	var rows []row
	_, err = d.selectAll(&rows,
		d.PrepareQuery(
			"select rdp_status, winrm_status, count(*) as cnt "+
				"from project__device where project_id=? "+
				"group by rdp_status, winrm_status"),
		projectID,
	)
	if err != nil {
		return
	}

	for _, r := range rows {
		stats.Total += r.Count
		switch r.RDP {
		case db.DeviceStatusOnline:
			stats.RDPOnline += r.Count
		case db.DeviceStatusOffline:
			stats.RDPOffline += r.Count
		}
		switch r.WinRM {
		case db.DeviceStatusOnline:
			stats.WinRMOnline += r.Count
		case db.DeviceStatusOffline:
			stats.WinRMOffline += r.Count
		}
		if r.RDP == db.DeviceStatusUnknown && r.WinRM == db.DeviceStatusUnknown {
			stats.Unknown += r.Count
		}
	}
	return
}

func (d *SqlDb) GetDeviceConfigItems(projectID, deviceID int) ([]db.DeviceConfigItem, error) {
	var items []db.DeviceConfigItem
	_, err := d.selectAll(&items,
		d.PrepareQuery(
			"select i.* from project__device_config_item i "+
				"inner join project__device d on d.id = i.device_id "+
				"where d.project_id=? and i.device_id=? "+
				"order by i.category, i.`key`"),
		projectID, deviceID,
	)
	return items, err
}

// SetDeviceConfigItems replaces the full set of config items for a device.
// The replacement is performed inside a transaction so callers always observe
// a consistent view.
func (d *SqlDb) SetDeviceConfigItems(projectID, deviceID int, items []db.DeviceConfigItem) error {
	// Validate the device belongs to the project before mutating anything.
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return err
	}

	tx, err := d.Sql().Begin()
	if err != nil {
		return err
	}

	if _, err = d.execTx(tx, "delete from project__device_config_item where device_id=?", deviceID); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, it := range items {
		if it.Key == "" {
			continue
		}
		if _, err = d.execTx(tx,
			"insert into project__device_config_item (device_id, category, `key`, value) "+
				"values (?, ?, ?, ?)",
			deviceID, it.Category, it.Key, it.Value,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *SqlDb) GetProjectDeviceSettings(projectID int) (settings db.ProjectDeviceSettings, err error) {
	err = d.selectOne(&settings,
		"select * from project__device_settings where project_id=?",
		projectID,
	)
	if errors.Is(err, db.ErrNotFound) {
		settings = db.ProjectDeviceSettings{ProjectID: projectID}
		err = nil
	}
	return
}

// UpdateProjectDeviceSettings upserts the per-project device settings row.
func (d *SqlDb) UpdateProjectDeviceSettings(s db.ProjectDeviceSettings) error {
	res, err := d.exec(
		"update project__device_settings set "+
			"discover_template_id=?, start_template_id=?, stop_template_id=?, "+
			"restart_template_id=?, status_template_id=?, config_template_id=?, "+
			"status_refresh_interval_min=? "+
			"where project_id=?",
		s.DiscoverTemplateID, s.StartTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.ConfigTemplateID,
		s.StatusRefreshIntervalMin,
		s.ProjectID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	_, err = d.exec(
		"insert into project__device_settings ("+
			"project_id, discover_template_id, start_template_id, stop_template_id, "+
			"restart_template_id, status_template_id, config_template_id, "+
			"status_refresh_interval_min) values (?, ?, ?, ?, ?, ?, ?, ?)",
		s.ProjectID,
		s.DiscoverTemplateID, s.StartTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.ConfigTemplateID,
		s.StatusRefreshIntervalMin,
	)
	return err
}

func (d *SqlDb) MarkProjectStatusRefreshed(projectID int, refreshed time.Time) error {
	_, err := d.exec(
		"update project__device_settings set last_status_refresh_at=? where project_id=?",
		refreshed, projectID,
	)
	return err
}

func (d *SqlDb) GetProjectsDueForStatusRefresh(now time.Time) ([]db.ProjectDeviceSettings, error) {
	var settings []db.ProjectDeviceSettings
	q, args, err := squirrel.Select("*").
		From("project__device_settings").
		Where(squirrel.Gt{"status_refresh_interval_min": 0}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if _, err = d.selectAll(&settings, d.PrepareQuery(q), args...); err != nil {
		return nil, err
	}

	var due []db.ProjectDeviceSettings
	for _, s := range settings {
		if s.StatusRefreshIntervalMin <= 0 {
			continue
		}
		if s.LastStatusRefreshAt == nil ||
			now.Sub(*s.LastStatusRefreshAt) >= time.Duration(s.StatusRefreshIntervalMin)*time.Minute {
			due = append(due, s)
		}
	}
	return due, nil
}
