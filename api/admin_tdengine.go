package api

import (
	"encoding/json"
	"net/http"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tdengine"
	"github.com/semaphoreui/semaphore/services/server"
)

func requireAdmin(w http.ResponseWriter, r *http.Request) (*db.User, bool) {
	user := helpers.GetFromContext(r, "user").(*db.User)
	if user == nil || !user.Admin {
		helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "User must be admin"})
		return nil, false
	}
	return user, true
}

func GetAdminTDengineConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	cfg := server.EffectiveTDengineConfig(helpers.Store(r))
	b, _ := cfg.RedactedJSON()
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	helpers.WriteJSON(w, http.StatusOK, out)
}

func PutAdminTDengineConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	var body tdengine.Config
	if !helpers.Bind(w, r, &body) {
		return
	}
	// Preserve password when UI sends placeholder
	if body.Password == "********" || body.Password == "" {
		prev := server.EffectiveTDengineConfig(helpers.Store(r))
		if body.Password == "" && prev.Password != "" {
			body.Password = prev.Password
		}
	}
	if err := server.SaveTDengineConfig(helpers.Store(r), body); err != nil {
		helpers.WriteError(w, err)
		return
	}
	GetAdminTDengineConfig(w, r)
}

func PostAdminTDengineTest(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	cfg := server.EffectiveTDengineConfig(helpers.Store(r))
	client := tdengine.NewClient(cfg)
	if err := client.TestConnection(); err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
