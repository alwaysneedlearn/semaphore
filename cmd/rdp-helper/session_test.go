package main

import "testing"

func TestMigrateLegacyStateToSessions(t *testing.T) {
	s := State{
		ConnectedEnv: "plant-a",
		SSHPid:       4242,
		SocksPort:    18080,
		LandTarget:   "user@host",
	}
	migrateLegacyState(&s)
	sess, ok := s.Sessions["plant-a"]
	if !ok {
		t.Fatal("missing migrated session")
	}
	if sess.SSHPid != 4242 || sess.SocksPort != 18080 {
		t.Fatalf("bad session: %+v", sess)
	}
	if s.ConnectedEnv != "" || s.SSHPid != 0 {
		t.Fatalf("legacy fields not cleared: %+v", s)
	}
}

func TestPickEnvForLaunchPrefersBaseURL(t *testing.T) {
	c := Config{
		ActiveEnv: "a",
		Environments: []Environment{
			{ID: "a", SemaphoreURL: "http://127.0.0.1:3000"},
			{ID: "b", SemaphoreURL: "http://10.1.1.1:3000"},
		},
	}
	env, ok := pickEnvForLaunch(c, State{}, "http://10.1.1.1:3000")
	if !ok || env.ID != "b" {
		t.Fatalf("got %+v ok=%v", env, ok)
	}
}
