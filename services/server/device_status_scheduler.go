package server

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
	log "github.com/sirupsen/logrus"
)

// DeviceTaskRunner is the minimal slice of services/tasks.TaskPool the
// scheduler needs to enqueue templates. It avoids importing services/tasks
// here, which would create a cycle (services/tasks → services/server).
type DeviceTaskRunner interface {
	AddTask(taskObj db.Task, userID *int, username string, projectID int, needAlias bool) (db.Task, error)
}

const deviceStatusTickInterval = 60 * time.Second

// DeviceStatusScheduler runs periodically and:
//
//  1. For every project whose ProjectDeviceSettings.StatusRefreshIntervalMin
//     interval has elapsed, performs a TCP port probe (RDP + WinRM) on every
//     device in the project and persists the results.
//  2. Optionally enqueues the project's configured status template via the
//     TaskPool, so user-defined ansible/script-based checks also run.
type DeviceStatusScheduler struct {
	store    db.Store
	taskPool DeviceTaskRunner

	stop chan struct{}
	wg   sync.WaitGroup
}

func NewDeviceStatusScheduler(store db.Store, taskPool DeviceTaskRunner) *DeviceStatusScheduler {
	return &DeviceStatusScheduler{
		store:    store,
		taskPool: taskPool,
		stop:     make(chan struct{}),
	}
}

func (s *DeviceStatusScheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *DeviceStatusScheduler) Stop() {
	close(s.stop)
	s.wg.Wait()
}

func (s *DeviceStatusScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(deviceStatusTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *DeviceStatusScheduler) tick() {
	now := tz.Now()
	due, err := s.store.GetProjectsDueForStatusRefresh(now)
	if err != nil {
		log.WithError(err).Warn("device status: failed to list projects due for refresh")
		return
	}

	for _, settings := range due {
		s.refreshProject(settings, now)
	}
}

func (s *DeviceStatusScheduler) refreshProject(settings db.ProjectDeviceSettings, now time.Time) {
	devices, err := s.store.GetDevices(settings.ProjectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		log.WithError(err).
			WithField("project_id", settings.ProjectID).
			Warn("device status: failed to load devices")
		return
	}

	for _, device := range devices {
		rdp, winrm, refreshed := ProbeDevice(device)
		if err := s.store.UpdateDeviceStatus(
			settings.ProjectID, device.ID, rdp, winrm, refreshed,
		); err != nil {
			log.WithError(err).
				WithField("project_id", settings.ProjectID).
				WithField("device_id", device.ID).
				Warn("device status: failed to persist probe result")
		}
	}

	if err := s.store.MarkProjectStatusRefreshed(settings.ProjectID, now); err != nil {
		log.WithError(err).
			WithField("project_id", settings.ProjectID).
			Warn("device status: failed to record last refresh time")
	}

	if settings.StatusTemplateID == nil || *settings.StatusTemplateID == 0 {
		return
	}
	s.enqueueStatusTemplate(settings, devices)
}

func (s *DeviceStatusScheduler) enqueueStatusTemplate(settings db.ProjectDeviceSettings, devices []db.Device) {
	tpl, err := s.store.GetTemplate(settings.ProjectID, *settings.StatusTemplateID)
	if err != nil {
		log.WithError(err).
			WithField("project_id", settings.ProjectID).
			WithField("template_id", *settings.StatusTemplateID).
			Warn("device status: failed to load status template")
		return
	}

	devicePayload := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		devicePayload = append(devicePayload, map[string]any{
			"id":       d.ID,
			"name":     d.Name,
			"ip":       d.IPAddress,
			"hostname": d.Hostname,
		})
	}
	envBytes, err := json.Marshal(map[string]any{"devices": devicePayload})
	if err != nil {
		log.WithError(err).Warn("device status: failed to marshal devices")
		return
	}

	task := db.Task{
		TemplateID:  tpl.ID,
		ProjectID:   settings.ProjectID,
		Environment: string(envBytes),
		InventoryID: settings.DefaultInventoryID,
	}

	if _, err := s.taskPool.AddTask(task, nil, "device-status-scheduler", settings.ProjectID, tpl.App.NeedTaskAlias()); err != nil {
		log.WithError(err).
			WithField("project_id", settings.ProjectID).
			WithField("template_id", tpl.ID).
			Warn("device status: failed to enqueue status template")
	}
}
