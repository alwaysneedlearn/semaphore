package projects

import (
	"testing"
	"time"
)

func TestRDPLaunchToken_createAndConsume(t *testing.T) {
	resetRDPLaunchTokenStoreForTest()

	token, expiresIn := createRDPLaunchToken(7, 3, 11, 99)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresIn != 90 {
		t.Fatalf("expires_in: got %d want 90", expiresIn)
	}

	entry, ok := consumeRDPLaunchToken(token)
	if !ok {
		t.Fatal("expected consume to succeed")
	}
	if entry.UserID != 7 || entry.ProjectID != 3 || entry.DeviceID != 11 || entry.LogID != 99 {
		t.Fatalf("unexpected entry: %+v", entry)
	}

	if _, ok := consumeRDPLaunchToken(token); ok {
		t.Fatal("token must be one-time; second consume should fail")
	}
}

func TestRDPLaunchToken_missingAndEmpty(t *testing.T) {
	resetRDPLaunchTokenStoreForTest()

	if _, ok := consumeRDPLaunchToken(""); ok {
		t.Fatal("empty token should fail")
	}
	if _, ok := consumeRDPLaunchToken("no-such-token"); ok {
		t.Fatal("unknown token should fail")
	}
}

func TestRDPLaunchToken_expired(t *testing.T) {
	resetRDPLaunchTokenStoreForTest()

	token, _ := createRDPLaunchToken(1, 2, 3, 4)
	v, ok := rdpLaunchTokenStore.Load(token)
	if !ok {
		t.Fatal("token missing after create")
	}
	entry := v.(rdpLaunchTokenEntry)
	entry.ExpiresAt = time.Now().Add(-time.Second)
	rdpLaunchTokenStore.Store(token, entry)

	if _, ok := consumeRDPLaunchToken(token); ok {
		t.Fatal("expired token should fail")
	}
	// Expired consume still deletes the entry.
	if _, ok := rdpLaunchTokenStore.Load(token); ok {
		t.Fatal("expired token should be removed on consume")
	}
}

func TestRDPLifecycleToken_lookupAndDelete(t *testing.T) {
	resetRDPLaunchTokenStoreForTest()

	token, expiresIn := createRDPLifecycleToken(7, 3, 11, 99)
	if token == "" {
		t.Fatal("expected non-empty lifecycle token")
	}
	if expiresIn != int(rdpLifecycleTokenTTL.Seconds()) {
		t.Fatalf("expires_in: got %d want %d", expiresIn, int(rdpLifecycleTokenTTL.Seconds()))
	}

	entry, ok := lookupRDPLifecycleToken(token)
	if !ok {
		t.Fatal("expected lookup to succeed")
	}
	if entry.UserID != 7 || entry.ProjectID != 3 || entry.DeviceID != 11 || entry.LogID != 99 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	// Reusable until deleted / expired.
	if _, ok := lookupRDPLifecycleToken(token); !ok {
		t.Fatal("lifecycle token should be reusable")
	}

	deleteRDPLifecycleToken(token)
	if _, ok := lookupRDPLifecycleToken(token); ok {
		t.Fatal("deleted lifecycle token should fail lookup")
	}
}

func TestRDPLifecycleToken_expired(t *testing.T) {
	resetRDPLaunchTokenStoreForTest()

	token, _ := createRDPLifecycleToken(1, 2, 3, 4)
	v, ok := rdpLifecycleTokenStore.Load(token)
	if !ok {
		t.Fatal("token missing after create")
	}
	entry := v.(rdpLifecycleTokenEntry)
	entry.ExpiresAt = time.Now().Add(-time.Second)
	rdpLifecycleTokenStore.Store(token, entry)

	if _, ok := lookupRDPLifecycleToken(token); ok {
		t.Fatal("expired lifecycle token should fail")
	}
	if _, ok := rdpLifecycleTokenStore.Load(token); ok {
		t.Fatal("expired lifecycle token should be removed on lookup")
	}
}
