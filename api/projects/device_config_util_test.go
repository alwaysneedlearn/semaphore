package projects

import "testing"

func TestDeviceConfigCategorizedValuesLegacy(t *testing.T) {
	raw := `{"SystemConfig":{"HistInterval":"15"},"Redeliver":{"startTime":"2026-5-24 10:10:10"}}`
	got := deviceConfigCategorizedValues(raw)
	if got["SystemConfig"]["HistInterval"] != "15" {
		t.Fatalf("SystemConfig: %+v", got)
	}
	if got["Redeliver"]["startTime"] != "2026-5-24 10:10:10" {
		t.Fatalf("Redeliver: %+v", got)
	}
}

func TestDeviceConfigCategorizedValuesItems(t *testing.T) {
	raw := `{"items":[{"category":"SystemConfig","key":"a","value":"1","remark":"note"}]}`
	got := deviceConfigCategorizedValues(raw)
	if got["SystemConfig"]["a"] != "1" {
		t.Fatalf("items: %+v", got)
	}
}
