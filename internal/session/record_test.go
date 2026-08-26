package session

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
)

func TestVerifySignedRecordBindsAuthenticatedPeer(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	record := datasets.RecordEnvelope{
		Dataset: "search.history", SchemaVersion: 1, RecordID: "history-1", Revision: 1,
		UpdatedAt: time.Unix(500, 0).UTC(), OriginDevice: "device-a",
		Payload: map[string]any{"query": "goreecloud"},
	}
	proof, err := identity.NewRecordProof(record, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	peer := AuthenticatedPeer{DeviceID: "device-a", KeyFingerprint: identity.Fingerprint(publicKey)}
	if err := VerifySignedRecord(peer, record, proof); err != nil {
		t.Fatalf("VerifySignedRecord: %v", err)
	}

	record.RecordID = "tampered"
	if err := VerifySignedRecord(peer, record, proof); err == nil {
		t.Fatal("tampered record unexpectedly verified")
	}
}
