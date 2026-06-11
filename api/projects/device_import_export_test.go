package projects

import (
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestParseDeviceExportIDs(t *testing.T) {
	ids, err := parseDeviceExportIDs("1, 2,2, 3")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("unexpected ids: %v", ids)
	}
	empty, err := parseDeviceExportIDs("")
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty, got %v err=%v", empty, err)
	}
	if _, err := parseDeviceExportIDs("abc"); err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestResolveImportProfileID(t *testing.T) {
	keyToID := map[string]int{"NEWARE": 5, "LAND": 7}
	id, err := resolveImportProfileID(db.DeviceBulkExportRow{ProfileKey: "neware"}, keyToID, 0)
	if err != nil || id != 5 {
		t.Fatalf("profile_key: id=%d err=%v", id, err)
	}
	id, err = resolveImportProfileID(db.DeviceBulkExportRow{DeviceProfileID: 9}, keyToID, 0)
	if err != nil || id != 9 {
		t.Fatalf("device_profile_id: id=%d err=%v", id, err)
	}
	if _, err := resolveImportProfileID(db.DeviceBulkExportRow{}, keyToID, 0); err == nil {
		t.Fatal("expected error when profile missing")
	}
}
