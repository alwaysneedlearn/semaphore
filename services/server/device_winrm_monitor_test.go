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
		{"E:\\data", "", true},
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
