package server

import "testing"

func TestSanitizeDeviceMonitorPath(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"\\", "", false},
		{"C:", "C:\\", false},
		{"c:\\", "C:\\", false},
		{"D:/Users", "D:\\Users", false},
		{"C:\\Users\\..\\Windows", "", true},
		{"E:\\data", "E:\\data", false},
		{"f:\\logs\\a.txt", "F:\\logs\\a.txt", false},
		{"\\\\server\\share", "", true},
		{"C:\\Program Files\\NEWARE", "C:\\Program Files\\NEWARE", false},
		{"C:\\foo\\.\\bar", "C:\\foo\\bar", false},
	}
	for _, tc := range cases {
		got, err := SanitizeDeviceMonitorPath(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("SanitizeDeviceMonitorPath(%q) expected error, got %q", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("SanitizeDeviceMonitorPath(%q) unexpected err: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("SanitizeDeviceMonitorPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParentDeviceMonitorPath(t *testing.T) {
	if got := ParentDeviceMonitorPath("C:\\Users\\a"); got != "C:\\Users" {
		t.Fatalf("got %q", got)
	}
	if got := ParentDeviceMonitorPath("C:\\Users"); got != "C:\\" {
		t.Fatalf("got %q", got)
	}
	if got := ParentDeviceMonitorPath("C:\\"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestSafeDownloadFilename(t *testing.T) {
	if got := SafeDownloadFilename(`C:\temp\报告.txt`); got != "报告.txt" {
		t.Fatalf("got %q", got)
	}
	if got := SafeDownloadFilename("../../../etc/passwd"); got != "passwd" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractMonitorJSONObject(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"ok":true,"cpu_percent":1}`, `{"ok":true,"cpu_percent":1}`},
		{`[{"ok":true,"cpu_percent":1}]`, `{"ok":true,"cpu_percent":1}`},
		{"noise\n[{\"ok\":true}]\n", `{"ok":true}`},
		{`{"ok":true,"msg":"a}b"}`, `{"ok":true,"msg":"a}b"}`},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractMonitorJSONObject(tc.in)
		if got != tc.want {
			t.Fatalf("extractMonitorJSONObject(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseMonitorJSONWrappedArray(t *testing.T) {
	env, err := parseMonitorJSON(`[{"ok":true,"cpu_percent":12.5,"memory":{"total_bytes":1},"disks":[]}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.CPU != 12.5 {
		t.Fatalf("got %+v", env)
	}
}
