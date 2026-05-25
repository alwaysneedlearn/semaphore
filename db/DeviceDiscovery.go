package db

import "time"

// DeviceDiscoveryRunStatus tracks whether playbook callback has populated results.
const (
	DeviceDiscoveryRunPending = "pending"
	DeviceDiscoveryRunReady     = "ready"
)

// DeviceDiscoveryRun stores scan results for one discover task (callback replaces log parsing).
type DeviceDiscoveryRun struct {
	TaskID      int       `db:"task_id" json:"task_id"`
	ProjectID   int       `db:"project_id" json:"project_id"`
	Subnet      string    `db:"subnet" json:"subnet"`
	Status      string    `db:"status" json:"status"`
	DevicesJSON string    `db:"devices_json" json:"-"`
	Updated     time.Time `db:"updated" json:"updated"`
}

// DiscoveredHost is a persisted discovery list row (upserted by project_id + ip_address).
type DiscoveredHost struct {
	ID             int        `db:"id" json:"id"`
	ProjectID      int        `db:"project_id" json:"project_id"`
	IPAddress      string     `db:"ip_address" json:"ip_address"`
	Hostname       string     `db:"hostname" json:"hostname"`
	DeviceStatus   string     `db:"device_status" json:"device_status"`
	RDPStatus      string     `db:"rdp_status" json:"rdp_status"`
	WinRMStatus    string     `db:"winrm_status" json:"winrm_status"`
	APIStatus      string     `db:"api_status" json:"api_status"`
	APIPort        int        `db:"api_port" json:"api_port,omitempty"`
	AbnormalReason *string    `db:"abnormal_reason" json:"abnormal_reason,omitempty"`
	LastTaskID     int        `db:"last_task_id" json:"last_task_id,omitempty"`
	Updated        time.Time  `db:"updated" json:"updated"`
}

// ToDiscoveredDeviceRow maps a stored host to API/playbook JSON shape.
func (h DiscoveredHost) ToDiscoveredDeviceRow() DiscoveredDeviceRow {
	row := DiscoveredDeviceRow{
		Hostname:     h.Hostname,
		IPAddress:    h.IPAddress,
		DeviceStatus: h.DeviceStatus,
		RDPStatus:    h.RDPStatus,
		WinRMStatus:  h.WinRMStatus,
		APIStatus:    h.APIStatus,
		APIPort:      h.APIPort,
	}
	if h.AbnormalReason != nil {
		row.AbnormalReason = *h.AbnormalReason
	}
	return row
}

// DiscoveredDeviceRow is one host reported by the discovery playbook callback.
type DiscoveredDeviceRow struct {
	Hostname       string `json:"hostname"`
	IPAddress      string `json:"ip_address"`
	IP             string `json:"ip,omitempty"`
	DeviceStatus   string `json:"device_status,omitempty"`
	Status         string `json:"status,omitempty"`
	RDPStatus      string `json:"rdp_status,omitempty"`
	WinRMStatus    string `json:"winrm_status,omitempty"`
	APIStatus      string `json:"api_status,omitempty"`
	APIPort        int    `json:"api_port,omitempty"`
	AbnormalReason string `json:"abnormal_reason,omitempty"`
}
