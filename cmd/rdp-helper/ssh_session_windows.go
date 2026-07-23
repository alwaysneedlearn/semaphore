//go:build windows

package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return false
	}
	s := string(out)
	if strings.Contains(s, "No tasks") || strings.Contains(s, "没有运行的任务") {
		return false
	}
	return strings.Contains(s, strconv.Itoa(pid))
}

func killProcess(pid int) {
	if pid <= 0 {
		return
	}
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F", "/T").Run()
}

func startSSHTunnel(args []string) (*exec.Cmd, error) {
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
