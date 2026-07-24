package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppDirIsExeDirectory(t *testing.T) {
	dir, err := appDir()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Dir(exe)
	if abs, err := filepath.Abs(want); err == nil {
		want = abs
	}
	got, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("appDir=%q want exe dir %q", got, want)
	}
}
