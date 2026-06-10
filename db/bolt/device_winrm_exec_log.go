package bolt

import "github.com/semaphoreui/semaphore/db"

func (d *BoltDb) GetDeviceWinRMExecLogs(projectID, deviceID, limit, offset int) (db.DeviceWinRMExecLogList, error) {
	return db.DeviceWinRMExecLogList{Logs: []db.DeviceWinRMExecLog{}, Total: 0}, nil
}

func (d *BoltDb) CreateDeviceWinRMExecLog(l db.DeviceWinRMExecLog) (db.DeviceWinRMExecLog, error) {
	return l, nil
}

func (d *BoltDb) DeleteDeviceWinRMExecLog(projectID, deviceID, logID int) error {
	return db.ErrNotFound
}

func (d *BoltDb) DeleteDeviceWinRMExecLogs(projectID, deviceID int, logIDs []int) (int, error) {
	return 0, nil
}

func (d *BoltDb) ClearDeviceWinRMExecLogs(projectID, deviceID int) (int, error) {
	return 0, nil
}
