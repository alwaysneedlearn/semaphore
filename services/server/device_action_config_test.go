package server

import (
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestDeviceConfigCategorizedValuesLegacy(t *testing.T) {
	raw := `{"SystemConfig":{"HistInterval":"15"},"Redeliver":{"startTime":"2026-5-24 10:10:10"}}`
	got := DeviceConfigCategorizedValues(raw)
	if got["SystemConfig"]["HistInterval"] != "15" {
		t.Fatalf("SystemConfig: %+v", got)
	}
	if got["Redeliver"]["startTime"] != "2026-5-24 10:10:10" {
		t.Fatalf("Redeliver: %+v", got)
	}
}

func TestDeviceConfigCategorizedValuesItems(t *testing.T) {
	raw := `{"items":[{"category":"SystemConfig","key":"a","value":"1","remark":"note"}]}`
	got := DeviceConfigCategorizedValues(raw)
	if got["SystemConfig"]["a"] != "1" {
		t.Fatalf("items: %+v", got)
	}
}

func TestMergeDefaultConfigForDeviceActionProfileFirst(t *testing.T) {
	extra := map[string]any{}
	MergeDefaultConfigForDeviceAction(extra,
		`{"SystemConfig":{"HistInterval":"99"}}`,
		`{"SystemConfig":{"HistInterval":"15"}}`,
	)
	dc, ok := extra["default_config"].(map[string]any)
	if !ok {
		t.Fatalf("default_config: %#v", extra["default_config"])
	}
	sc, ok := dc["SystemConfig"].(map[string]string)
	if !ok || sc["HistInterval"] != "99" {
		t.Fatalf("profile default: %#v", dc)
	}
}

func TestMergeDefaultConfigForDeviceActionProjectFallback(t *testing.T) {
	extra := map[string]any{}
	MergeDefaultConfigForDeviceAction(extra, "", `{"SystemConfig":{"HistInterval":"15"}}`)
	dc := extra["default_config"].(map[string]any)
	sc := dc["SystemConfig"].(map[string]string)
	if sc["HistInterval"] != "15" {
		t.Fatalf("project fallback: %#v", dc)
	}
}

func TestMergeRestartRedeployConfigExtraVarsSingle(t *testing.T) {
	dev := db.Device{ID: 7, Hostname: "host-a", IPAddress: "10.0.0.7"}
	cfg := map[string]map[string]string{"SystemConfig": {"k": "v"}}
	extra := map[string]any{}
	MergeRestartRedeployConfigExtraVars(extra, []db.Device{dev}, map[int]map[string]map[string]string{
		7: cfg,
	})
	if extra["config"] == nil {
		t.Fatal("expected legacy config for single device")
	}
	byHost := extra["configs_by_host"].(map[string]map[string]map[string]string)
	if byHost["10.0.0.7"]["SystemConfig"]["k"] != "v" {
		t.Fatalf("configs_by_host ip: %#v", byHost)
	}
}
