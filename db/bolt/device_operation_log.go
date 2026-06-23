package bolt

import "github.com/semaphoreui/semaphore/db"

func (d *BoltDb) GetDeviceOperationLogs(projectID, deviceID, limit, offset int) (db.DeviceOperationLogList, error) {
	return db.DeviceOperationLogList{Logs: []db.DeviceOperationLog{}, Total: 0}, nil
}

func (d *BoltDb) CreateDeviceOperationLogs(projectID int, logs []db.DeviceOperationLog) (int, error) {
	return 0, nil
}
