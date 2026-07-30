package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoveFileWithRetry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "launch-test.rdp")
	if err := os.WriteFile(path, []byte("full address:s:127.0.0.1:3389\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	removeFileWithRetry(path, 3, 10*time.Millisecond)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
	// Missing file should be a no-op.
	removeFileWithRetry(path, 2, time.Millisecond)
}
