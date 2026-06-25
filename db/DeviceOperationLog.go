package db

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DeviceOperationRestart   = "restart"
	DeviceOperationRedeploy  = "redeploy"
	DeviceOperationStatus    = "status"
	DeviceOperationResendData = "resend_data"

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

// UnmarshalJSON accepts task_id as JSON number or numeric string (Ansible often quotes ints).
func (in *DeviceOperationLogInput) UnmarshalJSON(data []byte) error {
	type payload struct {
		Hostname  string                `json:"hostname"`
		IPAddress string                `json:"ip"`
		Operation string                `json:"operation"`
		Result    string                `json:"result"`
		Summary   string                `json:"summary"`
		Steps     []DeviceOperationStep `json:"steps"`
		TaskID    json.RawMessage       `json:"task_id"`
	}
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	in.Hostname = p.Hostname
	in.IPAddress = p.IPAddress
	in.Operation = p.Operation
	in.Result = p.Result
	in.Summary = p.Summary
	in.Steps = p.Steps
	if len(p.TaskID) == 0 {
		return nil
	}
	var tid int
	if err := json.Unmarshal(p.TaskID, &tid); err == nil {
		in.TaskID = tid
		return nil
	}
	var tidStr string
	if err := json.Unmarshal(p.TaskID, &tidStr); err == nil {
		tidStr = strings.TrimSpace(tidStr)
		if tidStr == "" {
			in.TaskID = 0
			return nil
		}
		n, err := strconv.Atoi(tidStr)
		if err != nil {
			return fmt.Errorf("task_id: %w", err)
		}
		in.TaskID = n
		return nil
	}
	return fmt.Errorf("task_id must be a number or numeric string")
}
