package sql

import (
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *SqlDb) GetDeviceProfiles(projectID int) ([]db.DeviceProfile, error) {
	var profiles []db.DeviceProfile
	err := d.getObjects(projectID, db.DeviceProfileProps, db.RetrieveQueryParams{}, nil, &profiles)
	return profiles, err
}

func (d *SqlDb) GetDeviceProfile(projectID, profileID int) (db.DeviceProfile, error) {
	var p db.DeviceProfile
	err := d.getObject(projectID, db.DeviceProfileProps, profileID, &p)
	return p, err
}

func (d *SqlDb) GetDeviceProfileByKey(projectID int, key string) (db.DeviceProfile, error) {
	var p db.DeviceProfile
	q, args, err := squirrel.Select("*").
		From("`project__device_profile`").
		Where("project_id=?", projectID).
		Where("profile_key=?", key).
		Limit(1).
		ToSql()
	if err != nil {
		return p, err
	}
	err = d.selectOne(&p, d.PrepareQuery(q), args...)
	return p, err
}

func (d *SqlDb) DeleteDeviceProfile(projectID, profileID int) error {
	_, err := d.exec(
		"delete from project__device_profile_settings where project_id=? and profile_id=?",
		projectID, profileID,
	)
	if err != nil {
		return err
	}
	return d.deleteObject(projectID, db.DeviceProfileProps, profileID)
}

func (d *SqlDb) CreateDeviceProfile(p db.DeviceProfile) (db.DeviceProfile, error) {
	p.Created = tz.Now()
	id, err := d.insert("id",
		"insert into project__device_profile (project_id, profile_key, name, enabled, created) values (?, ?, ?, ?, ?)",
		p.ProjectID, p.ProfileKey, p.Name, p.Enabled, p.Created)
	if err != nil {
		return p, err
	}
	p.ID = id
	return p, nil
}

const projectDeviceProfileSettingsColumns = "" +
	"project_id, profile_id, discover_template_id, stop_template_id, restart_template_id, " +
	"status_template_id, redeploy_template_id, check_restart_template_id, config_template_id, resend_data_template_id, " +
	"default_inventory_id, default_ansible_user, default_ansible_password, default_ansible_connection, " +
	"default_ansible_winrm_transport, default_ansible_winrm_scheme, default_ansible_port, " +
	"default_ansible_winrm_server_cert_validation, default_config_json, " +
	"status_refresh_interval_min, last_status_refresh_at, tdengine_status_table"

func (d *SqlDb) GetProjectDeviceProfileSettings(projectID, profileID int) (db.ProjectDeviceProfileSettings, error) {
	var s db.ProjectDeviceProfileSettings
	err := d.Sql().SelectOne(&s, d.PrepareQuery(
		"select "+projectDeviceProfileSettingsColumns+" from project__device_profile_settings where project_id=? and profile_id=?",
	), projectID, profileID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return db.ProjectDeviceProfileSettings{ProjectID: projectID, ProfileID: profileID}, nil
		}
		return s, err
	}
	return s, nil
}

func (d *SqlDb) UpdateProjectDeviceProfileSettings(s db.ProjectDeviceProfileSettings) error {
	res, err := d.exec(
		"update project__device_profile_settings set "+
			"discover_template_id=?, stop_template_id=?, "+
			"restart_template_id=?, status_template_id=?, redeploy_template_id=?, check_restart_template_id=?, config_template_id=?, resend_data_template_id=?, "+
			"default_inventory_id=?, default_ansible_user=?, default_ansible_password=?, default_ansible_connection=?, "+
			"default_ansible_winrm_transport=?, default_ansible_winrm_scheme=?, default_ansible_port=?, default_ansible_winrm_server_cert_validation=?, "+
			"default_config_json=?, status_refresh_interval_min=?, tdengine_status_table=? "+
			"where project_id=? and profile_id=?",
		s.DiscoverTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.RedeployTemplateID, s.CheckRestartTemplateID, s.ConfigTemplateID, s.ResendDataTemplateID,
		s.DefaultInventoryID, s.DefaultAnsibleUser, s.DefaultAnsiblePassword, s.DefaultAnsibleConnection,
		s.DefaultAnsibleWinRMTransport, s.DefaultAnsibleWinRMScheme, s.DefaultAnsiblePort, s.DefaultAnsibleWinRMServerCertValidation,
		s.DefaultConfigJSON, s.StatusRefreshIntervalMin, s.TDengineStatusTable,
		s.ProjectID, s.ProfileID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		return nil
	}
	_, err = d.exec(
		"insert into project__device_profile_settings ("+
			"project_id, profile_id, discover_template_id, stop_template_id, "+
			"restart_template_id, status_template_id, redeploy_template_id, check_restart_template_id, config_template_id, resend_data_template_id, "+
			"default_inventory_id, default_ansible_user, default_ansible_password, default_ansible_connection, "+
			"default_ansible_winrm_transport, default_ansible_winrm_scheme, default_ansible_port, default_ansible_winrm_server_cert_validation, "+
			"default_config_json, status_refresh_interval_min, tdengine_status_table) "+
			"values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.ProjectID, s.ProfileID,
		s.DiscoverTemplateID, s.StopTemplateID,
		s.RestartTemplateID, s.StatusTemplateID, s.RedeployTemplateID, s.CheckRestartTemplateID, s.ConfigTemplateID, s.ResendDataTemplateID,
		s.DefaultInventoryID, s.DefaultAnsibleUser, s.DefaultAnsiblePassword, s.DefaultAnsibleConnection,
		s.DefaultAnsibleWinRMTransport, s.DefaultAnsibleWinRMScheme, s.DefaultAnsiblePort, s.DefaultAnsibleWinRMServerCertValidation,
		s.DefaultConfigJSON, s.StatusRefreshIntervalMin, s.TDengineStatusTable,
	)
	return err
}

func (d *SqlDb) AssignDevicesWithoutProfile(projectID, profileID int) error {
	_, err := d.exec(
		"update project__device set device_profile_id=? where project_id=? and (device_profile_id is null or device_profile_id=0)",
		profileID, projectID,
	)
	return err
}
