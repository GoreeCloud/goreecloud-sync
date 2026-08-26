package identity

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
)

func TestRecordProofBindsExactEnvelope(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	record := datasets.RecordEnvelope{
		Dataset: "search.history", SchemaVersion: 1, RecordID: "history-1", Revision: 1,
		UpdatedAt: time.Unix(400, 0).UTC(), OriginDevice: "device-a",
		Payload: map[string]any{"query": "goreecloud"},
	}
	proof, err := NewRecordProof(record, publicKey, privateKey)
	if err != nil {
		t.Fatalf("NewRecordProof: %v", err)
	}
	if _, err := proof.Verify(record); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	tampered := record
	tampered.Payload = map[string]any{"query": "tampered"}
	if _, err := proof.Verify(tampered); err == nil {
		t.Fatal("tampered record unexpectedly verified")
	}
}
