package server

import (
	"net"
	"strconv"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

// Default ports probed when device-specific ports are unset. RDP listens on
// 3389 (TCP) and WinRM on 5985 (HTTP) by default. API uses db.DefaultDeviceAPIPort.
const (
	defaultRDPPort   = 3389
	defaultWinRMPort = 5985

	probeDialTimeout = 1500 * time.Millisecond
)

// ProbeDevice runs a fast TCP reachability probe against a device's RDP,
// WinRM (Ansible), and application API ports from the Semaphore server.
// Ports: RDP/WinRM use fixed defaults; API uses db.EffectiveDeviceAPIProbePort(device).
// If the device has no IP/hostname, protocol statuses are "unknown".
func ProbeDevice(device db.Device) (rdp, winrm, api db.DeviceStatus, refreshed time.Time) {
	target := device.IPAddress
	if target == "" {
		target = device.Hostname
	}

	now := tz.Now()
	if target == "" {
		return db.DeviceStatusUnknown, db.DeviceStatusUnknown, db.DeviceStatusUnknown, now
	}

	rdp = probePort(target, defaultRDPPort)
	winrmPort := device.AnsiblePort
	if winrmPort <= 0 || winrmPort > 65535 {
		winrmPort = defaultWinRMPort
	}
	winrm = probePort(target, winrmPort)
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
