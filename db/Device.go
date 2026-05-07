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

// DeviceAction is the catalog of operations that can be performed on a device.
// Each action maps to an optional template id stored in ProjectDeviceSettings.
type DeviceAction string

const (
	DeviceActionDiscover DeviceAction = "discover"
	DeviceActionStart    DeviceAction = "start"
	DeviceActionStop     DeviceAction = "stop"
	DeviceActionRestart  DeviceAction = "restart"
	DeviceActionStatus   DeviceAction = "status"
	DeviceActionConfig   DeviceAction = "config"
)

// Device is a managed host belonging to a project.
type Device struct {
	ID             int          `db:"id" json:"id" backup:"-"`
	ProjectID      int          `db:"project_id" json:"project_id" backup:"-"`
	Name           string       `db:"name" json:"name"`
	IPAddress      string       `db:"ip_address" json:"ip_address"`
	Hostname       string       `db:"hostname" json:"hostname"`
	DeviceStatus   DeviceStatus `db:"device_status" json:"device_status"`
	RDPStatus      DeviceStatus `db:"rdp_status" json:"rdp_status"`
	WinRMStatus    DeviceStatus `db:"winrm_status" json:"winrm_status"`
	AbnormalReason *string      `db:"abnormal_reason" json:"abnormal_reason,omitempty"`
	LastUpdated    *time.Time   `db:"last_updated" json:"last_updated,omitempty"`
	Created        time.Time    `db:"created" json:"created" backup:"-"`
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
	case "", DeviceStatusUnknown, DeviceStatusHealthy, DeviceStatusUnhealthy, DeviceStatusChecking:
	default:
		return &ValidationError{"Device status is invalid"}
	}
	switch d.RDPStatus {
	case "", DeviceStatusUnknown, DeviceStatusOnline, DeviceStatusOffline:
	default:
		return &ValidationError{"Device rdp_status is invalid"}
	}
	switch d.WinRMStatus {
	case "", DeviceStatusUnknown, DeviceStatusOnline, DeviceStatusOffline:
	default:
		return &ValidationError{"Device winrm_status is invalid"}
	}
	return nil
}

// DeviceConfigItem is a key/value belonging to a device, grouped by Category.
// (device_id, category, key) is unique.
type DeviceConfigItem struct {
	ID       int    `db:"id" json:"id" backup:"-"`
	DeviceID int    `db:"device_id" json:"device_id" backup:"-"`
	Category string `db:"category" json:"category"`
	Key      string `db:"key" json:"key" binding:"required"`
	Value    string `db:"value" json:"value"`
}

// ProjectDeviceSettings holds the per-project mapping of device actions to
// templates plus the periodic refresh interval.
type ProjectDeviceSettings struct {
	ProjectID                int        `db:"project_id" json:"project_id"`
	DiscoverTemplateID       *int       `db:"discover_template_id" json:"discover_template_id,omitempty"`
	StartTemplateID          *int       `db:"start_template_id" json:"start_template_id,omitempty"`
	StopTemplateID           *int       `db:"stop_template_id" json:"stop_template_id,omitempty"`
	RestartTemplateID        *int       `db:"restart_template_id" json:"restart_template_id,omitempty"`
	StatusTemplateID         *int       `db:"status_template_id" json:"status_template_id,omitempty"`
	ConfigTemplateID         *int       `db:"config_template_id" json:"config_template_id,omitempty"`
	StatusRefreshIntervalMin int        `db:"status_refresh_interval_min" json:"status_refresh_interval_min"`
	LastStatusRefreshAt      *time.Time `db:"last_status_refresh_at" json:"last_status_refresh_at,omitempty"`
}

// TemplateIDForAction returns the configured template id for the given action,
// or nil if none is configured.
func (s ProjectDeviceSettings) TemplateIDForAction(action DeviceAction) *int {
	switch action {
	case DeviceActionDiscover:
		return s.DiscoverTemplateID
	case DeviceActionStart:
		return s.StartTemplateID
	case DeviceActionStop:
		return s.StopTemplateID
	case DeviceActionRestart:
		return s.RestartTemplateID
	case DeviceActionStatus:
		return s.StatusTemplateID
	case DeviceActionConfig:
		return s.ConfigTemplateID
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
	Status         DeviceStatus `json:"status"`
	RDPStatus      DeviceStatus `json:"rdp_status,omitempty"`
	WinRMStatus    DeviceStatus `json:"winrm_status,omitempty"`
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
	AbnormalReason *string      `db:"abnormal_reason" json:"abnormal_reason,omitempty"`
	Payload        string       `db:"payload" json:"payload"`
	Created        time.Time    `db:"created" json:"created"`
}
