package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/semaphoreui/semaphore/db"
)

const DeviceAutoInventoryGroup = "windows_hosts"

// DeviceProfileAutoInventoryName is the display name for per-type auto inventories.
func DeviceProfileAutoInventoryName(profile db.DeviceProfile) string {
	label := strings.TrimSpace(profile.Name)
	if label == "" {
		label = strings.TrimSpace(profile.ProfileKey)
	}
	return fmt.Sprintf("%s (auto: %s)", DeviceAutoInventoryGroup, label)
}

// BuildDeviceInventoryLine renders one windows_hosts inventory line (project defaults fill blanks).
func BuildDeviceInventoryLine(dev db.Device, settings db.ProjectDeviceSettings) string {
	user := dev.AnsibleUser
	if user == "" {
		user = settings.DefaultAnsibleUser
	}
	password := dev.AnsiblePassword
	if password == "" {
		password = settings.DefaultAnsiblePassword
	}
	connection := dev.AnsibleConnection
	if connection == "" {
		connection = settings.DefaultAnsibleConnection
	}
	if connection == "" {
		connection = "winrm"
	}
	transport := dev.AnsibleWinRMTransport
	if transport == "" {
		transport = settings.DefaultAnsibleWinRMTransport
	}
	if transport == "" {
		transport = "basic"
	}
	scheme := dev.AnsibleWinRMScheme
	if scheme == "" {
		scheme = settings.DefaultAnsibleWinRMScheme
	}
	if scheme == "" {
		scheme = "http"
	}
	port := db.EffectiveDeviceAnsiblePort(dev, settings)
	certValidation := dev.AnsibleWinRMServerCertValidation
	if certValidation == "" {
		certValidation = settings.DefaultAnsibleWinRMServerCertValidation
	}
	if certValidation == "" {
		certValidation = "ignore"
	}

	inventoryHost := strings.TrimSpace(dev.IPAddress)
	parts := []string{inventoryHost}
	if dev.IPAddress != "" {
		parts = append(parts, "ansible_host="+dev.IPAddress)
	}
	if user != "" {
		parts = append(parts, "ansible_user="+user)
	}
	if password != "" {
		parts = append(parts, "ansible_password="+password)
	}
	parts = append(parts, "ansible_connection="+connection)
	parts = append(parts, "ansible_winrm_transport="+transport)
	parts = append(parts, "ansible_winrm_scheme="+scheme)
	parts = append(parts, "ansible_port="+strconv.Itoa(port))
	parts = append(parts, "ansible_winrm_server_cert_validation="+certValidation)
	parts = append(parts, "rdp_port="+strconv.Itoa(db.EffectiveDeviceRDPPort(dev)))
	if hn := strings.TrimSpace(dev.Hostname); hn != "" {
		parts = append(parts, "device_hostname="+hn)
	}
	if name := strings.TrimSpace(dev.Name); name != "" {
		parts = append(parts, "device_name="+name)
	}
	if ap := db.EffectiveDeviceAPIPortForInventory(dev); ap > 0 {
		parts = append(parts, "api_port="+strconv.Itoa(ap))
	}
	return strings.Join(parts, " ")
}

// RenderWindowsInventory builds static inventory text for the given devices.
func RenderWindowsInventory(devices []db.Device, settings db.ProjectDeviceSettings) string {
	var b strings.Builder
	b.WriteString("[" + DeviceAutoInventoryGroup + "]\n")
	for _, dev := range devices {
		if strings.TrimSpace(dev.IPAddress) == "" {
			continue
		}
		b.WriteString(BuildDeviceInventoryLine(dev, settings))
		b.WriteString("\n")
	}
	return b.String()
}

func devicesForProfile(all []db.Device, profileID int) []db.Device {
	if profileID <= 0 {
		return nil
	}
	out := make([]db.Device, 0)
	for _, d := range all {
		if d.DeviceProfileID == profileID {
			out = append(out, d)
		}
	}
	return out
}

