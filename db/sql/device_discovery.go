package sql

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
)

func (d *SqlDb) UpsertDeviceDiscoveryRun(run db.DeviceDiscoveryRun) error {
	if run.Updated.IsZero() {
		run.Updated = time.Now()
	}
	res, err := d.exec(
		"update project__device_discovery_run set "+
			"project_id=?, subnet=?, status=?, devices_json=?, updated=? "+
			"where task_id=?",
		run.ProjectID, run.Subnet, run.Status, run.DevicesJSON, run.Updated, run.TaskID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = d.exec(
		"insert into project__device_discovery_run "+
			"(task_id, project_id, subnet, status, devices_json, updated) values (?, ?, ?, ?, ?, ?)",
		run.TaskID, run.ProjectID, run.Subnet, run.Status, run.DevicesJSON, run.Updated,
	)
	return err
}

func (d *SqlDb) GetDeviceDiscoveryRun(projectID, taskID int) (run db.DeviceDiscoveryRun, err error) {
	err = d.selectOne(&run,
		"select * from project__device_discovery_run where project_id=? and task_id=?",
		projectID, taskID,
	)
	return
}
