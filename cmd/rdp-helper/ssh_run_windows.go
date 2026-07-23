//go:build windows

package main

import (
	"os"
	"os/exec"
)

// runSSHConnect runs ssh in the helper's existing console (stdin/stdout/stderr).
// Do not use CREATE_NEW_CONSOLE — that window flashes and often exits with 0xffffffff.
func runSSHConnect(args []string) error {
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
