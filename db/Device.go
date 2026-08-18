package db

import (
	"net"
	"strings"
	"time"
)

// DeviceStatus represents the reachability status of a device on a given protocol.
type DeviceStatus string

const (
	DeviceStatusUnknown   DeviceStatus = "unknown"
	DeviceStatusOnline    DeviceStatus = "online"
	DeviceStatusOffline   DeviceStatus = "offline"
	DeviceStatusHealthy   DeviceStatus = "healthy"
	DeviceStatusUnhealthy DeviceStatus = "unhealthy"
	DeviceStatusChecking  DeviceStatus = "checking"
)

// DeviceStatusFromChannelProbes derives aggregate device_status from WinRM and application
// API reachability. RDP is treated as informational only: RDP offline does not downgrade
// healthy when WinRM and API indicate the device is fine. If the app API port is offline,
// the device cannot be considered healthy. If WinRM is offline, automation cannot reach the host.
func DeviceStatusFromChannelProbes(rdp, winrm, api DeviceStatus) DeviceStatus {
	_ = rdp
	if api == DeviceStatusOffline {
		return DeviceStatusUnhealthy
	}
	if winrm == DeviceStatusOffline {
		return DeviceStatusUnhealthy
	}
	if winrm == DeviceStatusOnline {
		return DeviceStatusHealthy
	}
	return DeviceStatusUnknown
}

// DeviceAction is the catalog of operations that can be performed on a device.
// Each action maps to an optional template id stored in ProjectDeviceSettings.
type DeviceAction string

const (
	DeviceActionDiscover   DeviceAction = "discover"
	DeviceActionRestart    DeviceAction = "restart"
	DeviceActionRedeploy   DeviceAction = "redeploy"
	DeviceActionStatus     DeviceAction = "status"
	DeviceActionResendData DeviceAction = "resend_data"
	DeviceActionConfig     DeviceAction = "config"
)

// Device is a managed host belonging to a project.
type Device struct {
	ID                               int          `db:"id" json:"id" backup:"-"`
	ProjectID                        int          `db:"project_id" json:"project_id" backup:"-"`
	Name                             string       `db:"name" json:"name"`
	IPAddress                        string       `db:"ip_address" json:"ip_address"`
	Hostname                         string       `db:"hostname" json:"hostname"`
	AnsibleUser                      string       `db:"ansible_user" json:"ansible_user"`
	AnsiblePassword                  string       `db:"ansible_password" json:"ansible_password"`
	AnsibleConnection                string       `db:"ansible_connection" json:"ansible_connection"`
	AnsibleWinRMTransport            string       `db:"ansible_winrm_transport" json:"ansible_winrm_transport"`
	AnsibleWinRMScheme               string       `db:"ansible_winrm_scheme" json:"ansible_winrm_scheme"`
	AnsiblePort                      int          `db:"ansible_port" json:"ansible_port"`
	AnsibleWinRMServerCertValidation string       `db:"ansible_winrm_server_cert_validation" json:"ansible_winrm_server_cert_validation"`
	RDPUser                          string       `db:"rdp_user" json:"rdp_user"`
	RDPPassword                      string       `db:"rdp_password" json:"rdp_password"`
	RDPPort                          int          `db:"rdp_port" json:"rdp_port"`
	APIPort                          int          `db:"api_port" json:"api_port"`
	DeviceProfileID                  int          `db:"device_profile_id" json:"device_profile_id"`
	DeviceStatus                     DeviceStatus `db:"device_status" json:"device_status"`
	RDPStatus                        DeviceStatus `db:"rdp_status" json:"rdp_status"`
	WinRMStatus                      DeviceStatus `db:"winrm_status" json:"winrm_status"`
	APIStatus                        DeviceStatus `db:"api_status" json:"api_status"`
	AbnormalReason                   *string      `db:"abnormal_reason" json:"abnormal_reason,omitempty"`
	LastUpdated                      *time.Time   `db:"last_updated" json:"last_updated,omitempty"`
	Created                          time.Time    `db:"created" json:"created" backup:"-"`
}

// DeviceBulkExportRow is the portable import/export shape (device edit form fields).
type DeviceBulkExportRow struct {
	IPAddress                        string `json:"ip_address"`
	Hostname                         string `json:"hostname"`
	ProfileKey                       string `json:"profile_key"`
	DeviceProfileID                  int    `json:"device_profile_id,omitempty"`
	RDPUser                          string `json:"rdp_user"`
	RDPPassword                      string `json:"rdp_password"`
	RDPPort                          int    `json:"rdp_port"`
	AnsibleUser                      string `json:"ansible_user"`
	AnsiblePassword                  string `json:"ansible_password"`
	AnsibleConnection                string `json:"ansible_connection"`
	AnsibleWinRMTransport            string `json:"ansible_winrm_transport"`
	AnsibleWinRMScheme               string `json:"ansible_winrm_scheme"`
	AnsiblePort                      int    `json:"ansible_port"`
	APIPort                          int    `json:"api_port"`
	AnsibleWinRMServerCertValidation string `json:"ansible_winrm_server_cert_validation"`
}

