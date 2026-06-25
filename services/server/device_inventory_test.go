package server

import (
	"strings"
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestBuildDeviceInventoryLineIncludesDeviceHostname(t *testing.T) {
	dev := db.Device{
		IPAddress: "10.33.31.139",
		Hostname:  "JSC1306XHXW011",
		Name:      "JSC1306XHXW011",
	}
	line := BuildDeviceInventoryLine(dev, db.ProjectDeviceSettings{})
	if !strings.Contains(line, "10.33.31.139") {
		t.Fatalf("expected inventory host IP in line: %q", line)
	}
	if !strings.Contains(line, "device_hostname=JSC1306XHXW011") {
		t.Fatalf("expected device_hostname in line: %q", line)
	}
	if !strings.Contains(line, "device_name=JSC1306XHXW011") {
		t.Fatalf("expected device_name in line: %q", line)
	}
}
