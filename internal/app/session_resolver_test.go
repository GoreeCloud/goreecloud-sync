package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func TestSessionPeerResolverUsesBearerToken(t *testing.T) {
	store := session.NewStore(filepath.Join(t.TempDir(), "sessions.json"))
	peer := session.AuthenticatedPeer{DeviceID: "device-a", KeyFingerprint: "fingerprint-a"}
	if err := store.Put("token-a", peer, time.Unix(200, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	resolver := SessionPeerResolver{Sessions: store, Now: func() time.Time { return time.Unix(100, 0).UTC() }}
	request := httptest.NewRequest("POST", "/api/v1/sync/search/history", nil)
	request.Header.Set("Authorization", "Bearer token-a")
	resolved, err := resolver.ResolvePeer(request.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DeviceID != peer.DeviceID || resolved.KeyFingerprint != peer.KeyFingerprint {
		t.Fatalf("unexpected peer: %+v", resolved)
	}
}

func TestSessionPeerResolverRejectsMissingBearer(t *testing.T) {
	resolver := SessionPeerResolver{Sessions: session.NewStore(filepath.Join(t.TempDir(), "sessions.json"))}
	request := httptest.NewRequest("POST", "/api/v1/sync/search/history", nil)
	if _, err := resolver.ResolvePeer(request.Context(), request); err == nil {
		t.Fatal("missing bearer token was accepted")
	}
}
