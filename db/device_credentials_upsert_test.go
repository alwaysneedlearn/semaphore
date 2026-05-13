package db

import "testing"

func TestMergeDeviceCredentialsOnUpsertPreservesWhenIncomingEmpty(t *testing.T) {
	existing := Device{
		AnsibleUser:     "u1",
		AnsiblePassword: "p1",
		RDPUser:         "r1",
		RDPPassword:     "rp1",
	}
	incoming := Device{}
	MergeDeviceCredentialsOnUpsert(&existing, incoming)
	if existing.AnsibleUser != "u1" || existing.AnsiblePassword != "p1" ||
		existing.RDPUser != "r1" || existing.RDPPassword != "rp1" {
		t.Fatalf("expected credentials preserved, got %+v", existing)
	}
}

func TestMergeDeviceCredentialsOnUpsertOverwritesWhenIncomingNonEmpty(t *testing.T) {
	existing := Device{
		AnsibleUser:     "oldu",
		AnsiblePassword: "oldp",
		RDPUser:         "oldr",
		RDPPassword:     "oldrp",
	}
	incoming := Device{
		AnsibleUser:     "newu",
		AnsiblePassword: "newp",
		RDPUser:         "newr",
		RDPPassword:     "newrp",
	}
	MergeDeviceCredentialsOnUpsert(&existing, incoming)
	if existing.AnsibleUser != "newu" || existing.AnsiblePassword != "newp" ||
		existing.RDPUser != "newr" || existing.RDPPassword != "newrp" {
		t.Fatalf("expected incoming credentials applied, got %+v", existing)
	}
}

func TestMergeDevicePortsOnUpsertPreservesWhenIncomingZero(t *testing.T) {
	existing := Device{RDPPort: 3390, AnsiblePort: 5986}
	incoming := Device{}
	MergeDevicePortsOnUpsert(&existing, incoming)
	if existing.RDPPort != 3390 || existing.AnsiblePort != 5986 {
		t.Fatalf("expected ports preserved, got %+v", existing)
	}
}

func TestMergeDevicePortsOnUpsertOverwritesWhenIncomingValid(t *testing.T) {
	existing := Device{RDPPort: 3389, AnsiblePort: 5985}
	incoming := Device{RDPPort: 13389, AnsiblePort: 15985}
	MergeDevicePortsOnUpsert(&existing, incoming)
	if existing.RDPPort != 13389 || existing.AnsiblePort != 15985 {
		t.Fatalf("expected incoming ports applied, got %+v", existing)
	}
}

func TestEffectiveDeviceRDPPort(t *testing.T) {
	if got := EffectiveDeviceRDPPort(Device{}); got != DefaultDeviceRDPPort {
		t.Fatalf("expected default %d, got %d", DefaultDeviceRDPPort, got)
	}
	if got := EffectiveDeviceRDPPort(Device{RDPPort: 4000}); got != 4000 {
		t.Fatalf("expected 4000, got %d", got)
	}
}

func TestEffectiveDeviceAnsiblePort(t *testing.T) {
	if got := EffectiveDeviceAnsiblePort(Device{}, ProjectDeviceSettings{}); got != DefaultDeviceAnsiblePort {
		t.Fatalf("expected default %d, got %d", DefaultDeviceAnsiblePort, got)
	}
	if got := EffectiveDeviceAnsiblePort(Device{}, ProjectDeviceSettings{DefaultAnsiblePort: 15985}); got != 15985 {
		t.Fatalf("expected project default 15985, got %d", got)
	}
	if got := EffectiveDeviceAnsiblePort(Device{AnsiblePort: 5986}, ProjectDeviceSettings{DefaultAnsiblePort: 15985}); got != 5986 {
		t.Fatalf("expected device port 5986, got %d", got)
	}
}
