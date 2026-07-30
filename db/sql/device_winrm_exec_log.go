package sql

import (
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/semaphoreui/semaphore/db"
)

func (d *SqlDb) GetDeviceWinRMExecLogs(projectID, deviceID, limit, offset int) (db.DeviceWinRMExecLogList, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return db.DeviceWinRMExecLogList{}, err
	}
	if limit <= 0 || limit > db.DeviceAuditLogRetainLimit {
		limit = db.DeviceAuditLogRetainLimit
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := d.Sql().QueryRow(
		d.PrepareQuery("select count(*) from project__device_winrm_exec_log where project_id=? and device_id=?"),
		projectID, deviceID,
	).Scan(&total); err != nil {
		return db.DeviceWinRMExecLogList{}, err
	}

	var logs []db.DeviceWinRMExecLog
	_, err := d.selectAll(&logs, d.PrepareQuery(
		"select * from project__device_winrm_exec_log "+
			"where project_id=? and device_id=? "+
			"order by created desc, id desc limit ? offset ?",
	), projectID, deviceID, limit, offset)
	if err != nil {
		return db.DeviceWinRMExecLogList{}, err
	}
	if logs == nil {
		logs = []db.DeviceWinRMExecLog{}
	}
	return db.DeviceWinRMExecLogList{Logs: logs, Total: total}, nil
}

func (d *SqlDb) CreateDeviceWinRMExecLog(l db.DeviceWinRMExecLog) (db.DeviceWinRMExecLog, error) {
	if _, err := d.GetDevice(l.ProjectID, l.DeviceID); err != nil {
		return db.DeviceWinRMExecLog{}, err
	}
	id, err := d.insert("id", "insert into project__device_winrm_exec_log ("+
		"project_id, device_id, user_id, username, credential_mode, shell, command, "+
		"ok, exit_code, error_code, error_message, stdout, stderr, output_truncated, "+
		"duration_ms, resolved_host, resolved_port, resolved_user, created"+
		") values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		l.ProjectID, l.DeviceID, l.UserID, l.Username, l.CredentialMode, l.Shell, l.Command,
		l.OK, l.ExitCode, l.ErrorCode, l.ErrorMessage, l.Stdout, l.Stderr, l.OutputTruncated,
		l.DurationMS, l.ResolvedHost, l.ResolvedPort, l.ResolvedUser, l.Created,
	)
	if err != nil {
		return db.DeviceWinRMExecLog{}, err
	}
	l.ID = id
	_ = d.pruneDeviceAuditLogs("project__device_winrm_exec_log", l.ProjectID, l.DeviceID, db.DeviceAuditLogRetainLimit)
	return l, nil
}

type deviceAuditLogIDRow struct {
	ID int `db:"id"`
}

func (d *SqlDb) pruneDeviceAuditLogs(table string, projectID, deviceID, keep int) error {
	if keep <= 0 {
		keep = db.DeviceAuditLogRetainLimit
	}
	var keepRows []deviceAuditLogIDRow
	_, err := d.selectAll(&keepRows, d.PrepareQuery(
		"select id from "+table+" where project_id=? and device_id=? order by created desc, id desc limit ?",
	), projectID, deviceID, keep)
	if err != nil {
		return err
	}
	if len(keepRows) == 0 {
		return nil
	}
	args := make([]any, 0, 2+len(keepRows))
	args = append(args, projectID, deviceID)
	placeholders := make([]string, 0, len(keepRows))
	for _, row := range keepRows {
		placeholders = append(placeholders, "?")
		args = append(args, row.ID)
	}
	_, err = d.exec(
		d.PrepareQuery(
			"delete from "+table+" where project_id=? and device_id=? and id not in ("+
				strings.Join(placeholders, ",")+
				")",
		),
		args...,
	)
	return err
}

func (d *SqlDb) DeleteDeviceWinRMExecLog(projectID, deviceID, logID int) error {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return err
	}
	res, err := d.exec(
		d.PrepareQuery("delete from project__device_winrm_exec_log where project_id=? and device_id=? and id=?"),
		projectID, deviceID, logID,
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

func (d *SqlDb) DeleteDeviceWinRMExecLogs(projectID, deviceID int, logIDs []int) (int, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return 0, err
	}
	if len(logIDs) == 0 {
		return 0, nil
	}
	ids := make([]int, 0, len(logIDs))
	for _, id := range logIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}

	q := squirrel.Delete("project__device_winrm_exec_log").
		Where(squirrel.Eq{"project_id": projectID, "device_id": deviceID}).
		Where(squirrel.Eq{"id": ids})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return 0, err
	}
	res, err := d.exec(d.PrepareQuery(sqlStr), args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (d *SqlDb) ClearDeviceWinRMExecLogs(projectID, deviceID int) (int, error) {
	if _, err := d.GetDevice(projectID, deviceID); err != nil {
		return 0, err
	}
	res, err := d.exec(
		d.PrepareQuery("delete from project__device_winrm_exec_log where project_id=? and device_id=?"),
		projectID, deviceID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
