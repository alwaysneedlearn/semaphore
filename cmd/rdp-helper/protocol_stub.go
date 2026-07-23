//go:build !windows

package main

import "fmt"

func registerProtocol(exePath string) error {
	return fmt.Errorf("protocol registration is only supported on Windows (exe=%s)", exePath)
}
