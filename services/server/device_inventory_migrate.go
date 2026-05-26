package server

import (
	"strings"

	"github.com/semaphoreui/semaphore/db"
	log "github.com/sirupsen/logrus"
)

func isLegacyProjectAutoInventory(inv db.Inventory) bool {
	if !inv.IsDeviceDefaultAuto {
		return false
	}
	return inv.DeviceProfileID == nil || *inv.DeviceProfileID <= 0
}

func legacyProjectAutoInventories(inventories []db.Inventory) []db.Inventory {
	out := make([]db.Inventory, 0)
	for _, inv := range inventories {
		if isLegacyProjectAutoInventory(inv) {
			out = append(out, inv)
		}
	}
	return out
}

func demoteLegacyAutoInventory(store db.Store, inv db.Inventory) error {
	inv.IsDeviceDefaultAuto = false
	if strings.TrimSpace(inv.Name) == LegacyAutoInventoryDisplayName() {
		inv.Name = LegacyAutoInventoryDisplayName() + " (legacy)"
	}
	return store.UpdateInventory(inv)
}

// migrateProjectDeviceInventories demotes project-wide auto rows and rebuilds per-profile autos.
func migrateProjectDeviceInventories(store db.Store, projectID int) error {
	_, err := EnsureDefaultDeviceProfile(store, projectID)
	if err != nil {
		return err
	}

	inventories, err := store.GetInventories(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return err
	}

	legacies := legacyProjectAutoInventories(inventories)
	legacyIDs := map[int]bool{}
	for _, leg := range legacies {
		legacyIDs[leg.ID] = true
		if err := demoteLegacyAutoInventory(store, leg); err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"project_id":   projectID,
			"inventory_id": leg.ID,
		}).Info("device inventory: demoted legacy project auto inventory")
	}

	if len(legacyIDs) > 0 {
		profiles, err := store.GetDeviceProfiles(projectID)
		if err != nil {
			return err
		}
		for _, prof := range profiles {
			ps, err := store.GetProjectDeviceProfileSettings(projectID, prof.ID)
			if err != nil {
				return err
			}
			if ps.DefaultInventoryID == nil || !legacyIDs[*ps.DefaultInventoryID] {
				continue
			}
			ps.DefaultInventoryID = nil
			if err := store.UpdateProjectDeviceProfileSettings(ps); err != nil {
				return err
			}
		}

		settings, err := store.GetProjectDeviceSettings(projectID)
		if err == nil && settings.DefaultInventoryID != nil && legacyIDs[*settings.DefaultInventoryID] {
			log.WithFields(log.Fields{
				"project_id":   projectID,
				"inventory_id": *settings.DefaultInventoryID,
			}).Warn("device inventory: project default_inventory_id pointed at legacy auto; re-select discovery inventory in device discovery settings if needed")
		}
	}

	return SyncAllDeviceProfilesAutoInventory(store, projectID)
}

// MigrateDeviceProfileAutoInventories runs at server start after DB migrations.
func MigrateDeviceProfileAutoInventories(store db.Store) error {
	projects, err := store.GetAllProjects()
	if err != nil {
		return err
	}
	for _, p := range projects {
		if err := migrateProjectDeviceInventories(store, p.ID); err != nil {
			log.WithError(err).WithField("project_id", p.ID).Warn("device inventory: migration failed for project")
			continue
		}
	}
	return nil
}

// ProfileSettingsReferencedLegacyAuto reports whether a profile still points at a legacy project auto row.
func ProfileSettingsReferencedLegacyAuto(ps db.ProjectDeviceProfileSettings, inventories []db.Inventory) bool {
	if ps.DefaultInventoryID == nil || *ps.DefaultInventoryID <= 0 {
		return false
	}
	for _, inv := range inventories {
		if inv.ID != *ps.DefaultInventoryID {
			continue
		}
		return isLegacyProjectAutoInventory(inv)
	}
	return false
}

// LegacyAutoInventoryDisplayName is the former project-wide auto inventory title.
func LegacyAutoInventoryDisplayName() string {
	return strings.TrimSpace(DeviceAutoInventoryGroup + " (auto)")
}
