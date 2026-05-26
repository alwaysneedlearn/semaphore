package bolt

import (
	"strings"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

var deviceDiscoveryHostProps = db.ObjectProps{
	TableName:         "project__device_discovery_host",
	PrimaryColumnName: "id",
}

func (d *BoltDb) UpsertDiscoveredHostsByIP(projectID, taskID int, rows []db.DiscoveredDeviceRow) (int, error) {
	existing, err := d.ListDiscoveredHosts(projectID, 0)
	if err != nil {
		return 0, err
	}
	byIP := map[string]db.DiscoveredHost{}
	for _, h := range existing {
		if ip := strings.TrimSpace(h.IPAddress); ip != "" {
			byIP[ip] = h
		}
	}
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
		rec := db.DiscoveredHost{
			ProjectID:      projectID,
			IPAddress:      ip,
			Hostname:       host,
			DeviceStatus:   devStatus,
			RDPStatus:      rdp,
			WinRMStatus:    winrm,
			APIStatus:      apiSt,
			APIPort:        row.APIPort,
			AbnormalReason: abnormal,
			LastTaskID:     taskID,
			Updated:        now,
		}
		if old, ok := byIP[ip]; ok {
			rec.ID = old.ID
			if err := d.updateObject(projectID, deviceDiscoveryHostProps, rec); err != nil {
				return n, err
			}
			byIP[ip] = rec
		} else {
			created, err := d.createObject(projectID, deviceDiscoveryHostProps, rec)
			if err != nil {
				return n, err
			}
			if h, ok := created.(db.DiscoveredHost); ok {
				byIP[ip] = h
			} else {
				byIP[ip] = rec
			}
		}
		n++
	}
	return n, nil
}

func (d *BoltDb) ListDiscoveredHosts(projectID int, taskID int) (hosts []db.DiscoveredHost, err error) {
	err = d.getObjects(projectID, deviceDiscoveryHostProps, db.RetrieveQueryParams{}, nil, &hosts)
	if err != nil {
		return nil, err
	}
	if taskID <= 0 {
		return hosts, nil
	}
	filtered := make([]db.DiscoveredHost, 0, len(hosts))
	for _, h := range hosts {
		if h.LastTaskID == taskID {
			filtered = append(filtered, h)
		}
	}
	return filtered, nil
}

func (d *BoltDb) DeleteDiscoveredHostsByIP(projectID int, ipAddresses []string) (int, error) {
	hosts, err := d.ListDiscoveredHosts(projectID, 0)
	if err != nil {
		return 0, err
	}
	toDelete := map[string]bool{}
	for _, ip := range ipAddresses {
		if t := strings.TrimSpace(ip); t != "" {
			toDelete[t] = true
		}
	}
	n := 0
	for _, h := range hosts {
		if !toDelete[strings.TrimSpace(h.IPAddress)] {
			continue
		}
		if err := d.deleteObject(projectID, deviceDiscoveryHostProps, intObjectID(h.ID), nil); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
