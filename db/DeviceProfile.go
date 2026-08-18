package db

import (
	"strings"
	"time"
)

const DefaultDeviceProfileKey = "NEWARE"

// DeviceProfile is a device type within a project (playbook family + template bindings).
type DeviceProfile struct {
	ID         int       `db:"id" json:"id" backup:"-"`
	ProjectID  int       `db:"project_id" json:"project_id" backup:"-"`
	ProfileKey string    `db:"profile_key" json:"profile_key"`
	Name       string    `db:"name" json:"name"`
	Enabled    bool      `db:"enabled" json:"enabled"`
	Created    time.Time `db:"created" json:"created" backup:"-"`
}

func (p DeviceProfile) Validate() error {
	if strings.TrimSpace(p.ProfileKey) == "" {
		return &ValidationError{"Device profile key can not be empty"}
	}
	if strings.TrimSpace(p.Name) == "" {
		return &ValidationError{"Device profile name can not be empty"}
	}
	return nil
}

// ProjectDeviceProfileSettings holds per-profile template bindings and TDengine table name.
type ProjectDeviceProfileSettings struct {
	ProjectID int `db:"project_id" json:"project_id"`
	ProfileID int `db:"profile_id" json:"profile_id"`

	RestartTemplateID    *int `db:"restart_template_id" json:"restart_template_id,omitempty"`
	StatusTemplateID     *int `db:"status_template_id" json:"status_template_id,omitempty"`
	RedeployTemplateID   *int `db:"redeploy_template_id" json:"redeploy_template_id,omitempty"`
	ResendDataTemplateID *int `db:"resend_data_template_id" json:"resend_data_template_id,omitempty"`

	DefaultInventoryID                      *int   `db:"default_inventory_id" json:"default_inventory_id,omitempty"`
	DefaultAnsibleUser                      string `db:"default_ansible_user" json:"default_ansible_user"`
	DefaultAnsiblePassword                  string `db:"default_ansible_password" json:"default_ansible_password"`
	DefaultAnsibleConnection                string `db:"default_ansible_connection" json:"default_ansible_connection"`
	DefaultAnsibleWinRMTransport            string `db:"default_ansible_winrm_transport" json:"default_ansible_winrm_transport"`
	DefaultAnsibleWinRMScheme               string `db:"default_ansible_winrm_scheme" json:"default_ansible_winrm_scheme"`
	DefaultAnsiblePort                      int    `db:"default_ansible_port" json:"default_ansible_port"`
	DefaultAnsibleWinRMServerCertValidation string `db:"default_ansible_winrm_server_cert_validation" json:"default_ansible_winrm_server_cert_validation"`
	DefaultConfigJSON                       string `db:"default_config_json" json:"default_config_json"`

	TDengineStatusTable string `db:"tdengine_status_table" json:"tdengine_status_table"`
}

func (s ProjectDeviceProfileSettings) TemplateIDForAction(action DeviceAction) *int {
	switch action {
	case DeviceActionRestart:
		return s.RestartTemplateID
	case DeviceActionRedeploy:
		return s.RedeployTemplateID
	case DeviceActionStatus:
		return s.StatusTemplateID
	case DeviceActionResendData:
		return s.ResendDataTemplateID
	}
	return nil
}

// EffectiveTDengineStatusTable returns the TDengine table for this profile.
func (s ProjectDeviceProfileSettings) EffectiveTDengineStatusTable(profileKey string) string {
	t := strings.TrimSpace(s.TDengineStatusTable)
	if t != "" {
		return t
	}
	if strings.EqualFold(strings.TrimSpace(profileKey), DefaultDeviceProfileKey) {
		return "status"
	}
	return "status_" + strings.ToLower(strings.TrimSpace(profileKey))
}
