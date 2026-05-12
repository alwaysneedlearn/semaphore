package sql

import (
	"errors"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *SqlDb) GetDevice(projectID int, deviceID int) (device db.Device, err error) {
	err = d.getObject(projectID, db.DeviceProps, deviceID, &device)
	return
}

func deviceLikeEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func applyDeviceListFilters(q squirrel.SelectBuilder, filter *db.DeviceListFilter) squirrel.SelectBuilder {
	if filter == nil {
		return q
	}
	if t := strings.TrimSpace(filter.HostnameSubstring); t != "" {
		q = q.Where("`hostname` LIKE ?", "%"+deviceLikeEscape(t)+"%")
	}
	if t := strings.TrimSpace(filter.IPSubstring); t != "" {
		q = q.Where("`ip_address` LIKE ?", "%"+deviceLikeEscape(t)+"%")
	}
	if t := strings.TrimSpace(filter.DeviceStatus); t != "" {
		q = q.Where("`device_status` = ?", t)
	}
	if t := strings.TrimSpace(filter.RDPStatus); t != "" {
		q = q.Where("`rdp_status` = ?", t)
	}
	if t := strings.TrimSpace(filter.WinRMStatus); t != "" {
		q = q.Where("`winrm_status` = ?", t)
	}
	return q
}

func (d *SqlDb) GetDevices(projectID int, params db.RetrieveQueryParams, filter *db.DeviceListFilter) ([]db.Device, error) {
	var devices []db.Device
	err := d.getObjects(projectID, db.DeviceProps, params, func(q squirrel.SelectBuilder) squirrel.SelectBuilder {
		return applyDeviceListFilters(q, filter)
	}, &devices)
	return devices, err
}

func (d *SqlDb) CountDevices(projectID int, filter *db.DeviceListFilter) (int, error) {
	q := squirrel.Select("count(*)").From("`project__device`").Where("project_id=?", projectID)
	q = applyDeviceListFilters(q, filter)
	query, args, err := q.ToSql()
	if err != nil {
		return 0, err
	}
	cnt, err := d.Sql().SelectInt(d.PrepareQuery(query), args...)
	return int(cnt), err
}

func (d *SqlDb) DeleteDevice(projectID int, deviceID int) error {
	return d.deleteObject(projectID, db.DeviceProps, deviceID)
}

func (d *SqlDb) CreateDevice(device db.Device) (newDevice db.Device, err error) {
	if device.Name == "" {
		device.Name = device.Hostname
	}
	if device.DeviceStatus == "" {
		device.DeviceStatus = db.DeviceStatusUnknown
	}
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
			"project_id, name, ip_address, hostname, ansible_user, ansible_password, ansible_connection, "+
			"ansible_winrm_transport, ansible_winrm_scheme, ansible_port, ansible_winrm_server_cert_validation, "+
			"rdp_user, rdp_password, "+
			"device_status, rdp_status, winrm_status, abnormal_reason, last_updated, created) values "+
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		device.ProjectID,
		device.Name,
		device.IPAddress,
		device.Hostname,
		device.AnsibleUser,
		device.AnsiblePassword,
		device.AnsibleConnection,
		device.AnsibleWinRMTransport,
		device.AnsibleWinRMScheme,
		device.AnsiblePort,
		device.AnsibleWinRMServerCertValidation,
		device.RDPUser,
		device.RDPPassword,
		device.DeviceStatus,
		device.RDPStatus,
		device.WinRMStatus,
		device.AbnormalReason,
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
			"name=?, ip_address=?, hostname=?, ansible_user=?, ansible_password=?, ansible_connection=?, "+
			"ansible_winrm_transport=?, ansible_winrm_scheme=?, ansible_port=?, ansible_winrm_server_cert_validation=?, "+
			"rdp_user=?, rdp_password=?, "+
			"device_status=?, rdp_status=?, winrm_status=?, abnormal_reason=?, last_updated=? "+
			"where id=? and project_id=?",
		device.Name,
		device.IPAddress,
		device.Hostname,
		device.AnsibleUser,
		device.AnsiblePassword,
		device.AnsibleConnection,
		device.AnsibleWinRMTransport,
		device.AnsibleWinRMScheme,
		device.AnsiblePort,
		device.AnsibleWinRMServerCertValidation,
		device.RDPUser,
		device.RDPPassword,
		device.DeviceStatus,
		device.RDPStatus,
		device.WinRMStatus,
		device.AbnormalReason,
		device.LastUpdated,
		device.ID,
		device.ProjectID,
	)
	return err
}

