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
	cmd := fmt.Sprintf(`"%s" "%%1"`, exePath)

	steps := [][]string{
		{"reg", "add", key, "/ve", "/d", "URL:" + protocolScheme, "/f"},
		{"reg", "add", key, "/v", "URL Protocol", "/d", "", "/f"},
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
