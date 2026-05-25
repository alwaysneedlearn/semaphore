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
		ProjectID:                projectID,
		ProfileID:                  p.ID,
		DiscoverTemplateID:       projectSettings.DiscoverTemplateID,
		StartTemplateID:          projectSettings.StartTemplateID,
		StopTemplateID:           projectSettings.StopTemplateID,
		RestartTemplateID:        projectSettings.RestartTemplateID,
		StatusTemplateID:         projectSettings.StatusTemplateID,
		ConfigTemplateID:         projectSettings.ConfigTemplateID,
		DefaultInventoryID:       projectSettings.DefaultInventoryID,
		DefaultAnsibleUser:       projectSettings.DefaultAnsibleUser,
		DefaultAnsiblePassword:   projectSettings.DefaultAnsiblePassword,
		DefaultAnsibleConnection: projectSettings.DefaultAnsibleConnection,
		DefaultAnsibleWinRMTransport: projectSettings.DefaultAnsibleWinRMTransport,
		DefaultAnsibleWinRMScheme:    projectSettings.DefaultAnsibleWinRMScheme,
		DefaultAnsiblePort:           projectSettings.DefaultAnsiblePort,
		DefaultAnsibleWinRMServerCertValidation: projectSettings.DefaultAnsibleWinRMServerCertValidation,
		DefaultConfigJSON:            projectSettings.DefaultConfigJSON,
		StatusRefreshIntervalMin:     projectSettings.StatusRefreshIntervalMin,
		TDengineStatusTable:          "status",
	}
	if err := store.UpdateProjectDeviceProfileSettings(ps); err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("device profile: failed to seed profile settings")
	}

	if err := store.AssignDevicesWithoutProfile(projectID, p.ID); err != nil {
		log.WithError(err).WithField("project_id", projectID).Warn("device profile: failed to assign devices")
	}
	return p, nil
}

// syncProfileSettingsFromProjectIfNeeded copies project__device_settings into NEWARE profile when templates are unset.
func syncProfileSettingsFromProjectIfNeeded(store db.Store, projectID, profileID int) error {
	ps, err := store.GetProjectDeviceProfileSettings(projectID, profileID)
	if err != nil {
		return err
	}
	needsTemplates := ps.DiscoverTemplateID == nil && ps.StartTemplateID == nil &&
		ps.StopTemplateID == nil && ps.RestartTemplateID == nil &&
		ps.StatusTemplateID == nil && ps.ConfigTemplateID == nil
	if !needsTemplates {
		return nil
	}
	projectSettings, err := store.GetProjectDeviceSettings(projectID)
	if err != nil {
		return err
	}
	ps.DiscoverTemplateID = projectSettings.DiscoverTemplateID
	ps.StartTemplateID = projectSettings.StartTemplateID
	ps.StopTemplateID = projectSettings.StopTemplateID
	ps.RestartTemplateID = projectSettings.RestartTemplateID
	ps.StatusTemplateID = projectSettings.StatusTemplateID
	ps.ConfigTemplateID = projectSettings.ConfigTemplateID
	MergeProfileSettingsFromProject(&ps, projectSettings)
	if ps.TDengineStatusTable == "" {
		ps.TDengineStatusTable = "status"
	}
	return store.UpdateProjectDeviceProfileSettings(ps)
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
	if ps.DefaultInventoryID == nil {
		ps.DefaultInventoryID = project.DefaultInventoryID
	}
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
	empty := db.ProjectDeviceProfileSettings{}
	ps := db.ProjectDeviceProfileSettings{
		ProjectID:           prof.ProjectID,
		ProfileID:           prof.ID,
		TDengineStatusTable: empty.EffectiveTDengineStatusTable(prof.ProfileKey),
	}
	return store.UpdateProjectDeviceProfileSettings(ps)
}

// ProfileSettingsAsProjectDeviceSettings adapts profile settings for probe/enqueue helpers.
func ProfileSettingsAsProjectDeviceSettings(ps db.ProjectDeviceProfileSettings) db.ProjectDeviceSettings {
	return db.ProjectDeviceSettings{
		ProjectID:                               ps.ProjectID,
		DefaultInventoryID:                      ps.DefaultInventoryID,
		DefaultAnsibleUser:                      ps.DefaultAnsibleUser,
		DefaultAnsiblePassword:                  ps.DefaultAnsiblePassword,
		DefaultAnsibleConnection:                ps.DefaultAnsibleConnection,
		DefaultAnsibleWinRMTransport:            ps.DefaultAnsibleWinRMTransport,
		DefaultAnsibleWinRMScheme:               ps.DefaultAnsibleWinRMScheme,
		DefaultAnsiblePort:                      ps.DefaultAnsiblePort,
		DefaultAnsibleWinRMServerCertValidation: ps.DefaultAnsibleWinRMServerCertValidation,
		StatusTemplateID:                        ps.StatusTemplateID,
		StatusRefreshIntervalMin:                ps.StatusRefreshIntervalMin,
		LastStatusRefreshAt:                     ps.LastStatusRefreshAt,
	}
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
