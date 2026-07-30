package db

import "time"

const (
	DeviceRDPLaunchPhaseRequested     = "requested"
	DeviceRDPLaunchPhaseHelperFetched = "helper_fetched"
)

// DeviceRDPLaunchLog stores one remote-desktop launch attempt for audit/tracking.
type DeviceRDPLaunchLog struct {
	ID              int        `db:"id" json:"id"`
	ProjectID       int        `db:"project_id" json:"project_id"`
	DeviceID        int        `db:"device_id" json:"device_id"`
	UserID          int        `db:"user_id" json:"user_id"`
	Username        string     `db:"username" json:"username"`
	Phase           string     `db:"phase" json:"phase"`
	Host            string     `db:"host" json:"host"`
	RDPPort         int        `db:"rdp_port" json:"rdp_port"`
	RDPUser         string     `db:"rdp_user" json:"rdp_user"`
	ClientIP        string     `db:"client_ip" json:"client_ip"`
	Created         time.Time  `db:"created" json:"created"`
	HelperFetchedAt *time.Time `db:"helper_fetched_at" json:"helper_fetched_at,omitempty"`
}

// DeviceRDPLaunchLogList is a paginated list of RDP launch logs for one device.
type DeviceRDPLaunchLogList struct {
	Logs  []DeviceRDPLaunchLog `json:"logs"`
	Total int                  `json:"total"`
}
