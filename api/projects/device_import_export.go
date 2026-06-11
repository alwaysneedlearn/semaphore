package projects

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/semaphoreui/semaphore/api/helpers"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/services/server"
)

const deviceBulkImportMaxRows = 5000

var deviceBulkExportCSVHeaders = []string{
	"ip_address",
	"hostname",
	"profile_key",
	"rdp_user",
	"rdp_password",
	"rdp_port",
	"ansible_user",
	"ansible_password",
	"ansible_connection",
	"ansible_winrm_transport",
	"ansible_winrm_scheme",
	"ansible_port",
	"api_port",
	"ansible_winrm_server_cert_validation",
}

func deviceToBulkExportRow(dev db.Device, profileKey string) db.DeviceBulkExportRow {
	return db.DeviceBulkExportRow{
		IPAddress:                        dev.IPAddress,
		Hostname:                         dev.Hostname,
		ProfileKey:                       profileKey,
		DeviceProfileID:                  dev.DeviceProfileID,
		RDPUser:                          dev.RDPUser,
		RDPPassword:                      dev.RDPPassword,
		RDPPort:                          dev.RDPPort,
		AnsibleUser:                      dev.AnsibleUser,
		AnsiblePassword:                  dev.AnsiblePassword,
		AnsibleConnection:                dev.AnsibleConnection,
		AnsibleWinRMTransport:            dev.AnsibleWinRMTransport,
		AnsibleWinRMScheme:               dev.AnsibleWinRMScheme,
		AnsiblePort:                      dev.AnsiblePort,
		APIPort:                          dev.APIPort,
		AnsibleWinRMServerCertValidation: dev.AnsibleWinRMServerCertValidation,
	}
}

func parseDeviceExportIDs(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid device id: %q", part)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func filterDevicesByIDs(devices []db.Device, ids []int) []db.Device {
	if len(ids) == 0 {
		return devices
	}
	want := map[int]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := make([]db.Device, 0, len(ids))
	for _, dev := range devices {
		if want[dev.ID] {
			out = append(out, dev)
		}
	}
	return out
}

func buildDeviceProfileMaps(profiles []db.DeviceProfile) (idToKey map[int]string, keyToID map[string]int) {
	idToKey = map[int]string{}
	keyToID = map[string]int{}
	for _, p := range profiles {
		idToKey[p.ID] = p.ProfileKey
		keyToID[strings.ToUpper(strings.TrimSpace(p.ProfileKey))] = p.ID
	}
	return idToKey, keyToID
}

func resolveImportProfileID(row db.DeviceBulkExportRow, keyToID map[string]int, defaultProfileID int) (int, error) {
	if row.DeviceProfileID > 0 {
		return row.DeviceProfileID, nil
	}
	key := strings.ToUpper(strings.TrimSpace(row.ProfileKey))
	if key != "" {
		if id, ok := keyToID[key]; ok {
			return id, nil
		}
		return 0, fmt.Errorf("unknown profile_key %q", row.ProfileKey)
	}
	if defaultProfileID > 0 {
		return defaultProfileID, nil
	}
	return 0, fmt.Errorf("profile_key or device_profile_id is required")
}

func bulkExportRowToDevice(row db.DeviceBulkExportRow, profileID int, settings db.ProjectDeviceSettings) db.Device {
	dev := db.Device{
		IPAddress:                        strings.TrimSpace(row.IPAddress),
		Hostname:                         strings.TrimSpace(row.Hostname),
		DeviceProfileID:                  profileID,
		RDPUser:                          strings.TrimSpace(row.RDPUser),
		RDPPassword:                      strings.TrimSpace(row.RDPPassword),
		RDPPort:                          row.RDPPort,
		AnsibleUser:                      strings.TrimSpace(row.AnsibleUser),
		AnsiblePassword:                  strings.TrimSpace(row.AnsiblePassword),
		AnsibleConnection:                strings.TrimSpace(row.AnsibleConnection),
		AnsibleWinRMTransport:            strings.TrimSpace(row.AnsibleWinRMTransport),
		AnsibleWinRMScheme:               strings.TrimSpace(row.AnsibleWinRMScheme),
		AnsiblePort:                      row.AnsiblePort,
		APIPort:                          row.APIPort,
		AnsibleWinRMServerCertValidation: strings.TrimSpace(row.AnsibleWinRMServerCertValidation),
	}
	dev.Name = dev.Hostname
	normalizeDeviceConnection(&dev, settings)
	return dev
}

