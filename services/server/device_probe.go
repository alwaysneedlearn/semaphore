package server

import (
	"net"
	"strconv"
	"time"

	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/tz"
)

// Default ports probed by ProbeDevice. RDP listens on 3389 (TCP) and WinRM on
// 5985 (HTTP) by default. Connectivity to the port is treated as "online".
const (
	defaultRDPPort   = 3389
	defaultWinRMPort = 5985

	probeDialTimeout = 1500 * time.Millisecond
)

// ProbeDevice runs a fast TCP-port reachability probe against a device's RDP
// and WinRM ports and returns the resulting statuses plus the refresh time.
// If the device has no IP/hostname the result is "unknown".
func ProbeDevice(device db.Device) (rdp, winrm db.DeviceStatus, refreshed time.Time) {
	target := device.IPAddress
	if target == "" {
		target = device.Hostname
	}

	now := tz.Now()
	if target == "" {
		return db.DeviceStatusUnknown, db.DeviceStatusUnknown, now
	}

	rdp = probePort(target, defaultRDPPort)
	winrm = probePort(target, defaultWinRMPort)
	return rdp, winrm, now
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
