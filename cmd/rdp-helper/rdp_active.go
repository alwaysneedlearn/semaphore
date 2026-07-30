package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const activeRDPFileName = "rdp-active.json"

// ActiveRDPSession tracks a local mstsc process started by the helper.
type ActiveRDPSession struct {
	DeviceID       int       `json:"device_id"`
	LogID          int       `json:"log_id"`
	PID            int       `json:"pid"`
	APIBase        string    `json:"api_base"`
	LifecycleToken string    `json:"lifecycle_token"`
	Host           string    `json:"host"`
	StartedAt      time.Time `json:"started_at"`
}

type activeRDPStore struct {
	ByDevice map[string]ActiveRDPSession `json:"by_device"`
}

var activeRDPMu sync.Mutex

func activeRDPPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, activeRDPFileName), nil
}

func loadActiveRDPStore() activeRDPStore {
	p, err := activeRDPPath()
	if err != nil {
		return activeRDPStore{ByDevice: map[string]ActiveRDPSession{}}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return activeRDPStore{ByDevice: map[string]ActiveRDPSession{}}
	}
	var s activeRDPStore
	if err := json.Unmarshal(b, &s); err != nil || s.ByDevice == nil {
		return activeRDPStore{ByDevice: map[string]ActiveRDPSession{}}
	}
	return s
}

func saveActiveRDPStore(s activeRDPStore) error {
	if s.ByDevice == nil {
		s.ByDevice = map[string]ActiveRDPSession{}
	}
	p, err := activeRDPPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func putActiveRDPSession(sess ActiveRDPSession) {
	if sess.DeviceID <= 0 || sess.PID <= 0 {
		return
	}
	activeRDPMu.Lock()
	defer activeRDPMu.Unlock()
	store := loadActiveRDPStore()
	key := strconv.Itoa(sess.DeviceID)
	if prev, ok := store.ByDevice[key]; ok && prev.PID > 0 && prev.PID != sess.PID {
		killProcess(prev.PID)
	}
	store.ByDevice[key] = sess
	if err := saveActiveRDPStore(store); err != nil {
		logf("active rdp save failed device=%d err=%v", sess.DeviceID, err)
	}
}

func getActiveRDPSession(deviceID int) (ActiveRDPSession, bool) {
	if deviceID <= 0 {
		return ActiveRDPSession{}, false
	}
	activeRDPMu.Lock()
	defer activeRDPMu.Unlock()
	store := loadActiveRDPStore()
	sess, ok := store.ByDevice[strconv.Itoa(deviceID)]
	if !ok || sess.PID <= 0 {
		return ActiveRDPSession{}, false
	}
	return sess, true
}

func clearActiveRDPSession(deviceID, pid int) {
	if deviceID <= 0 {
		return
	}
	activeRDPMu.Lock()
	defer activeRDPMu.Unlock()
	store := loadActiveRDPStore()
	key := strconv.Itoa(deviceID)
	cur, ok := store.ByDevice[key]
	if !ok {
		return
	}
	if pid > 0 && cur.PID != pid {
		return
	}
	delete(store.ByDevice, key)
	if err := saveActiveRDPStore(store); err != nil {
		logf("active rdp clear failed device=%d err=%v", deviceID, err)
	}
}
