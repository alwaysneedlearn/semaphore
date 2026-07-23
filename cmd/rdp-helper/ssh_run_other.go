//go:build !windows

package main

import (
	"os"
	"os/exec"
)

// runSSHConnect attaches to the current terminal so interactive SSH auth works.
func runSSHConnect(args []string) error {
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