// DeviceListFilter narrows the device list for API pagination (substring match for text fields).
type DeviceListFilter struct {
	HostnameSubstring string
	IPSubstring       string
	DeviceStatus      string
	RDPStatus         string
	WinRMStatus       string
	APIStatus         string
	DeviceProfileID   int // 0 = no filter
}

// Validate enforces the device invariants checked at the API/store boundary.
func (d Device) Validate() error {
	if strings.TrimSpace(d.Hostname) == "" {
		return &ValidationError{"Device hostname can not be empty"}
	}
	if d.IPAddress != "" && net.ParseIP(d.IPAddress) == nil {
		return &ValidationError{"Device ip_address must be a valid IPv4/IPv6 address"}
	}
	switch d.DeviceStatus {
	case "", DeviceStatusHealthy, DeviceStatusUnhealthy, DeviceStatusChecking:
	default:
		return &ValidationError{"Device status is invalid"}
	}
	switch d.RDPStatus {
	case "", DeviceStatusOnline, DeviceStatusOffline:
	default:
		return &ValidationError{"Device rdp_status is invalid"}
	}
	switch d.WinRMStatus {
	case "", DeviceStatusOnline, DeviceStatusOffline:
	default:
		return &ValidationError{"Device winrm_status is invalid"}
	}
	switch d.APIStatus {
	case "", DeviceStatusOnline, DeviceStatusOffline:
	default:
		return &ValidationError{"Device api_status is invalid"}
	}
	if d.APIPort != 0 && (d.APIPort < 1 || d.APIPort > 65535) {
		return &ValidationError{"Device api_port must be between 1 and 65535"}
	}
	if d.DeviceProfileID <= 0 {
		return &ValidationError{"Device profile is required"}
	}
	return nil
}

// Default TCP ports for device connectivity (RDP / WinRM over HTTP).
const (
	DefaultDeviceRDPPort     = 3389
	DefaultDeviceAnsiblePort = 5985
	DefaultDeviceAPIPort     = 9002
)

// EffectiveDeviceRDPPort returns a valid RDP TCP port for probes and inventory.
func EffectiveDeviceRDPPort(d Device) int {
	if d.RDPPort > 0 && d.RDPPort <= 65535 {
		return d.RDPPort
	}
	return DefaultDeviceRDPPort
}

// EffectiveDeviceAnsiblePort returns the WinRM/Ansible TCP port for probes and inventory.
func EffectiveDeviceAnsiblePort(d Device, settings ProjectDeviceSettings) int {
	if d.AnsiblePort > 0 && d.AnsiblePort <= 65535 {
		return d.AnsiblePort
	}
	if settings.DefaultAnsiblePort > 0 && settings.DefaultAnsiblePort <= 65535 {
		return settings.DefaultAnsiblePort
	}
	return DefaultDeviceAnsiblePort
}

// EffectiveDeviceAPIPortForInventory returns a positive TCP port to embed as host var `api_port`
// when generating Ansible inventory. Returns 0 when the device has no explicit port so playbooks
// fall back to Variable Group env `API_PORT` then default 9002.
func EffectiveDeviceAPIPortForInventory(d Device) int {
	if d.APIPort > 0 && d.APIPort <= 65535 {
		return d.APIPort
	}
	return 0
}

// EffectiveDeviceAPIPortForExtraVars returns the device api_port for task extra-vars, or 0 when
// unset so JSON consumers treat it as "use env / default" (same semantics as inventory).
func EffectiveDeviceAPIPortForExtraVars(d Device) int {
	return EffectiveDeviceAPIPortForInventory(d)
}

// EffectiveDeviceAPIProbePort returns the TCP port used for API reachability probes (defaults to 9002).
func EffectiveDeviceAPIProbePort(d Device) int {
	if d.APIPort > 0 && d.APIPort <= 65535 {
		return d.APIPort
	}
	return DefaultDeviceAPIPort
}

// MergeDeviceCredentialsOnUpsert copies Ansible/RDP credential fields from incoming onto existing
// only when incoming provides non-empty trimmed values. Discovery import payloads usually omit
// secrets; assigning empty strings would erase manually configured credentials.
func MergeDeviceCredentialsOnUpsert(existing *Device, incoming Device) {
	if strings.TrimSpace(incoming.AnsibleUser) != "" {
		existing.AnsibleUser = incoming.AnsibleUser
	}
	if strings.TrimSpace(incoming.AnsiblePassword) != "" {
		existing.AnsiblePassword = incoming.AnsiblePassword
	}
	if strings.TrimSpace(incoming.RDPUser) != "" {
		existing.RDPUser = incoming.RDPUser
	}
	if strings.TrimSpace(incoming.RDPPassword) != "" {
		existing.RDPPassword = incoming.RDPPassword
	}
}

