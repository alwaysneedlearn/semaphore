package main

import (
	"strings"
	"testing"
)

func TestBuildRDPFileIncludesUserAndAddress(t *testing.T) {
	body := buildRDPFile("10.33.30.6:3389", 3389, `NEWARE\admin`)
	if !strings.Contains(body, "full address:s:10.33.30.6:3389") {
		t.Fatalf("missing full address: %q", body)
	}
	if !strings.Contains(body, "username:s:NEWARE\\admin") {
		t.Fatalf("missing username: %q", body)
	}
	if strings.Contains(body, "/u:") {
		t.Fatalf("must not embed mstsc /u switch: %q", body)
	}
}

func TestBuildRDPFileWithoutUser(t *testing.T) {
	body := buildRDPFile("127.0.0.1:3390", 3390, "")
	if strings.Contains(body, "username:s:") {
		t.Fatalf("unexpected username line: %q", body)
	}
	if !strings.Contains(body, "server port:i:3390") {
		t.Fatalf("missing port: %q", body)
	}
}
