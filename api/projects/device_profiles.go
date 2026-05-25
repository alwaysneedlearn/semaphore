package projects

import (
	"net/http"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/server"
)

func ListDeviceProfiles(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	_, _ = server.EnsureDefaultDeviceProfile(helpers.Store(r), project.ID)
	profiles, err := helpers.Store(r).GetDeviceProfiles(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profiles)
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
	projectSettings, _ := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	server.MergeProfileSettingsFromProject(&ps, projectSettings)
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
