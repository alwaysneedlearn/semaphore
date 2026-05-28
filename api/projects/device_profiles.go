package projects

import (
	"errors"
	"net/http"
	"strings"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/server"
)

func CreateDeviceProfile(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	var body struct {
		ProfileKey string `json:"profile_key"`
		Name       string `json:"name"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	body.ProfileKey = strings.ToUpper(strings.TrimSpace(body.ProfileKey))
	body.Name = strings.TrimSpace(body.Name)
	p := db.DeviceProfile{
		ProjectID:  project.ID,
		ProfileKey: body.ProfileKey,
		Name:       body.Name,
		Enabled:    true,
	}
	if err := p.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}
	if _, err := helpers.Store(r).GetDeviceProfileByKey(project.ID, body.ProfileKey); err == nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Device profile key already exists in this project",
		})
		return
	} else if err != nil && !errors.Is(err, db.ErrNotFound) {
		helpers.WriteError(w, err)
		return
	}
	created, err := helpers.Store(r).CreateDeviceProfile(p)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	store := helpers.Store(r)
	if err := server.SeedDeviceProfileSettings(store, created); err != nil {
		helpers.WriteError(w, err)
		return
	}
	if err := server.SyncDeviceProfileAutoInventory(store, project.ID, created.ID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, created)
}

func DeleteDeviceProfile(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	profileID, err := helpers.GetIntParam("profile_id", w, r)
	if err != nil {
		return
	}
	store := helpers.Store(r)
	if _, err := store.GetDeviceProfile(project.ID, profileID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	filter := &db.DeviceListFilter{DeviceProfileID: profileID}
	count, err := store.CountDevices(project.ID, filter)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	if count > 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Cannot delete device profile: devices are still assigned to this type",
		})
		return
	}
	if err := store.DeleteDeviceProfile(project.ID, profileID); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type deviceProfileListItem struct {
	db.DeviceProfile
	DeviceCount int `json:"device_count"`
}

func ListDeviceProfiles(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	store := helpers.Store(r)
	_, _ = server.EnsureDefaultDeviceProfile(store, project.ID)
	profiles, err := store.GetDeviceProfiles(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	out := make([]deviceProfileListItem, 0, len(profiles))
	for _, p := range profiles {
		n, err := store.CountDevices(project.ID, &db.DeviceListFilter{DeviceProfileID: p.ID})
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		out = append(out, deviceProfileListItem{DeviceProfile: p, DeviceCount: n})
	}
	helpers.WriteJSON(w, http.StatusOK, out)
}

// deviceProfileSettingsView is per-type template/config only (connection defaults are project-level).
type deviceProfileSettingsView struct {
	ProjectID                int    `json:"project_id"`
	ProfileID                int    `json:"profile_id"`
	StartTemplateID          *int   `json:"start_template_id,omitempty"`
	StopTemplateID           *int   `json:"stop_template_id,omitempty"`
	RestartTemplateID        *int   `json:"restart_template_id,omitempty"`
	StatusTemplateID         *int   `json:"status_template_id,omitempty"`
	CheckRestartRedeployTemplateID *int `json:"check_restart_redeploy_template_id,omitempty"`
	DefaultInventoryID       *int   `json:"default_inventory_id,omitempty"`
	DefaultConfigJSON        string `json:"default_config_json"`
	StatusRefreshIntervalMin int    `json:"status_refresh_interval_min"`
}

func profileSettingsToView(ps db.ProjectDeviceProfileSettings) deviceProfileSettingsView {
	return deviceProfileSettingsView{
		ProjectID:                ps.ProjectID,
		ProfileID:                ps.ProfileID,
		StartTemplateID:          ps.StartTemplateID,
		StopTemplateID:           ps.StopTemplateID,
		RestartTemplateID:        ps.RestartTemplateID,
		StatusTemplateID:         ps.StatusTemplateID,
		CheckRestartRedeployTemplateID: ps.CheckRestartRedeployTemplateID,
		DefaultInventoryID:       ps.DefaultInventoryID,
		DefaultConfigJSON:        ps.DefaultConfigJSON,
		StatusRefreshIntervalMin: ps.StatusRefreshIntervalMin,
	}
}

func applyProfileSettingsView(existing *db.ProjectDeviceProfileSettings, body deviceProfileSettingsView) {
	existing.StartTemplateID = body.StartTemplateID
	existing.StopTemplateID = body.StopTemplateID
	existing.RestartTemplateID = body.RestartTemplateID
	existing.StatusTemplateID = body.StatusTemplateID
	existing.CheckRestartRedeployTemplateID = body.CheckRestartRedeployTemplateID
	existing.DefaultInventoryID = body.DefaultInventoryID
	existing.DefaultConfigJSON = body.DefaultConfigJSON
	existing.StatusRefreshIntervalMin = body.StatusRefreshIntervalMin
}

func GetDeviceProfileSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	profileID, err := helpers.GetIntParam("profile_id", w, r)
	if err != nil {
		return
	}
	ps, err := helpers.Store(r).GetProjectDeviceProfileSettings(project.ID, profileID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profileSettingsToView(ps))
}

func UpdateDeviceProfileSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	profileID, err := helpers.GetIntParam("profile_id", w, r)
	if err != nil {
		return
	}
	var body deviceProfileSettingsView
	if !helpers.Bind(w, r, &body) {
		return
	}
	store := helpers.Store(r)
	existing, err := store.GetProjectDeviceProfileSettings(project.ID, profileID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	applyProfileSettingsView(&existing, body)
	existing.ProjectID = project.ID
	existing.ProfileID = profileID
	if err := store.UpdateProjectDeviceProfileSettings(existing); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
