package projects

import "testing"

func TestParseDiscoveredDevicesFromTaskLog(t *testing.T) {
	log := `TASK [输出发现结果 JSON] ***
ok: [localhost] => {
    "msg": "SEMAPHORE_DISCOVERY_JSON=[{\"hostname\":\"host1\",\"ip_address\":\"10.0.0.5\",\"device_status\":\"healthy\",\"rdp_status\":\"online\",\"winrm_status\":\"online\",\"api_status\":\"online\"}]"
}
`
	rows, ok := parseDiscoveredDevicesFromTaskLog(log)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected 1 row, got ok=%v len=%d", ok, len(rows))
	}
	if rows[0].IPAddress != "10.0.0.5" {
		t.Fatalf("unexpected ip: %q", rows[0].IPAddress)
	}
}