func (d *SqlDb) UpdateDeviceStatus(projectID, deviceID int, rdp, winrm db.DeviceStatus, refreshed time.Time) error {
	deviceStatus := db.DeviceStatusUnknown
	if rdp == db.DeviceStatusOnline && winrm == db.DeviceStatusOnline {
		deviceStatus = db.DeviceStatusHealthy
	} else if rdp == db.DeviceStatusOffline && winrm == db.DeviceStatusOffline {
		deviceStatus = db.DeviceStatusUnhealthy
	}
	_, err := d.exec(
		"update project__device set rdp_status=?, winrm_status=?, device_status=?, last_updated=? "+
			"where id=? and project_id=?",
		rdp, winrm, deviceStatus, refreshed, deviceID, projectID,
	)
	return err
}

func (d *SqlDb) GetDeviceStats(projectID int) (stats db.DeviceStats, err error) {
	type row struct {
		Status db.DeviceStatus `db:"device_status"`
		Count  int             `db:"cnt"`
	}
	var rows []row
	_, err = d.selectAll(&rows,
		d.PrepareQuery(
			"select device_status, count(*) as cnt "+
				"from project__device where project_id=? "+
				"group by device_status"),
		projectID,
	)
	if err != nil {
		return
	}

	for _, r := range rows {
		stats.Total += r.Count
		switch r.Status {
		case db.DeviceStatusHealthy:
			stats.Healthy += r.Count
		case db.DeviceStatusUnhealthy:
			stats.Unhealthy += r.Count
		case db.DeviceStatusChecking:
			stats.Checking += r.Count
		default:
			stats.Unknown += r.Count
		}
	}
	return
}

func (d *SqlDb) UpdateDeviceStatusByHostname(projectID int, hostname string, status db.DeviceStatus, refreshed time.Time) error {
	_, err := d.exec(
		"update project__device set device_status=?, last_updated=? where project_id=? and hostname=?",
		status,
		refreshed,
		projectID,
		hostname,
	)
	return err
}

func (d *SqlDb) UpsertDevicesByIPAddress(projectID int, devices []db.Device) ([]db.Device, error) {
	var saved []db.Device
	for _, dev := range devices {
		ip := strings.TrimSpace(dev.IPAddress)
		if ip == "" {
			continue
		}
		dev.IPAddress = ip

		var existing db.Device
		err := d.selectOne(&existing,
			"select * from project__device where project_id=? and ip_address=?",
			projectID, ip)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		if errors.Is(err, db.ErrNotFound) {
			dev.ProjectID = projectID
			if strings.TrimSpace(dev.Hostname) == "" {
				dev.Hostname = ip
			}
			dev.Name = dev.Hostname
			created, cErr := d.CreateDevice(dev)
			if cErr != nil {
				return nil, cErr
			}
			saved = append(saved, created)
			continue
		}

		existing.IPAddress = ip
		if strings.TrimSpace(dev.Hostname) != "" {
			existing.Hostname = strings.TrimSpace(dev.Hostname)
			existing.Name = existing.Hostname
		}
		existing.AnsibleUser = dev.AnsibleUser
		existing.AnsiblePassword = dev.AnsiblePassword
		existing.AnsibleConnection = dev.AnsibleConnection
		existing.AnsibleWinRMTransport = dev.AnsibleWinRMTransport
		existing.AnsibleWinRMScheme = dev.AnsibleWinRMScheme
		existing.AnsiblePort = dev.AnsiblePort
		existing.AnsibleWinRMServerCertValidation = dev.AnsibleWinRMServerCertValidation
		if dev.DeviceStatus != "" {
			existing.DeviceStatus = dev.DeviceStatus
		}
		if dev.RDPStatus != "" {
			existing.RDPStatus = dev.RDPStatus
		}
		if dev.WinRMStatus != "" {
			existing.WinRMStatus = dev.WinRMStatus
		}
		existing.AbnormalReason = dev.AbnormalReason
		now := tz.Now()
		existing.LastUpdated = &now
		if err = d.UpdateDevice(existing); err != nil {
			return nil, err
		}
		saved = append(saved, existing)
	}
	return saved, nil
}

