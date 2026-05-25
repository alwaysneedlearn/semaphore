package server

import (
	"strings"

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
