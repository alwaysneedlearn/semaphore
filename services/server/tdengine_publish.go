package server

import (
	"fmt"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tdengine"
	log "github.com/sirupsen/logrus"
)

// PublishSnapshotResult summarizes one manual or automatic TDengine publish run.
type PublishSnapshotResult struct {
	Projects int `json:"projects"`
	Tables   int `json:"tables"`
	Rows     int `json:"rows"`
}

// PublishProjectStatusSnapshots writes full device snapshots to TDengine per profile table.
func PublishProjectStatusSnapshots(store db.Store, projectID int) (PublishSnapshotResult, error) {
	cfg := EffectiveTDengineConfig(store)
	client := tdengine.NewClient(cfg)
	if !client.Enabled() {
		return PublishSnapshotResult{}, fmt.Errorf("tdengine is disabled or url is empty")
	}

	devices, err := store.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return PublishSnapshotResult{}, err
	}

	res := PublishSnapshotResult{Projects: 1}
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
		if len(rows) == 0 {
			continue
		}
		if err := client.PublishStatusSnapshot(table, rows); err != nil {
			log.WithError(err).
				WithField("project_id", projectID).
				WithField("profile", prof.ProfileKey).
				WithField("table", table).
				Warn("tdengine: publish snapshot failed")
			return res, err
		}
		res.Tables++
		res.Rows += len(rows)
		log.WithField("project_id", projectID).
			WithField("profile", prof.ProfileKey).
			WithField("table", table).
			WithField("rows", len(rows)).
			Info("tdengine: published status snapshot")
	}
	return res, nil
}

// PublishAllProjectStatusSnapshots publishes every project when projectID is 0.
func PublishAllProjectStatusSnapshots(store db.Store, projectID int) (PublishSnapshotResult, error) {
	if projectID > 0 {
		return PublishProjectStatusSnapshots(store, projectID)
	}
	projects, err := store.GetAllProjects()
	if err != nil {
		return PublishSnapshotResult{}, err
	}
	total := PublishSnapshotResult{}
	var firstErr error
	for _, p := range projects {
		part, err := PublishProjectStatusSnapshots(store, p.ID)
		total.Projects++
		total.Tables += part.Tables
		total.Rows += part.Rows
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return total, firstErr
}

// MaybePublishAfterBulkStatusCallback runs async TDengine snapshot when auto_sync_on_bulk is enabled.
func MaybePublishAfterBulkStatusCallback(store db.Store, projectID int) {
	cfg := EffectiveTDengineConfig(store)
	if !cfg.Enabled || !cfg.AutoSyncOnBulk {
		return
	}
	go func() {
		if _, err := PublishProjectStatusSnapshots(store, projectID); err != nil {
			log.WithError(err).WithField("project_id", projectID).Warn("tdengine: auto sync after bulk callback")
		}
	}()
}