func (d *SqlDb) GetDeviceStatusCallbackLogs(projectID int, deviceID int, limit int) ([]db.DeviceStatusCallbackLog, error) {
	if limit <= 0 {
		limit = 20
	}
	var logs []db.DeviceStatusCallbackLog
	_, err := d.selectAll(&logs, d.PrepareQuery(
		"select * from project__device_status_callback where project_id=? and device_id=? order by created desc limit ?",
	), projectID, deviceID, limit)
	return logs, err
}

func (d *SqlDb) CreateDeviceStatusCallbackLog(l db.DeviceStatusCallbackLog) (db.DeviceStatusCallbackLog, error) {
	if l.DeviceID != nil {
		if _, err := d.exec(
			d.PrepareQuery("delete from project__device_status_callback where project_id=? and device_id=?"),
			l.ProjectID,
			*l.DeviceID,
		); err != nil {
			return db.DeviceStatusCallbackLog{}, err
		}
	}

	id, err := d.insert("id", "insert into project__device_status_callback ("+
		"project_id, device_id, hostname, status, rdp_status, winrm_status, abnormal_reason, payload, created"+
		") values (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		l.ProjectID, l.DeviceID, l.Hostname, l.Status, l.RDPStatus, l.WinRMStatus, l.AbnormalReason, l.Payload, l.Created,
	)
	if err != nil {
		return db.DeviceStatusCallbackLog{}, err
	}
	l.ID = id
	return l, nil
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
			"default_inventory_id=?, default_ansible_user=?, default_ansible_password=?, default_ansible_connection=?, "+
			"default_ansible_winrm_transport=?, default_ansible_winrm_scheme=?, default_ansible_port=?, default_ansible_winrm_server_cert_validation=?, "+
			"default_config_json=?, status_refresh_interval_min=? "+
			"where project_id=?",
		s.DiscoverTemplateID, s.StartTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.ConfigTemplateID,
		s.DefaultInventoryID, s.DefaultAnsibleUser, s.DefaultAnsiblePassword, s.DefaultAnsibleConnection,
		s.DefaultAnsibleWinRMTransport, s.DefaultAnsibleWinRMScheme, s.DefaultAnsiblePort, s.DefaultAnsibleWinRMServerCertValidation,
		s.DefaultConfigJSON,
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
			"default_inventory_id, default_ansible_user, default_ansible_password, default_ansible_connection, "+
			"default_ansible_winrm_transport, default_ansible_winrm_scheme, default_ansible_port, default_ansible_winrm_server_cert_validation, "+
			"default_config_json, status_refresh_interval_min) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.ProjectID,
		s.DiscoverTemplateID, s.StartTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.ConfigTemplateID,
		s.DefaultInventoryID, s.DefaultAnsibleUser, s.DefaultAnsiblePassword, s.DefaultAnsibleConnection,
		s.DefaultAnsibleWinRMTransport, s.DefaultAnsibleWinRMScheme, s.DefaultAnsiblePort, s.DefaultAnsibleWinRMServerCertValidation,
		s.DefaultConfigJSON,
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
