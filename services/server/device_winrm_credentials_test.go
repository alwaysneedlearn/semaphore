package server

import (
	"testing"

	"github.com/semaphoreui/semaphore/db"
)

func TestResolveDeviceWinRMExecCredentialsWinRM(t *testing.T) {
	dev := db.Device{
		IPAddress:   "10.0.0.1",
		AnsibleUser: "admin",
	}
	settings := db.ProjectDeviceSettings{
		DefaultAnsiblePassword: "secret",
		DefaultAnsiblePort:     5985,
	}
	creds, err := ResolveDeviceWinRMExecCredentials(dev, settings, "winrm")
	if err != nil {
		t.Fatal(err)
	}
	if creds.User != "admin" || creds.Password != "secret" || creds.Port != 5985 {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestResolveDeviceWinRMExecCredentialsRDPMissingUser(t *testing.T) {
	dev := db.Device{IPAddress: "10.0.0.2"}
	_, err := ResolveDeviceWinRMExecCredentials(dev, db.ProjectDeviceSettings{}, "rdp")
	if err == nil {
		t.Fatal("expected missing_credentials error")
	}
}

func TestResolveDeviceWinRMExecCredentialsRDP(t *testing.T) {
	dev := db.Device{
		IPAddress:    "10.0.0.3",
		RDPUser:      "DOMAIN\\user",
		RDPPassword:  "pass",
		AnsiblePort:  5986,
	}
	creds, err := ResolveDeviceWinRMExecCredentials(dev, db.ProjectDeviceSettings{}, "rdp")
	if err != nil {
		t.Fatal(err)
	}
	if creds.User != `DOMAIN\user` || creds.Password != "pass" || creds.Port != 5986 {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}
