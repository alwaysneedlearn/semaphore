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
