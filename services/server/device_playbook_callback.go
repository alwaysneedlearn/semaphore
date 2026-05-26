package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/util"
)

// InjectPlaybookCallbackVars merges SEMAPHORE_API_TOKEN / SEMAPHORE_URL into task extra-vars
// when configured on the server (util.Config.EnvVars or web_host).
func InjectPlaybookCallbackVars(merged map[string]any) {
	if merged == nil {
		return
	}
	if _, ok := merged["SEMAPHORE_API_TOKEN"]; !ok && util.Config.EnvVars != nil {
		if t := strings.TrimSpace(util.Config.EnvVars["SEMAPHORE_DEVICE_CALLBACK_API_TOKEN"]); t != "" {
			merged["SEMAPHORE_API_TOKEN"] = t
		} else if t := strings.TrimSpace(util.Config.EnvVars["SEMAPHORE_API_TOKEN"]); t != "" {
			merged["SEMAPHORE_API_TOKEN"] = t
		}
	}
	if _, ok := merged["SEMAPHORE_URL"]; !ok {
		if util.Config.EnvVars != nil {
			if u := strings.TrimSpace(util.Config.EnvVars["SEMAPHORE_URL"]); u != "" {
				merged["SEMAPHORE_URL"] = strings.TrimSuffix(u, "/")
			}
		}
		if _, ok2 := merged["SEMAPHORE_URL"]; !ok2 {
			if wh := strings.TrimSpace(util.Config.WebHost); wh != "" {
				merged["SEMAPHORE_URL"] = strings.TrimSuffix(wh, "/")
			}
		}
	}
}

// HasPlaybookCallbackToken reports whether device playbook API callbacks can authenticate.
func HasPlaybookCallbackToken(merged map[string]any) bool {
	if merged != nil {
		if t, ok := merged["SEMAPHORE_API_TOKEN"]; ok {
			if strings.TrimSpace(fmt.Sprint(t)) != "" {
				return true
			}
		}
	}
	if util.Config.EnvVars == nil {
		return false
	}
	if t := strings.TrimSpace(util.Config.EnvVars["SEMAPHORE_DEVICE_CALLBACK_API_TOKEN"]); t != "" {
		return true
	}
	if t := strings.TrimSpace(util.Config.EnvVars["SEMAPHORE_API_TOKEN"]); t != "" {
		return true
	}
	return false
}

func environmentHasCallbackToken(env db.Environment) bool {
	if strings.TrimSpace(env.JSON) != "" && env.JSON != "{}" {
		var partial map[string]any
		if err := json.Unmarshal([]byte(env.JSON), &partial); err == nil {
			if t, ok := partial["SEMAPHORE_API_TOKEN"]; ok && strings.TrimSpace(fmt.Sprint(t)) != "" {
				return true
			}
		}
	}
	if env.ENV != nil && strings.TrimSpace(*env.ENV) != "" && *env.ENV != "{}" {
		var partial map[string]string
		if err := json.Unmarshal([]byte(*env.ENV), &partial); err == nil {
			if strings.TrimSpace(partial["SEMAPHORE_API_TOKEN"]) != "" {
				return true
			}
		}
	}
	return false
}

// TemplateHasPlaybookCallbackToken checks Variable Groups linked on the template (JSON or ENV tab).
func TemplateHasPlaybookCallbackToken(store db.Store, projectID, templateID int) bool {
	if templateID <= 0 {
		return false
	}
	envIDs, err := store.GetTemplateEnvironments(projectID, templateID)
	if err != nil || len(envIDs) == 0 {
		return false
	}
	for _, envID := range envIDs {
		env, err := store.GetEnvironment(projectID, envID)
		if err != nil {
			continue
		}
		if environmentHasCallbackToken(env) {
			return true
		}
	}
	return false
}

// DiscoveryCallbackTokenConfigured is true when token is available via server env_vars or the discover template's Variable Groups.
func DiscoveryCallbackTokenConfigured(store db.Store, projectID int, discoverTemplateID *int) bool {
	if HasPlaybookCallbackToken(nil) {
		return true
	}
	if discoverTemplateID == nil || *discoverTemplateID <= 0 {
		return false
	}
	return TemplateHasPlaybookCallbackToken(store, projectID, *discoverTemplateID)
}

// ApplySemaphoreTaskIDToTask injects semaphore_task_id into task JSON environment before the runner starts.
// Device playbooks already receive semaphore_project_id; task id must be present for discovery/status callbacks.
func ApplySemaphoreTaskIDToTask(store db.Store, task *db.Task) error {
	if task == nil || task.ID <= 0 || strings.TrimSpace(task.Environment) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(task.Environment), &m); err != nil {
		return nil
	}
	if _, ok := m["semaphore_project_id"]; !ok {
		return nil
	}
	if tid, ok := m["semaphore_task_id"]; ok {
		if strings.TrimSpace(fmt.Sprint(tid)) != "" && fmt.Sprint(tid) != "0" {
			return nil
		}
	}
	m["semaphore_task_id"] = task.ID
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	task.Environment = string(b)
	return store.UpdateTask(*task)
}
