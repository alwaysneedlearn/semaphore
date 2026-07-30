//go:build !windows

package main

import "os/exec"

func runHidden(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func combinedOutputHidden(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
