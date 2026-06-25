package db

import (
	"encoding/json"
	"testing"
)

func TestDeviceOperationLogInput_UnmarshalJSON_taskID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    int
	}{
		{name: "int", payload: `{"task_id":814}`, want: 814},
		{name: "string", payload: `{"task_id":"814"}`, want: 814},
		{name: "empty string", payload: `{"task_id":""}`, want: 0},
		{name: "omitted", payload: `{}`, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var in DeviceOperationLogInput
			if err := json.Unmarshal([]byte(tt.payload), &in); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if in.TaskID != tt.want {
				t.Fatalf("TaskID = %d, want %d", in.TaskID, tt.want)
			}
		})
	}
}
