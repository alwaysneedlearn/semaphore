package server

import (
	"fmt"
	"strings"

	"github.com/semaphoreui/semaphore/db"
)

// DeviceWinRMExecCredentials holds resolved connection parameters for a one-shot WinRM exec.
type DeviceWinRMExecCredentials struct {
	Mode               string
	User               string
	Password           string
	Host               string
	Port               int
	Transport          string
	Scheme             string
	CertValidation     string
	UseHTTPS           bool
	InsecureSkipVerify bool
}

// ResolveDeviceWinRMExecCredentials resolves user/password and WinRM connection params
// aligned with BuildDeviceInventoryLine / Ansible inventory generation.
func ResolveDeviceWinRMExecCredentials(
	device db.Device,
	settings db.ProjectDeviceSettings,
	mode string,
) (DeviceWinRMExecCredentials, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = db.DeviceWinRMCredentialModeWinRM
	}
	if mode != db.DeviceWinRMCredentialModeWinRM && mode != db.DeviceWinRMCredentialModeRDP {
		return DeviceWinRMExecCredentials{}, &db.ValidationError{Message: "credential_mode must be winrm or rdp"}
	}

	host := strings.TrimSpace(device.IPAddress)
	if host == "" {
		return DeviceWinRMExecCredentials{}, &db.ValidationError{Message: "device ip_address is required"}
	}

	creds := DeviceWinRMExecCredentials{
		Mode:   mode,
		Host:   host,
		Port:   db.EffectiveDeviceAnsiblePort(device, settings),
		Scheme: device.AnsibleWinRMScheme,
	}
	if creds.Scheme == "" {
		creds.Scheme = settings.DefaultAnsibleWinRMScheme
	}
	if creds.Scheme == "" {
		creds.Scheme = "http"
	}
	creds.Transport = device.AnsibleWinRMTransport
	if creds.Transport == "" {
		creds.Transport = settings.DefaultAnsibleWinRMTransport
	}
	if creds.Transport == "" {
		creds.Transport = "basic"
	}
	creds.CertValidation = device.AnsibleWinRMServerCertValidation
	if creds.CertValidation == "" {
		creds.CertValidation = settings.DefaultAnsibleWinRMServerCertValidation
	}
	if creds.CertValidation == "" {
		creds.CertValidation = "ignore"
	}
	creds.UseHTTPS = strings.EqualFold(strings.TrimSpace(creds.Scheme), "https")
	creds.InsecureSkipVerify = strings.EqualFold(strings.TrimSpace(creds.CertValidation), "ignore")

	switch mode {
	case db.DeviceWinRMCredentialModeWinRM:
		creds.User = device.AnsibleUser
		if creds.User == "" {
			creds.User = settings.DefaultAnsibleUser
		}
		creds.Password = device.AnsiblePassword
		if creds.Password == "" {
			creds.Password = settings.DefaultAnsiblePassword
		}
	case db.DeviceWinRMCredentialModeRDP:
		creds.User = strings.TrimSpace(device.RDPUser)
		creds.Password = device.RDPPassword
	}

	if strings.TrimSpace(creds.User) == "" {
		return DeviceWinRMExecCredentials{}, &db.ValidationError{Message: "missing_credentials"}
	}

	return creds, nil
}

// PreviewDeviceWinRMConnection returns non-secret connection summary for the UI.
func PreviewDeviceWinRMConnection(creds DeviceWinRMExecCredentials) map[string]any {
	return map[string]any{
		"credential_mode": creds.Mode,
		"resolved_user":   creds.User,
		"resolved_host":   creds.Host,
		"resolved_port":   creds.Port,
		"transport":       creds.Transport,
		"scheme":          creds.Scheme,
		"endpoint":        fmt.Sprintf("winrm://%s@%s:%d", creds.User, creds.Host, creds.Port),
	}
}
