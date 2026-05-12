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
