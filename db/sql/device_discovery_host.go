package sql

import (
	"errors"
	"strings"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

func (d *SqlDb) UpsertDiscoveredHostsByIP(projectID, taskID int, rows []db.DiscoveredDeviceRow) (int, error) {
	now := tz.Now()
	n := 0
	for _, row := range rows {
		ip := strings.TrimSpace(row.IPAddress)
		if ip == "" {
			ip = strings.TrimSpace(row.IP)
		}
		if ip == "" {
			continue
		}
		host := strings.TrimSpace(row.Hostname)
		if host == "" {
			host = ip
		}
		devStatus := strings.TrimSpace(row.DeviceStatus)
		if devStatus == "" {
			devStatus = strings.TrimSpace(row.Status)
		}
		if devStatus == "" {
			devStatus = string(db.DeviceStatusUnknown)
		}
		rdp := strings.TrimSpace(row.RDPStatus)
		if rdp == "" {
			rdp = string(db.DeviceStatusOffline)
		}
		winrm := strings.TrimSpace(row.WinRMStatus)
		if winrm == "" {
			winrm = string(db.DeviceStatusOffline)
		}
		apiSt := strings.TrimSpace(row.APIStatus)
		if apiSt == "" {
			apiSt = string(db.DeviceStatusOffline)
		}
		var abnormal *string
		if ar := strings.TrimSpace(row.AbnormalReason); ar != "" {
			abnormal = &ar
		}

		var existing db.DiscoveredHost
		err := d.selectOne(&existing,
			"select * from project__device_discovery_host where project_id=? and ip_address=?",
			projectID, ip)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return n, err
		}

		if errors.Is(err, db.ErrNotFound) {
			_, err = d.exec(
				"insert into project__device_discovery_host ("+
					"project_id, ip_address, hostname, device_status, rdp_status, winrm_status, api_status, "+
					"api_port, abnormal_reason, last_task_id, updated) "+
					"values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				projectID, ip, host, devStatus, rdp, winrm, apiSt,
				row.APIPort, abnormal, taskID, now,
			)
			if err != nil {
				return n, err
			}
			n++
			continue
		}

		_, err = d.exec(
			"update project__device_discovery_host set "+
				"hostname=?, device_status=?, rdp_status=?, winrm_status=?, api_status=?, "+
				"api_port=?, abnormal_reason=?, last_task_id=?, updated=? "+
				"where project_id=? and ip_address=?",
			host, devStatus, rdp, winrm, apiSt,
			row.APIPort, abnormal, taskID, now,
			projectID, ip,
		)
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (d *SqlDb) ListDiscoveredHosts(projectID int, taskID int) (hosts []db.DiscoveredHost, err error) {
	if taskID > 0 {
		_, err = d.selectAll(&hosts,
			"select * from project__device_discovery_host where project_id=? and last_task_id=? order by ip_address",
			projectID, taskID)
	} else {
		_, err = d.selectAll(&hosts,
			"select * from project__device_discovery_host where project_id=? order by ip_address",
			projectID)
	}
	return
}
