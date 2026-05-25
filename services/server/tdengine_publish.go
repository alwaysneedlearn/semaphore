package server

import (
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tdengine"
	log "github.com/sirupsen/logrus"
)

// PublishProjectStatusSnapshots writes full device snapshots to TDengine per profile table.
func PublishProjectStatusSnapshots(store db.Store, projectID int) {
	cfg := EffectiveTDengineConfig(store)
	client := tdengine.NewClient(cfg)
	if !client.Enabled() {
		return
	}

	devices, err := store.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("tdengine: load devices")
		return
	}

	byProfile := GroupDevicesByProfile(devices)
	for profileID, devs := range byProfile {
		if profileID <= 0 {
			continue
		}
		prof, err := store.GetDeviceProfile(projectID, profileID)
		if err != nil {
			continue
		}
		ps, _ := store.GetProjectDeviceProfileSettings(projectID, profileID)
		table := ps.EffectiveTDengineStatusTable(prof.ProfileKey)
		rows := make([]tdengine.StatusRow, 0, len(devs))
		for _, d := range devs {
			rows = append(rows, tdengine.StatusRow{
				ProjectID:       projectID,
				DeviceID:        d.ID,
				Hostname:        d.Hostname,
				IP:              d.IPAddress,
				Status:          MapDeviceStatusForTDengine(d.DeviceStatus),
				DeviceStatusRaw: string(d.DeviceStatus),
				WinRMStatus:     string(d.WinRMStatus),
				APIStatus:       string(d.APIStatus),
			})
		}
		if err := client.PublishStatusSnapshot(table, rows); err != nil {
			log.WithError(err).
				WithField("project_id", projectID).
				WithField("profile", prof.ProfileKey).
				WithField("table", table).
				Warn("tdengine: publish snapshot failed")
		} else {
			log.WithField("project_id", projectID).
				WithField("profile", prof.ProfileKey).
				WithField("table", table).
				WithField("rows", len(rows)).
				Info("tdengine: published status snapshot")
		}
	}
}
