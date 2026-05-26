package projects

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/semaphoreui/semaphore/db"
)

func parseDiscoveryPutRequest(r *http.Request) (taskID int, devices []db.DiscoveredDeviceRow, err error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, nil, err
	}
	if len(raw) == 0 {
		return 0, nil, fmt.Errorf("empty request body")
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return 0, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	taskID, err = parseFlexibleInt(envelope["task_id"])
	if err != nil {
		return 0, nil, fmt.Errorf("task_id: %w", err)
	}

	devices, err = parseDiscoveryDevicesField(envelope["devices"])
	if err != nil {
		return 0, nil, fmt.Errorf("devices: %w", err)
	}
	return taskID, devices, nil
}

func parseFlexibleInt(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("missing")
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, fmt.Errorf("empty")
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int(f), nil
	}
	return 0, fmt.Errorf("unsupported type")
}

func parseDiscoveryDevicesField(raw json.RawMessage) ([]db.DiscoveredDeviceRow, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rows []db.DiscoveredDeviceRow
	if err := json.Unmarshal(raw, &rows); err == nil {
		return rows, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" || asString == "[]" {
			return nil, nil
		}
		if err := json.Unmarshal([]byte(asString), &rows); err == nil {
			return rows, nil
		}
		// Ansible occasionally double-encodes or sends Python-ish strings; try bracket slice.
		start := strings.Index(asString, "[")
		end := strings.LastIndex(asString, "]")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(asString[start:end+1]), &rows); err == nil {
				return rows, nil
			}
		}
	}
	var loose []map[string]any
	if err := json.Unmarshal(raw, &loose); err == nil {
		return mapDiscoveryDeviceMaps(loose), nil
	}
	return nil, fmt.Errorf("expected JSON array of devices")
}

func mapDiscoveryDeviceMaps(items []map[string]any) []db.DiscoveredDeviceRow {
	out := make([]db.DiscoveredDeviceRow, 0, len(items))
	for _, m := range items {
		row := db.DiscoveredDeviceRow{
			Hostname:       strings.TrimSpace(anyToString(m["hostname"])),
			IPAddress:      strings.TrimSpace(anyToString(m["ip_address"])),
			IP:             strings.TrimSpace(anyToString(m["ip"])),
			DeviceStatus:   strings.TrimSpace(anyToString(m["device_status"])),
			Status:         strings.TrimSpace(anyToString(m["status"])),
			RDPStatus:      strings.TrimSpace(anyToString(m["rdp_status"])),
			WinRMStatus:    strings.TrimSpace(anyToString(m["winrm_status"])),
			APIStatus:      strings.TrimSpace(anyToString(m["api_status"])),
			AbnormalReason: strings.TrimSpace(anyToString(m["abnormal_reason"])),
			APIPort:        anyToInt(m["api_port"]),
		}
		out = append(out, row)
	}
	return out
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func anyToInt(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(t)))
		return n
	}
}
