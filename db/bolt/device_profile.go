package bolt

import (
	"time"

	"github.com/semaphoreui/semaphore/db"
)

func (d *BoltDb) GetDeviceProfiles(projectID int) ([]db.DeviceProfile, error) {
	var profiles []db.DeviceProfile
	err := d.getObjects(projectID, db.DeviceProfileProps, db.RetrieveQueryParams{}, nil, &profiles)
	return profiles, err
}

func (d *BoltDb) GetDeviceProfile(projectID, profileID int) (db.DeviceProfile, error) {
	var p db.DeviceProfile
	err := d.getObject(projectID, db.DeviceProfileProps, intObjectID(profileID), &p)
	return p, err
}

func (d *BoltDb) GetDeviceProfileByKey(projectID int, key string) (db.DeviceProfile, error) {
	profiles, err := d.GetDeviceProfiles(projectID)
	if err != nil {
		return db.DeviceProfile{}, err
	}
	for _, p := range profiles {
		if p.ProfileKey == key {
			return p, nil
		}
	}
	return db.DeviceProfile{}, db.ErrNotFound
}

func (d *BoltDb) DeleteDeviceProfile(projectID, profileID int) error {
	devices, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, &db.DeviceListFilter{DeviceProfileID: profileID})
	if err != nil {
		return err
	}
	if len(devices) > 0 {
		return &db.ValidationError{Message: "Cannot delete device profile while devices are assigned to it"}
	}
	return d.deleteObject(projectID, db.DeviceProfileProps, intObjectID(profileID), nil)
}

func (d *BoltDb) CreateDeviceProfile(p db.DeviceProfile) (db.DeviceProfile, error) {
	res, err := d.createObject(p.ProjectID, db.DeviceProfileProps, p)
	if err != nil {
		return p, err
	}
	return res.(db.DeviceProfile), nil
}

func (d *BoltDb) GetProjectDeviceProfileSettings(projectID, profileID int) (db.ProjectDeviceProfileSettings, error) {
	return db.ProjectDeviceProfileSettings{ProjectID: projectID, ProfileID: profileID}, nil
}

func (d *BoltDb) UpdateProjectDeviceProfileSettings(s db.ProjectDeviceProfileSettings) error {
	return nil
}

func (d *BoltDb) AssignDevicesWithoutProfile(projectID, profileID int) error {
	devices, err := d.GetDevices(projectID, db.RetrieveQueryParams{}, nil)
	if err != nil {
		return err
	}
	for _, dev := range devices {
		if dev.DeviceProfileID <= 0 {
			dev.DeviceProfileID = profileID
			if err := d.UpdateDevice(dev); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *BoltDb) GetDeviceProfileSettingsDueForRefresh(_ time.Time) ([]db.ProjectDeviceProfileSettings, error) {
	return nil, nil
}

func (d *BoltDb) MarkDeviceProfileStatusRefreshed(projectID, profileID int, refreshed time.Time) error {
	return nil
}
