package projects

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/task_logger"
	"github.com/semaphoreui/semaphore/services/server"
)

const discoveryJSONLogMarker = "SEMAPHORE_DISCOVERY_JSON="

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// SyncDiscoveryResultsFromTaskOutput parses SEMAPHORE_DISCOVERY_JSON= from task logs when API callback was skipped.
func SyncDiscoveryResultsFromTaskOutput(store db.Store, projectID, taskID int) (int, error) {
	if taskID <= 0 {
		return 0, nil
	}
	task, err := store.GetTask(projectID, taskID)
	if err != nil {
		return 0, err
	}
	if task.Status != task_logger.TaskSuccessStatus {
		return 0, nil
	}

	outputs, err := store.GetTaskOutputs(projectID, taskID, db.RetrieveQueryParams{})
	if err != nil {
		return 0, err
	}
	var buf strings.Builder
	for _, o := range outputs {
		buf.WriteString(o.Output)
	}
	rows, ok := parseDiscoveredDevicesFromTaskLog(buf.String())
	if !ok || len(rows) == 0 {
		return 0, nil
	}

	n, err := store.UpsertDiscoveredHostsByIP(projectID, taskID, rows)
	if err != nil {
		return 0, err
	}

	subnet := ""
	if run, runErr := store.GetDeviceDiscoveryRun(projectID, taskID); runErr == nil {
		subnet = run.Subnet
	}
	_ = store.UpsertDeviceDiscoveryRun(db.DeviceDiscoveryRun{
		TaskID:      taskID,
		ProjectID:   projectID,
		Subnet:      subnet,
		Status:      db.DeviceDiscoveryRunReady,
		DevicesJSON: "[]",
		Updated:     time.Now(),
	})
	return n, nil
}

func parseDiscoveredDevicesFromTaskLog(logText string) ([]db.DiscoveredDeviceRow, bool) {
	clean := ansiEscapeRE.ReplaceAllString(logText, "")
	idx := strings.LastIndex(clean, discoveryJSONLogMarker)
	if idx < 0 {
		return nil, false
	}
	rest := strings.TrimSpace(clean[idx+len(discoveryJSONLogMarker):])
	if rest == "" {
		return nil, false
	}
	if strings.HasPrefix(rest, `"`) {
		var unquoted string
		if err := json.Unmarshal([]byte(rest), &unquoted); err == nil {
			rest = unquoted
		} else {
			rest = strings.Trim(rest, `"`)
		}
	}
	start := strings.Index(rest, "[")
	end := strings.LastIndex(rest, "]")
	if start < 0 || end <= start {
		return nil, false
	}
	payload := strings.ReplaceAll(rest[start:end+1], `\"`, `"`)
	var rows []db.DiscoveredDeviceRow
	if err := json.Unmarshal([]byte(payload), &rows); err != nil {
		return nil, false
	}
	normalized := normalizeDiscoveredDeviceRows(rows)
	return normalized, len(normalized) > 0
}

func discoveryCallbackWarning(merged map[string]any) string {
	if !server.HasPlaybookCallbackToken(merged) {
		return "missing_semaphore_api_token"
	}
	return ""
}
