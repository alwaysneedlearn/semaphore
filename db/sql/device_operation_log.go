package sql

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

const deviceOperationLogRetention = 30 * 24 * time.Hour

func (d *SqlDb) GetDeviceOperationLogs(projectID, deviceID, limit, offset int) (db.DeviceOperationLogList, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return db.DeviceOperationLogList{}, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	cutoff := tz.Now().Add(-deviceOperationLogRetention)

	var total int
	if err := d.Sql().QueryRow(
		d.PrepareQuery("select count(*) from project__device_operation_log where project_id=? and device_id=? and created>=?"),
		projectID, deviceID, cutoff,
	).Scan(&total); err != nil {
		return db.DeviceOperationLogList{}, err
	}

	var logs []db.DeviceOperationLog
	_, err := d.selectAll(&logs, d.PrepareQuery(
		"select * from project__device_operation_log "+
			"where project_id=? and device_id=? and created>=? "+
			"order by created desc, id desc limit ? offset ?",
	), projectID, deviceID, cutoff, limit, offset)
	if err != nil {
		return db.DeviceOperationLogList{}, err
	}
	for i := range logs {
		logs[i].Steps = decodeOperationSteps(logs[i].StepsJSON)
	}
	if logs == nil {
		logs = []db.DeviceOperationLog{}
	}
	return db.DeviceOperationLogList{Logs: logs, Total: total}, nil
}

func decodeOperationSteps(raw string) []db.DeviceOperationStep {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []db.DeviceOperationStep{}
	}
	var steps []db.DeviceOperationStep
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return []db.DeviceOperationStep{}
	}
	return steps
}

func encodeOperationSteps(steps []db.DeviceOperationStep) string {
	if len(steps) == 0 {
		return "[]"
	}
	b, err := json.Marshal(steps)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func (d *SqlDb) CreateDeviceOperationLogs(projectID int, logs []db.DeviceOperationLog) (int, error) {
	if len(logs) == 0 {
		return 0, nil
	}
	now := tz.Now()
	inserted := 0
	for _, l := range logs {
		if l.DeviceID <= 0 {
			continue
		}
		stepsJSON := l.StepsJSON
		if stepsJSON == "" {
			stepsJSON = encodeOperationSteps(l.Steps)
		}
		created := l.Created
		if created.IsZero() {
			created = now
		}
		_, err := d.insert("id", "insert into project__device_operation_log ("+
			"project_id, device_id, task_id, operation, result, summary, steps_json, created"+
			") values (?, ?, ?, ?, ?, ?, ?, ?)",
			projectID, l.DeviceID, l.TaskID, l.Operation, l.Result, l.Summary, stepsJSON, created,
		)
		if err != nil {
			return inserted, err
		}
		inserted++
	}
	cutoff := now.Add(-deviceOperationLogRetention)
	_, _ = d.exec(
		d.PrepareQuery("delete from project__device_operation_log where project_id=? and created<?"),
		projectID, cutoff,
	)
	return inserted, nil
}
