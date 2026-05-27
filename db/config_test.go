package db

import "testing"

func TestFilterOptionsForConfigMerge_excludesTDengineBlob(t *testing.T) {
	opts := map[string]string{
		"tdengine.config":  `{"enabled":true,"url":"http://x:6041"}`,
		"tdengine_config":  `{"enabled":false}`,
		"some.other":       "value",
	}
	filtered := filterOptionsForConfigMerge(opts)
	if _, ok := filtered["tdengine.config"]; ok {
		t.Fatal("tdengine.config should be excluded")
	}
	if _, ok := filtered["tdengine_config"]; ok {
		t.Fatal("tdengine_config should be excluded")
	}
	if filtered["some.other"] != "value" {
		t.Fatalf("expected other key preserved, got %+v", filtered)
	}
}

func TestConvertFlatToNested_tdengineConfigWouldBreakWithoutFilter(t *testing.T) {
	nested := ConvertFlatToNested(map[string]string{
		"tdengine.config": `{"enabled":true}`,
	})
	td, ok := nested["tdengine"].(map[string]any)
	if !ok {
		t.Fatal("expected tdengine map")
	}
	if _, hasConfig := td["config"]; !hasConfig {
		t.Fatal("dot key creates nested config child — must not merge into util.Config")
	}
}
