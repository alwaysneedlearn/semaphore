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
	if err := server.SeedDeviceProfileSettings(helpers.Store(r), created); err != nil {
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
	helpers.WriteJSON(w, http.StatusOK, ps)
}

func UpdateDeviceProfileSettings(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	profileID, err := helpers.GetIntParam("profile_id", w, r)
	if err != nil {
		return
	}
	var ps db.ProjectDeviceProfileSettings
	if !helpers.Bind(w, r, &ps) {
		return
	}
	ps.ProjectID = project.ID
	ps.ProfileID = profileID
	if err := helpers.Store(r).UpdateProjectDeviceProfileSettings(ps); err != nil {
		helpers.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
