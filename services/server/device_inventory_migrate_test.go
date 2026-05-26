package server

import (
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestIsLegacyProjectAutoInventory(t *testing.T) {
	zero := 0
	pid := 3
	if isLegacyProjectAutoInventory(db.Inventory{IsDeviceDefaultAuto: true}) != true {
		t.Fatal("expected legacy without profile id")
	}
	if isLegacyProjectAutoInventory(db.Inventory{IsDeviceDefaultAuto: true, DeviceProfileID: &zero}) != true {
		t.Fatal("expected legacy with zero profile id")
	}
	if isLegacyProjectAutoInventory(db.Inventory{IsDeviceDefaultAuto: true, DeviceProfileID: &pid}) != false {
		t.Fatal("expected non-legacy with profile id")
	}
	if isLegacyProjectAutoInventory(db.Inventory{DeviceProfileID: &pid}) != false {
		t.Fatal("expected non-auto to be non-legacy")
	}
}

func TestLegacyProjectAutoInventories(t *testing.T) {
	leg := db.Inventory{ID: 1, IsDeviceDefaultAuto: true}
	pid := 2
	typed := db.Inventory{ID: 2, IsDeviceDefaultAuto: true, DeviceProfileID: &pid}
	found := legacyProjectAutoInventories([]db.Inventory{leg, typed, {ID: 3}})
	if len(found) != 1 || found[0].ID != 1 {
		t.Fatalf("expected one legacy, got %+v", found)
	}
}