// MergeDevicePortsOnUpsert copies RDP/Ansible TCP ports when incoming sets valid values.
// Import clears ports to 0 before upsert so normalized defaults do not overwrite stored ports.
func MergeDevicePortsOnUpsert(existing *Device, incoming Device) {
	if incoming.RDPPort > 0 && incoming.RDPPort <= 65535 {
		existing.RDPPort = incoming.RDPPort
	}
	if incoming.AnsiblePort > 0 && incoming.AnsiblePort <= 65535 {
		existing.AnsiblePort = incoming.AnsiblePort
	}
	if incoming.APIPort > 0 && incoming.APIPort <= 65535 {
		existing.APIPort = incoming.APIPort
	}
}

// DeviceConfigItem is a key/value belonging to a device, grouped by Category.
// (device_id, category, key) is unique.
type DeviceConfigItem struct {
	ID       int    `db:"id" json:"id" backup:"-"`
	DeviceID int    `db:"device_id" json:"device_id" backup:"-"`
	Category string `db:"category" json:"category"`
	Key      string `db:"key" json:"key" binding:"required"`
	Value    string `db:"value" json:"value"`
	Remark   string `db:"remark" json:"remark"`
}

// ProjectDeviceSettings holds the per-project mapping of device actions to
// templates plus the periodic refresh interval.
type ProjectDeviceSettings struct {
	ProjectID                               int        `db:"project_id" json:"project_id"`
	DiscoverTemplateID                      *int       `db:"discover_template_id" json:"discover_template_id,omitempty"`
	RestartTemplateID                       *int       `db:"restart_template_id" json:"restart_template_id,omitempty"`
	StatusTemplateID                        *int       `db:"status_template_id" json:"status_template_id,omitempty"`
	ConfigTemplateID                        *int       `db:"config_template_id" json:"config_template_id,omitempty"`
	DefaultInventoryID                      *int       `db:"default_inventory_id" json:"default_inventory_id,omitempty"`
	DefaultAnsibleUser                      string     `db:"default_ansible_user" json:"default_ansible_user"`
	DefaultAnsiblePassword                  string     `db:"default_ansible_password" json:"default_ansible_password"`
	DefaultAnsibleConnection                string     `db:"default_ansible_connection" json:"default_ansible_connection"`
	DefaultAnsibleWinRMTransport            string     `db:"default_ansible_winrm_transport" json:"default_ansible_winrm_transport"`
	DefaultAnsibleWinRMScheme               string     `db:"default_ansible_winrm_scheme" json:"default_ansible_winrm_scheme"`
	DefaultAnsiblePort                      int        `db:"default_ansible_port" json:"default_ansible_port"`
	DefaultAnsibleWinRMServerCertValidation string     `db:"default_ansible_winrm_server_cert_validation" json:"default_ansible_winrm_server_cert_validation"`
	DefaultConfigJSON                       string     `db:"default_config_json" json:"default_config_json"`
	StatusRefreshIntervalMin                int        `db:"status_refresh_interval_min" json:"status_refresh_interval_min"`
	LastStatusRefreshAt                     *time.Time `db:"last_status_refresh_at" json:"last_status_refresh_at,omitempty"`
}

// TemplateIDForAction returns the configured template id for the given action,
// or nil if none is configured.
func (s ProjectDeviceSettings) TemplateIDForAction(action DeviceAction) *int {
	switch action {
	case DeviceActionDiscover:
		return s.DiscoverTemplateID
	case DeviceActionRestart:
		return s.RestartTemplateID
	case DeviceActionStatus:
		return s.StatusTemplateID
	}
	return nil
}

// DeviceStats are the aggregate counts shown in the Devices page header.
type DeviceStats struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Checking  int `json:"checking"`
	Unknown   int `json:"unknown"`
}

type DeviceStatusUpdate struct {
	Hostname       string       `json:"hostname"`
	IPAddress      string       `json:"ip,omitempty"` // optional; bulk callbacks match device by IP when set (see BulkUpdateDeviceStatus)
	Status         DeviceStatus `json:"status"`
	RDPStatus      DeviceStatus `json:"rdp_status,omitempty"`
	WinRMStatus    DeviceStatus `json:"winrm_status,omitempty"`
	APIStatus      DeviceStatus `json:"api_status,omitempty"`
	AbnormalReason *string      `json:"abnormal_reason,omitempty"`
	CheckedAt      *time.Time   `json:"checked_at,omitempty"`
}

type DeviceStatusCallbackLog struct {
	ID             int          `db:"id" json:"id"`
	ProjectID      int          `db:"project_id" json:"project_id"`
	DeviceID       *int         `db:"device_id" json:"device_id,omitempty"`
	Hostname       string       `db:"hostname" json:"hostname"`
	Status         DeviceStatus `db:"status" json:"status"`
	RDPStatus      DeviceStatus `db:"rdp_status" json:"rdp_status"`
	WinRMStatus    DeviceStatus `db:"winrm_status" json:"winrm_status"`
	APIStatus      DeviceStatus `db:"api_status" json:"api_status"`
	AbnormalReason *string      `db:"abnormal_reason" json:"abnormal_reason,omitempty"`
	Payload        string       `db:"payload" json:"payload"`
	Created        time.Time    `db:"created" json:"created"`
}
