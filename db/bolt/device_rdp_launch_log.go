package bolt

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
)

func (d *BoltDb) GetDeviceRDPLaunchLogs(projectID, deviceID, limit, offset int) (db.DeviceRDPLaunchLogList, error) {
	return db.DeviceRDPLaunchLogList{Logs: []db.DeviceRDPLaunchLog{}, Total: 0}, nil
}

func (d *BoltDb) CreateDeviceRDPLaunchLog(l db.DeviceRDPLaunchLog) (db.DeviceRDPLaunchLog, error) {
	return l, nil
}

func (d *BoltDb) MarkDeviceRDPLaunchHelperFetched(projectID, deviceID, logID int, fetchedAt time.Time) error {
	return nil
}
