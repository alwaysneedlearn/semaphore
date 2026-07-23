//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const createNewConsole = 0x00000010 // CREATE_NEW_CONSOLE

// runSSHConnect starts ssh in a new console so password / confirmation prompts work
// when the helper was launched without a TTY (protocol handler, web UI, double-click).
func runSSHConnect(args []string) error {
	cmd := exec.Command("ssh", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewConsole}
	return cmd.Run()
}
