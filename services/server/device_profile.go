package server

import (
	"strings"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
	log "github.com/sirupsen/logrus"
)

// EnsureDefaultDeviceProfile creates NEWARE profile for a project and assigns orphan devices.
func EnsureDefaultDeviceProfile(store db.Store, projectID int) (db.DeviceProfile, error) {
	p, err := store.GetDeviceProfileByKey(projectID, db.DefaultDeviceProfileKey)
	if err == nil && p.ID > 0 {
		_ = store.AssignDevicesWithoutProfile(projectID, p.ID)
		_ = syncProfileSettingsFromProjectIfNeeded(store, projectID, p.ID)
		return p, nil
	}

	projectSettings, _ := store.GetProjectDeviceSettings(projectID)
	p, err = store.CreateDeviceProfile(db.DeviceProfile{
		ProjectID:  projectID,
		ProfileKey: db.DefaultDeviceProfileKey,
		Name:       "NEWARE",
		Enabled:    true,
		Created:    tz.Now(),
	})
	if err != nil {
		return p, err
	}

	ps := db.ProjectDeviceProfileSettings{
		ProjectID:                               projectID,
		ProfileID:                               p.ID,
		DefaultAnsibleUser:                      projectSettings.DefaultAnsibleUser,
		DefaultAnsiblePassword:                  projectSettings.DefaultAnsiblePassword,
		DefaultAnsibleConnection:                projectSettings.DefaultAnsibleConnection,
		DefaultAnsibleWinRMTransport:            projectSettings.DefaultAnsibleWinRMTransport,
		DefaultAnsibleWinRMScheme:               projectSettings.DefaultAnsibleWinRMScheme,
		DefaultAnsiblePort:                      projectSettings.DefaultAnsiblePort,
		DefaultAnsibleWinRMServerCertValidation: projectSettings.DefaultAnsibleWinRMServerCertValidation,
		DefaultConfigJSON:                       projectSettings.DefaultConfigJSON,
	}
	if err := store.UpdateProjectDeviceProfileSettings(ps); err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("device profile: failed to seed profile settings")
	}

	if err := store.AssignDevicesWithoutProfile(projectID, p.ID); err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("device profile: failed to assign devices")
	}
	if err := SyncDeviceProfileAutoInventory(store, projectID, p.ID); err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("device profile: failed to sync auto inventory")
	}
	return p, nil
}

// syncProfileSettingsFromProjectIfNeeded copies empty profile connection defaults from project settings.
func syncProfileSettingsFromProjectIfNeeded(store db.Store, projectID, profileID int) error {
	ps, err := store.GetProjectDeviceProfileSettings(projectID, profileID)
	if err != nil {
		return err
	}
	if !profileNeedsProjectConnectionDefaults(ps) {
		return nil
	}
	projectSettings, err := store.GetProjectDeviceSettings(projectID)
	if err != nil {
		return err
	}
	MergeProfileSettingsFromProject(&ps, projectSettings)
	return store.UpdateProjectDeviceProfileSettings(ps)
}

func profileNeedsProjectConnectionDefaults(ps db.ProjectDeviceProfileSettings) bool {
	return strings.TrimSpace(ps.DefaultAnsibleUser) == "" ||
		strings.TrimSpace(ps.DefaultAnsiblePassword) == "" ||
		strings.TrimSpace(ps.DefaultAnsibleConnection) == "" ||
		strings.TrimSpace(ps.DefaultAnsibleWinRMTransport) == "" ||
		strings.TrimSpace(ps.DefaultAnsibleWinRMScheme) == "" ||
		ps.DefaultAnsiblePort == 0 ||
		strings.TrimSpace(ps.DefaultAnsibleWinRMServerCertValidation) == "" ||
		strings.TrimSpace(ps.DefaultConfigJSON) == ""
}

// ResolveDeviceProfileSettings returns profile settings, falling back to project-level templates.
func ResolveDeviceProfileSettings(store db.Store, projectID int, device db.Device) (db.DeviceProfile, db.ProjectDeviceProfileSettings, error) {
	if device.DeviceProfileID <= 0 {
		prof, err := EnsureDefaultDeviceProfile(store, projectID)
		if err != nil {
			return db.DeviceProfile{}, db.ProjectDeviceProfileSettings{}, err
		}
		device.DeviceProfileID = prof.ID
	}
	prof, err := store.GetDeviceProfile(projectID, device.DeviceProfileID)
	if err != nil {
		return db.DeviceProfile{}, db.ProjectDeviceProfileSettings{}, err
	}
	ps, err := store.GetProjectDeviceProfileSettings(projectID, device.DeviceProfileID)
	if err != nil {
		return prof, db.ProjectDeviceProfileSettings{}, err
	}
	// Fallback empty template IDs from project settings
	projectSettings, _ := store.GetProjectDeviceSettings(projectID)
	MergeProfileSettingsFromProject(&ps, projectSettings)
	return prof, ps, nil
}

// MergeProfileSettingsFromProject fills empty profile connection defaults from project-level row (legacy DB only).
func MergeProfileSettingsFromProject(ps *db.ProjectDeviceProfileSettings, project db.ProjectDeviceSettings) {
	if strings.TrimSpace(ps.DefaultAnsibleUser) == "" {
		ps.DefaultAnsibleUser = project.DefaultAnsibleUser
	}
	if strings.TrimSpace(ps.DefaultAnsiblePassword) == "" {
		ps.DefaultAnsiblePassword = project.DefaultAnsiblePassword
	}
	if strings.TrimSpace(ps.DefaultAnsibleConnection) == "" {
		ps.DefaultAnsibleConnection = project.DefaultAnsibleConnection
	}
	if strings.TrimSpace(ps.DefaultAnsibleWinRMTransport) == "" {
		ps.DefaultAnsibleWinRMTransport = project.DefaultAnsibleWinRMTransport
	}
	if strings.TrimSpace(ps.DefaultAnsibleWinRMScheme) == "" {
		ps.DefaultAnsibleWinRMScheme = project.DefaultAnsibleWinRMScheme
	}
	if ps.DefaultAnsiblePort == 0 {
		ps.DefaultAnsiblePort = project.DefaultAnsiblePort
	}
	if strings.TrimSpace(ps.DefaultAnsibleWinRMServerCertValidation) == "" {
		ps.DefaultAnsibleWinRMServerCertValidation = project.DefaultAnsibleWinRMServerCertValidation
	}
	if strings.TrimSpace(ps.DefaultConfigJSON) == "" {
		ps.DefaultConfigJSON = project.DefaultConfigJSON
	}
}

func ValidateDeviceHasProfile(device db.Device) error {
	if device.DeviceProfileID <= 0 {
		return &db.ValidationError{Message: "Device has no profile type; assign a profile before running actions"}
	}
	return nil
}

// SeedDeviceProfileSettings creates default per-profile settings row (templates empty until configured).
func SeedDeviceProfileSettings(store db.Store, prof db.DeviceProfile) error {
	ps := db.ProjectDeviceProfileSettings{
		ProjectID: prof.ProjectID,
		ProfileID: prof.ID,
	}
	return store.UpdateProjectDeviceProfileSettings(ps)
}

// GroupDevicesByProfile groups devices by device_profile_id.
func GroupDevicesByProfile(devices []db.Device) map[int][]db.Device {
	m := map[int][]db.Device{}
	for _, d := range devices {
		pid := d.DeviceProfileID
		if pid <= 0 {
			pid = 0
		}
		m[pid] = append(m[pid], d)
	}
	return m
}
