package projects

import (
	"encoding/json"
	"strings"
)

// deviceConfigCategorizedValues parses profile default_config_json or builds from
// device items shape for Ansible: { "SystemConfig": {"k":"v"}, "Redeliver": {...} }.
// Supports legacy categorized object or {"items":[{"category","key","value","remark"},...]}.
func deviceConfigCategorizedValues(raw string) map[string]map[string]string {
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
	cat := deviceConfigCategorizedValues(raw)
	if cat == nil {
		return nil
	}
	out := make(map[string]any, len(cat))
	for k, v := range cat {
		out[k] = v
	}
	return out
}
