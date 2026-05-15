package db

import "testing"

func TestDeviceStatusFromChannelProbes_IgnoresRDPForAggregate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rdp      DeviceStatus
		winrm    DeviceStatus
		api      DeviceStatus
		expected DeviceStatus
	}{
		{"api offline ignores healthy channels", DeviceStatusOnline, DeviceStatusOnline, DeviceStatusOffline, DeviceStatusUnhealthy},
		{"winrm offline ignores rdp online", DeviceStatusOnline, DeviceStatusOffline, DeviceStatusOnline, DeviceStatusUnhealthy},
		{"rdp offline winrm online api online is healthy", DeviceStatusOffline, DeviceStatusOnline, DeviceStatusOnline, DeviceStatusHealthy},
		{"rdp offline winrm online api empty is healthy", DeviceStatusOffline, DeviceStatusOnline, "", DeviceStatusHealthy},
		{"both rdp and winrm offline is unhealthy (winrm)", DeviceStatusOffline, DeviceStatusOffline, DeviceStatusOnline, DeviceStatusUnhealthy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DeviceStatusFromChannelProbes(tc.rdp, tc.winrm, tc.api)
			if got != tc.expected {
				t.Fatalf("DeviceStatusFromChannelProbes(%q,%q,%q) = %q, want %q",
					tc.rdp, tc.winrm, tc.api, got, tc.expected)
			}
		})
	}
}
