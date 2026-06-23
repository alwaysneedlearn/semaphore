package db

import "time"

const (
	DeviceOperationRestart  = "restart"
	DeviceOperationRedeploy = "redeploy"
	DeviceOperationStatus   = "status"

	DeviceOperationResultSuccess = "success"
	DeviceOperationResultFailed  = "failed"
)

// DeviceOperationStep is one key step in a restart/redeploy run (playbook callback).
type DeviceOperationStep struct {
	Step   string `json:"step"`
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

// DeviceOperationLog is a persisted restart/redeploy history row for one device.
type DeviceOperationLog struct {
	ID        int       `db:"id" json:"id"`
	ProjectID int       `db:"project_id" json:"project_id"`
	DeviceID  int       `db:"device_id" json:"device_id"`
	TaskID    *int      `db:"task_id" json:"task_id,omitempty"`
	Operation string    `db:"operation" json:"operation"`
	Result    string    `db:"result" json:"result"`
	Summary   string    `db:"summary" json:"summary"`
	StepsJSON string    `db:"steps_json" json:"-"`
	Steps     []DeviceOperationStep `db:"-" json:"steps"`
	Created   time.Time `db:"created" json:"created"`
}

// DeviceOperationLogList is a paginated list for the device detail UI.
type DeviceOperationLogList struct {
	Logs  []DeviceOperationLog `json:"logs"`
	Total int                  `json:"total"`
}

// DeviceOperationLogInput is playbook bulk callback payload for one host.
type DeviceOperationLogInput struct {
	Hostname  string                `json:"hostname"`
	IPAddress string                `json:"ip"`
	Operation string                `json:"operation"`
	Result    string                `json:"result"`
	Summary   string                `json:"summary"`
	Steps     []DeviceOperationStep `json:"steps"`
	TaskID    int                   `json:"task_id"`
}
