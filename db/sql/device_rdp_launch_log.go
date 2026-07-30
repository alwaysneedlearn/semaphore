package sql

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *SqlDb) GetDeviceRDPLaunchLogs(projectID, deviceID, limit, offset int) (db.DeviceRDPLaunchLogList, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return db.DeviceRDPLaunchLogList{}, err
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

	var total int
	if err := d.Sql().QueryRow(
		d.PrepareQuery("select count(*) from project__device_rdp_launch_log where project_id=? and device_id=?"),
		projectID, deviceID,
	).Scan(&total); err != nil {
		return db.DeviceRDPLaunchLogList{}, err
	}

	var logs []db.DeviceRDPLaunchLog
	_, err := d.selectAll(&logs, d.PrepareQuery(
		"select * from project__device_rdp_launch_log "+
			"where project_id=? and device_id=? "+
			"order by created desc, id desc limit ? offset ?",
	), projectID, deviceID, limit, offset)
	if err != nil {
		return db.DeviceRDPLaunchLogList{}, err
	}
	if logs == nil {
		logs = []db.DeviceRDPLaunchLog{}
	}
	return db.DeviceRDPLaunchLogList{Logs: logs, Total: total}, nil
}

func (d *SqlDb) CreateDeviceRDPLaunchLog(l db.DeviceRDPLaunchLog) (db.DeviceRDPLaunchLog, error) {
	if _, err := d.GetDevice(l.ProjectID, l.DeviceID); err != nil {
		return db.DeviceRDPLaunchLog{}, err
	}
	if l.Created.IsZero() {
		l.Created = tz.Now()
	}
	if l.Phase == "" {
		l.Phase = db.DeviceRDPLaunchPhaseRequested
	}
	id, err := d.insert("id", "insert into project__device_rdp_launch_log ("+
		"project_id, device_id, user_id, username, phase, host, rdp_port, rdp_user, client_ip, created, helper_fetched_at"+
		") values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		l.ProjectID, l.DeviceID, l.UserID, l.Username, l.Phase, l.Host, l.RDPPort, l.RDPUser, l.ClientIP, l.Created, l.HelperFetchedAt,
	)
	if err != nil {
		return db.DeviceRDPLaunchLog{}, err
	}
	l.ID = id
	return l, nil
}

func (d *SqlDb) MarkDeviceRDPLaunchHelperFetched(projectID, deviceID, logID int, fetchedAt time.Time) error {
	if fetchedAt.IsZero() {
		fetchedAt = tz.Now()
	}
	res, err := d.exec(
		d.PrepareQuery(
			"update project__device_rdp_launch_log set phase=?, helper_fetched_at=? "+
				"where project_id=? and device_id=? and id=?",
		),
		db.DeviceRDPLaunchPhaseHelperFetched, fetchedAt, projectID, deviceID, logID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return db.ErrNotFound
	}
	return nil
}
