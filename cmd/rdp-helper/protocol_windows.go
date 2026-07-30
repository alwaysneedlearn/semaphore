//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func registerProtocol(exePath string) error {
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	key := `HKCU\Software\Classes\` + protocolScheme
	cmdKey := key + `\shell\open\command`
	iconKey := key + `\DefaultIcon`
	appKey := key + `\Application`
	// Friendly label shown in browser/OS "open this application" prompts.
	displayName := "Semaphore RDP Helper"
	cmd := fmt.Sprintf(`"%s" "%%1"`, exePath)
	icon := fmt.Sprintf(`"%s",0`, exePath)

	steps := [][]string{
		{"reg", "add", key, "/ve", "/d", "URL:" + displayName, "/f"},
		{"reg", "add", key, "/v", "URL Protocol", "/d", "", "/f"},
		{"reg", "add", key, "/v", "FriendlyTypeName", "/d", displayName, "/f"},
		{"reg", "add", appKey, "/v", "ApplicationName", "/d", displayName, "/f"},
		{"reg", "add", appKey, "/v", "ApplicationDescription", "/d", displayName, "/f"},
		{"reg", "add", iconKey, "/ve", "/d", icon, "/f"},
		{"reg", "add", cmdKey, "/ve", "/d", cmd, "/f"},
	}
	for _, args := range steps {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %w (%s)", args, err, string(out))
		}
	}
	return nil
}
