package server

import (
	"net"
	"strconv"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

const probeDialTimeout = 1500 * time.Millisecond

// ProbeDevice runs a fast TCP-port reachability probe against a device's RDP,
// WinRM (Ansible), and application API ports and returns the resulting statuses plus the refresh time.
// Ports come from the device row and project defaults (see db.EffectiveDeviceRDPPort /
// db.EffectiveDeviceAnsiblePort / db.EffectiveDeviceAPIProbePort). If the device has no IP/hostname
// the result is "offline" for all probes.
func ProbeDevice(device db.Device, settings db.ProjectDeviceSettings) (rdp, winrm, api db.DeviceStatus, refreshed time.Time) {
	target := device.IPAddress
	if target == "" {
		target = device.Hostname
	}

	now := tz.Now()
	if target == "" {
		return db.DeviceStatusOffline, db.DeviceStatusOffline, db.DeviceStatusOffline, now
	}

	rdp = probePort(target, db.EffectiveDeviceRDPPort(device))
	winrm = probePort(target, db.EffectiveDeviceAnsiblePort(device, settings))
	api = probePort(target, db.EffectiveDeviceAPIProbePort(device))
	return rdp, winrm, api, now
}

func probePort(host string, port int) db.DeviceStatus {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, probeDialTimeout)
	if err != nil {
		return db.DeviceStatusOffline
	}
	_ = conn.Close()
	return db.DeviceStatusOnline
}