func findProfileAutoInventory(inventories []db.Inventory, profileID int) *db.Inventory {
	for i := range inventories {
		inv := &inventories[i]
		if !inv.IsDeviceDefaultAuto {
			continue
		}
		if inv.DeviceProfileID == nil || *inv.DeviceProfileID != profileID {
			continue
		}
		return inv
	}
	return nil
}

// EnsureDeviceProfileAutoInventory creates or updates the auto inventory for one device type.
func EnsureDeviceProfileAutoInventory(store db.Store, projectID, profileID int) (db.Inventory, error) {
	if profileID <= 0 {
		return db.Inventory{}, &db.ValidationError{Message: "device profile id is required"}
	}
	profile, err := store.GetDeviceProfile(projectID, profileID)
	if err != nil {
		return db.Inventory{}, err
	}
	settings, err := store.GetProjectDeviceSettings(projectID)
	if err != nil {
		return db.Inventory{}, err
	}
	allDevices, err := store.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return db.Inventory{}, err
	}
	content := RenderWindowsInventory(devicesForProfile(allDevices, profileID), settings)

	inventories, err := store.GetInventories(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return db.Inventory{}, err
	}
	name := DeviceProfileAutoInventoryName(profile)
	if existing := findProfileAutoInventory(inventories, profileID); existing != nil {
		existing.Name = name
		existing.Type = db.InventoryStatic
		existing.Inventory = content
		if err = store.UpdateInventory(*existing); err != nil {
			return db.Inventory{}, err
		}
		return *existing, nil
	}

	pid := profileID
	inv, err := store.CreateInventory(db.Inventory{
		ProjectID:           projectID,
		Name:                name,
		Type:                db.InventoryStatic,
		Inventory:           content,
		IsDeviceDefaultAuto: true,
		DeviceProfileID:     &pid,
	})
	if err != nil {
		return db.Inventory{}, err
	}
	return inv, nil
}

func linkProfileDefaultInventoryToAuto(store db.Store, projectID, profileID int, autoInv db.Inventory) error {
	ps, err := store.GetProjectDeviceProfileSettings(projectID, profileID)
	if err != nil {
		return err
	}
	if ps.DefaultInventoryID != nil && *ps.DefaultInventoryID > 0 {
		if *ps.DefaultInventoryID == autoInv.ID {
			return nil
		}
		existing, err := store.GetInventory(projectID, *ps.DefaultInventoryID)
		if err == nil && !existing.IsDeviceDefaultAuto {
			return nil
		}
	}
	id := autoInv.ID
	ps.DefaultInventoryID = &id
	return store.UpdateProjectDeviceProfileSettings(ps)
}

// SyncDeviceProfileAutoInventory refreshes one profile auto inventory and links it when unset.
func SyncDeviceProfileAutoInventory(store db.Store, projectID, profileID int) error {
	if profileID <= 0 {
		return nil
	}
	inv, err := EnsureDeviceProfileAutoInventory(store, projectID, profileID)
	if err != nil {
		return err
	}
	return linkProfileDefaultInventoryToAuto(store, projectID, profileID, inv)
}

// SyncDeviceProfilesAutoInventory syncs auto inventories for the given profile IDs (deduplicated).
func SyncDeviceProfilesAutoInventory(store db.Store, projectID int, profileIDs []int) error {
	seen := map[int]bool{}
	for _, id := range profileIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if err := SyncDeviceProfileAutoInventory(store, projectID, id); err != nil {
			return err
		}
	}
	return nil
}

// SyncAllDeviceProfilesAutoInventory refreshes auto inventory for every device type in the project.
func SyncAllDeviceProfilesAutoInventory(store db.Store, projectID int) error {
	profiles, err := store.GetDeviceProfiles(projectID)
	if err != nil {
		return err
	}
	ids := make([]int, 0, len(profiles))
	for _, p := range profiles {
		ids = append(ids, p.ID)
	}
	return SyncDeviceProfilesAutoInventory(store, projectID, ids)
}
