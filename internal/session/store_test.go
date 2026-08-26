package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStorePersistsHashedSessionToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	store := NewStore(path)
	peer := AuthenticatedPeer{DeviceID: "device-a", KeyFingerprint: "fingerprint-a"}
	expires := time.Unix(200, 0).UTC()
	if err := store.Put("secret-session-token", peer, expires); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-session-token") {
		t.Fatal("raw session token was persisted")
	}
	resolved, err := NewStore(path).Resolve("secret-session-token", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DeviceID != peer.DeviceID || resolved.KeyFingerprint != peer.KeyFingerprint {
		t.Fatalf("unexpected peer: %+v", resolved)
	}
}

func TestStoreRejectsExpiredSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	store := NewStore(path)
	peer := AuthenticatedPeer{DeviceID: "device-a", KeyFingerprint: "fingerprint-a"}
	if err := store.Put("expired-token", peer, time.Unix(100, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	_, err := store.Resolve("expired-token", time.Unix(101, 0).UTC())
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("error = %v, want %v", err, ErrSessionExpired)
	}
}