func writeDeviceBulkExportCSV(w http.ResponseWriter, rows []db.DeviceBulkExportRow) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=devices_export_%d.csv", time.Now().Unix()))
	cw := csv.NewWriter(w)
	if err := cw.Write(deviceBulkExportCSVHeaders); err != nil {
		return err
	}
	for _, row := range rows {
		record := []string{
			row.IPAddress,
			row.Hostname,
			row.ProfileKey,
			row.RDPUser,
			row.RDPPassword,
			strconv.Itoa(row.RDPPort),
			row.AnsibleUser,
			row.AnsiblePassword,
			row.AnsibleConnection,
			row.AnsibleWinRMTransport,
			row.AnsibleWinRMScheme,
			strconv.Itoa(row.AnsiblePort),
			strconv.Itoa(row.APIPort),
			row.AnsibleWinRMServerCertValidation,
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ExportDevices returns device edit-form fields as JSON or CSV.
// Query: format=json|csv (default json), ids=1,2,3 (optional; omit for all devices).
func ExportDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	store := helpers.Store(r)

	ids, err := parseDeviceExportIDs(r.URL.Query().Get("ids"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	devices, err := store.GetDevices(project.ID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	devices = filterDevicesByIDs(devices, ids)

	profiles, err := store.GetDeviceProfiles(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	idToKey, _ := buildDeviceProfileMaps(profiles)

	rows := make([]db.DeviceBulkExportRow, 0, len(devices))
	for _, dev := range devices {
		rows = append(rows, deviceToBulkExportRow(dev, idToKey[dev.DeviceProfileID]))
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "csv" {
		if err := writeDeviceBulkExportCSV(w, rows); err != nil {
			helpers.WriteError(w, err)
		}
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"version":   1,
		"exported":  time.Now().UTC().Format(time.RFC3339),
		"count":     len(rows),
		"devices":   rows,
	})
}

type deviceBulkImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportDevices upserts devices from JSON export rows (match by ip_address per project).
func ImportDevices(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	settings, err := helpers.Store(r).GetProjectDeviceSettings(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	var body struct {
		Devices           []db.DeviceBulkExportRow `json:"devices"`
		DefaultProfileKey string                   `json:"default_profile_key"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	if len(body.Devices) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "devices array is required",
		})
		return
	}
	if len(body.Devices) > deviceBulkImportMaxRows {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("too many rows (max %d)", deviceBulkImportMaxRows),
		})
		return
	}

	store := helpers.Store(r)
	profiles, err := store.GetDeviceProfiles(project.ID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	_, keyToID := buildDeviceProfileMaps(profiles)

	defaultProfileID := 0
	if k := strings.TrimSpace(body.DefaultProfileKey); k != "" {
		if id, ok := keyToID[strings.ToUpper(k)]; ok {
			defaultProfileID = id
		}
	}
	if defaultProfileID <= 0 {
		def, dErr := server.EnsureDefaultDeviceProfile(store, project.ID)
		if dErr != nil {
			helpers.WriteError(w, dErr)
			return
		}
		defaultProfileID = def.ID
	}

	importErrors := make([]deviceBulkImportError, 0)
	byIP := map[string]db.Device{}

	allDevices, err := store.GetDevices(project.ID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	existingByIP := map[string]bool{}
	for _, d := range allDevices {
		if ip := strings.TrimSpace(d.IPAddress); ip != "" {
			existingByIP[ip] = true
		}
	}

	for i, row := range body.Devices {
		rowNum := i + 1
		ip := strings.TrimSpace(row.IPAddress)
		if ip == "" {
			importErrors = append(importErrors, deviceBulkImportError{
				Row:     rowNum,
				Message: "ip_address is required",
			})
			continue
		}
		profileID, pErr := resolveImportProfileID(row, keyToID, defaultProfileID)
		if pErr != nil {
			importErrors = append(importErrors, deviceBulkImportError{Row: rowNum, Message: pErr.Error()})
			continue
		}
		if _, err := store.GetDeviceProfile(project.ID, profileID); err != nil {
			importErrors = append(importErrors, deviceBulkImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("invalid device_profile_id %d", profileID),
			})
			continue
		}

		dev := bulkExportRowToDevice(row, profileID, settings)
		if err := dev.Validate(); err != nil {
			importErrors = append(importErrors, deviceBulkImportError{Row: rowNum, Message: err.Error()})
			continue
		}

		byIP[ip] = dev
	}

	toUpsert := make([]db.Device, 0, len(byIP))
	for _, dev := range byIP {
		toUpsert = append(toUpsert, dev)
	}

	if len(toUpsert) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "no valid rows to import",
			"errors": importErrors,
		})
		return
	}

	saved, err := store.UpsertDevicesFromBulkImport(project.ID, toUpsert)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	profileIDs := make([]int, 0, len(saved))
	for _, d := range saved {
		profileIDs = append(profileIDs, d.DeviceProfileID)
	}
	if err = syncDeviceProfilesAutoInventory(r, project.ID, profileIDs...); err != nil {
		helpers.WriteError(w, err)
		return
	}

	created := 0
	updated := 0
	for _, d := range saved {
		ip := strings.TrimSpace(d.IPAddress)
		if existingByIP[ip] {
			updated++
		} else {
			created++
		}
	}

	helpers.EventLog(r, helpers.EventLogUpdate, helpers.EventLogItem{
		UserID:      helpers.UserFromContext(r).ID,
		ProjectID:   project.ID,
		ObjectType:  db.EventDevice,
		Description: fmt.Sprintf("Imported %d device(s) (%d created, %d updated)", len(saved), created, updated),
	})

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"saved_count": len(saved),
		"created":     created,
		"updated":     updated,
		"errors":      importErrors,
	})
}
