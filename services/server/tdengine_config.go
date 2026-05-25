package server

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tdengine"
	"github.com/semaphoreui/semaphore/util"
)

const tdengineOptionKey = "tdengine.config"

var tdengineConfigMu sync.RWMutex

// EffectiveTDengineConfig merges file config, env, and DB option (DB wins).
func EffectiveTDengineConfig(store db.Store) tdengine.Config {
	cfg := tdengine.Config{}
	if util.Config != nil && util.Config.TDengine != nil {
		t := util.Config.TDengine
		cfg = tdengine.Config{
			Enabled:  t.Enabled,
			URL:      t.URL,
			User:     t.User,
			Password: t.Password,
			Database: t.Database,
		}
	}
	if v := strings.TrimSpace(os.Getenv("SEMAPHORE_TDENGINE_URL")); v != "" {
		cfg.URL = v
		cfg.Enabled = true
	}
	if v := strings.TrimSpace(os.Getenv("SEMAPHORE_TDENGINE_USER")); v != "" {
		cfg.User = v
	}
	if v := strings.TrimSpace(os.Getenv("SEMAPHORE_TDENGINE_PASSWORD")); v != "" {
		cfg.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("SEMAPHORE_TDENGINE_DATABASE")); v != "" {
		cfg.Database = v
	}
	if store != nil {
		raw, err := store.GetOption(tdengineOptionKey)
		if err == nil && strings.TrimSpace(raw) != "" {
			if dbCfg, err := tdengine.ParseConfigJSON(raw); err == nil {
				if dbCfg.Enabled {
					cfg.Enabled = true
				}
				if strings.TrimSpace(dbCfg.URL) != "" {
					cfg.URL = dbCfg.URL
				}
				if strings.TrimSpace(dbCfg.User) != "" {
					cfg.User = dbCfg.User
				}
				if dbCfg.Password != "" {
					cfg.Password = dbCfg.Password
				}
				if strings.TrimSpace(dbCfg.Database) != "" {
					cfg.Database = dbCfg.Database
				}
			}
		}
	}
	tdengineConfigMu.RLock()
	defer tdengineConfigMu.RUnlock()
	return cfg
}

// SaveTDengineConfig persists admin UI settings to options.
func SaveTDengineConfig(store db.Store, cfg tdengine.Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	tdengineConfigMu.Lock()
	defer tdengineConfigMu.Unlock()
	return store.SetOption(tdengineOptionKey, string(b))
}

// MapDeviceStatusForTDengine maps DB device_status to online/offline.
func MapDeviceStatusForTDengine(status db.DeviceStatus) string {
	if status == db.DeviceStatusHealthy {
		return "online"
	}
	return "offline"
}
