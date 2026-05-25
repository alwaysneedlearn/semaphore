package bolt

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
)

var deviceDiscoveryRunProps = db.ObjectProps{
	TableName:         "project__device_discovery_run",
	PrimaryColumnName: "task_id",
}

func (d *BoltDb) UpsertDeviceDiscoveryRun(run db.DeviceDiscoveryRun) error {
	if run.Updated.IsZero() {
		run.Updated = time.Now()
	}
	existing, err := d.GetDeviceDiscoveryRun(run.ProjectID, run.TaskID)
	if err != nil && err != db.ErrNotFound {
		return err
	}
	if err == db.ErrNotFound || existing.TaskID == 0 {
		_, err = d.createObject(run.ProjectID, deviceDiscoveryRunProps, run)
		return err
	}
	return d.updateObject(run.ProjectID, deviceDiscoveryRunProps, run)
}

func (d *BoltDb) GetDeviceDiscoveryRun(projectID, taskID int) (run db.DeviceDiscoveryRun, err error) {
	err = d.getObject(projectID, deviceDiscoveryRunProps, intObjectID(taskID), &run)
	return
}
