package db

import (
	"testing"

	"github.com/semaphoreui/semaphore/util"
)

func TestConfig_assignMapToStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Detail struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}

	type User struct {
		Name    string            `json:"name"`
		Age     int               `json:"age"`
		Email   string            `json:"email"`
		Address Address           `json:"address"`
		Details map[string]Detail `json:"details"`
		Tags    []string          `json:"tags"`
	}

	johnData := map[string]any{
		"name":  "John Doe",
		"age":   30,
		"email": "john.doe@example.com",
		"address": map[string]any{
			"street": "123 Main St",
			"city":   "Anytown",
		},
		"details": map[string]any{
			"occupation": map[string]any{
				"value":       "engineer",
				"description": "Works with computers",
			},
			"hobby": map[string]any{
				"value":       "hiking",
				"description": "Enjoys the outdoors",
			},
			"interests": map[string]any{
				"description": "Ho ho ho",
			},
		},
		"tags": "[\"test\"]",
	}

	var john User
	john.Details = make(map[string]Detail)
	john.Details["interests"] = Detail{
		Value:       "politics",
		Description: "Follows current events",
	}

	err := util.AssignMapToStruct(johnData, &john)

	if err != nil {
		t.Fatal(err)
	}

	if john.Details["interests"].Description != "Ho ho ho" {
		t.Errorf("Expected interests description to be 'Ho ho ho' but got %s", john.Details["interests"].Description)
	}

	if john.Details["interests"].Value != "politics" {
		t.Errorf("Expected interests to be politics but got '%s'", john.Details["interests"].Value)
	}

	if len(john.Tags) < 1 {
		t.Fatal("Expected user tags")
	}
}

func TestFilterOptionsForConfigMerge_excludesTDengineBlob(t *testing.T) {
	opts := map[string]string{
		"tdengine.config": `{"enabled":true,"url":"http://x:6041"}`,
		"tdengine_config": `{"enabled":false}`,
		"some.other":      "value",
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
