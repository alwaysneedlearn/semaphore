package server

import "testing"

func TestNeedsCmdExe(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{`netstat -ano | findstr ":3389"`, true},
		{`netstat -ano | findstr LISTENING`, true},
		{`Get-Process`, false},
		{`netstat -ano | Select-String ':3389'`, false},
	}
	for _, tc := range cases {
		if got := needsCmdExe(tc.cmd); got != tc.want {
			t.Fatalf("needsCmdExe(%q) = %v, want %v", tc.cmd, got, tc.want)
		}
	}
}

func TestPrepareWinRMCommand(t *testing.T) {
	cmd, shell := prepareWinRMCommand(`netstat -ano | findstr ":3389"`, "powershell")
	if shell != "powershell" {
		t.Fatalf("shell = %q", shell)
	}
	if cmd != `cmd /c 'netstat -ano | findstr ":3389"'` {
		t.Fatalf("cmd = %q", cmd)
	}

	cmd, shell = prepareWinRMCommand("Get-Process", "powershell")
	if cmd != "Get-Process" || shell != "powershell" {
		t.Fatalf("got %q / %q", cmd, shell)
	}

	cmd, shell = prepareWinRMCommand("dir", "cmd")
	if cmd != "dir" || shell != "cmd" {
		t.Fatalf("got %q / %q", cmd, shell)
	}
}
