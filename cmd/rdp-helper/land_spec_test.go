package main

import "testing"

func TestLandSpecUsesLastHopPortWhenLandHostEmpty(t *testing.T) {
	env := Environment{
		ID:       "jt",
		LandPort: 22, // stale UI default must not win
		Hops: []Hop{
			{Host: "10.20.1.134", Port: 2222, User: "root"},
		},
	}
	user, host, port, err := landSpec(env)
	if err != nil {
		t.Fatal(err)
	}
	if user != "root" || host != "10.20.1.134" || port != 2222 {
		t.Fatalf("got user=%q host=%q port=%d; want root@10.20.1.134:2222", user, host, port)
	}
	if pj := proxyJumpArg(env); pj != "" {
		t.Fatalf("single hop should have empty ProxyJump, got %q", pj)
	}
}

func TestLandSpecExplicitLandPort(t *testing.T) {
	env := Environment{
		ID:       "jt",
		LandHost: "10.33.35.154",
		LandUser: "calb",
		LandPort: 2200,
		Hops: []Hop{
			{Host: "10.20.1.134", Port: 2222, User: "root"},
		},
	}
	user, host, port, err := landSpec(env)
	if err != nil {
		t.Fatal(err)
	}
	if user != "calb" || host != "10.33.35.154" || port != 2200 {
		t.Fatalf("got %s@%s:%d", user, host, port)
	}
	pj := proxyJumpArg(env)
	if pj != "root@10.20.1.134:2222" {
		t.Fatalf("ProxyJump=%q", pj)
	}
}

func TestProxyJumpTwoHopsLastIsLand(t *testing.T) {
	env := Environment{
		Hops: []Hop{
			{Host: "hop1", Port: 2222, User: "u1"},
			{Host: "hop2", Port: 2201, User: "u2"},
		},
	}
	_, _, port, err := landSpec(env)
	if err != nil {
		t.Fatal(err)
	}
	if port != 2201 {
		t.Fatalf("land port=%d want 2201", port)
	}
	if pj := proxyJumpArg(env); pj != "u1@hop1:2222" {
		t.Fatalf("ProxyJump=%q", pj)
	}
}
