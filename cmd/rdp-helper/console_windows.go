//go:build windows

package main

import "syscall"

var (
	modKernel32          = syscall.NewLazyDLL("kernel32.dll")
	modUser32            = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow = modKernel32.NewProc("GetConsoleWindow")
	procShowWindow       = modUser32.NewProc("ShowWindow")
)

const swHide = 0

// hideConsoleWindow hides the console allocated to this process (protocol launches).
// Interactive shell / CLI commands leave the console visible.
func hideConsoleWindow() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	_, _, _ = procShowWindow.Call(hwnd, uintptr(swHide))
}
