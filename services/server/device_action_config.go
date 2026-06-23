package server

import (
	"encoding/json"
	"strings"

	"github.com/semaphoreui/semaphore/db"
)

// DeviceConfigCategorizedValues parses profile default_config_json for Ansible:
// { "SystemConfig": {"k":"v"}, "Redeliver": {...} } or items-array shape.
func DeviceConfigCategorizedValues(raw string) map[string]map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	if itemsRaw, ok := parsed["items"]; ok && len(itemsRaw) > 0 {
		var items []struct {
			Category string `json:"category"`
			Key      string `json:"key"`
			Value    string `json:"value"`
		}
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil
		}
		out := map[string]map[string]string{}
		for _, it := range items {
			cat := strings.TrimSpace(it.Category)
			key := strings.TrimSpace(it.Key)
			if cat == "" || key == "" {
				continue
			}
			if out[cat] == nil {
				out[cat] = map[string]string{}
			}
			out[cat][key] = it.Value
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	out := map[string]map[string]string{}
	for cat, keysRaw := range parsed {
		if cat == "items" || cat == "remarks" {
			continue
		}
		var keys map[string]string
		if err := json.Unmarshal(keysRaw, &keys); err != nil {
			continue
		}
		if len(keys) == 0 {
			continue
		}
		out[cat] = keys
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func defaultConfigForPlaybook(raw string) map[string]any {
	cat := DeviceConfigCategorizedValues(raw)
	if cat == nil {
		return nil
	}
	out := make(map[string]any, len(cat))
	for k, v := range cat {
		out[k] = v
	}
	return out
}

// MergeDefaultConfigForDeviceAction sets extraVars["default_config"] from device-profile
// default_config_json first, then project-level default_config_json.
func MergeDefaultConfigForDeviceAction(extraVars map[string]any, profileDefaultJSON, projectDefaultJSON string) {
	if defaultConfig := defaultConfigForPlaybook(profileDefaultJSON); defaultConfig != nil {
		extraVars["default_config"] = defaultConfig
		return
	}
	if defaultConfig := defaultConfigForPlaybook(projectDefaultJSON); defaultConfig != nil {
		extraVars["default_config"] = defaultConfig
	}
}

// BuildCategorizedDeviceConfig groups device config items for Ansible extra-vars.
func BuildCategorizedDeviceConfig(items []db.DeviceConfigItem) map[string]map[string]string {
	categorized := map[string]map[string]string{}
	for _, it := range items {
		cat := strings.TrimSpace(it.Category)
		if cat == "" {
			cat = "default"
		}
		if categorized[cat] == nil {
			categorized[cat] = map[string]string{}
		}
		categorized[cat][it.Key] = it.Value
	}
	return categorized
}

// MergeRestartRedeployConfigExtraVars adds per-device config maps for Ansible playbooks.
// Single-device actions also receive legacy "config"; bulk uses configs_by_host keyed by IP and hostname.
func MergeRestartRedeployConfigExtraVars(
	extraVars map[string]any,
	devices []db.Device,
	configByDeviceID map[int]map[string]map[string]string,
) {
	if len(devices) == 0 {
		return
	}
	configByHostname := map[string]map[string]map[string]string{}
	configsByHost := map[string]map[string]map[string]string{}
	for _, d := range devices {
		cfg := configByDeviceID[d.ID]
		if cfg == nil {
			continue
		}
		if h := strings.TrimSpace(d.Hostname); h != "" {
			configByHostname[h] = cfg
			configsByHost[h] = cfg
		}
		if ip := strings.TrimSpace(d.IPAddress); ip != "" {
			configsByHost[ip] = cfg
		}
	}
	if len(configByHostname) > 0 {
		extraVars["configs_by_hostname"] = configByHostname
	}
	if len(configsByHost) > 0 {
		extraVars["configs_by_host"] = configsByHost
	}
	if len(devices) == 1 {
		if cfg := configByDeviceID[devices[0].ID]; cfg != nil {
			extraVars["config"] = cfg
		}
	}
}
