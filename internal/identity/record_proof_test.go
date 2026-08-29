package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

type recordProofVector struct {
	Protocol                  string `json:"protocol"`
	TestOnlyPrivateKeySeedHex string `json:"testOnlyPrivateKeySeedHex"`
	CanonicalPayloadJSON      string `json:"canonicalPayloadJson"`
	PayloadSHA256             string `json:"payloadSha256"`
	Message                   string `json:"message"`
	Record                    struct {
		Dataset       string         `json:"dataset"`
		SchemaVersion int            `json:"schemaVersion"`
		RecordID      string         `json:"recordId"`
		Revision      uint64         `json:"revision"`
		UpdatedAt     string         `json:"updatedAt"`
		OriginDevice  string         `json:"originDevice"`
		Deleted       bool           `json:"deleted"`
		Payload       map[string]any `json:"payload"`
	} `json:"record"`
	Proof RecordProof `json:"proof"`
}

func TestRecordProofCanonicalVector(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve record proof test source path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "protocol", "testdata", "record-proof-v1.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read vector: %v", err)
	}
	var vector recordProofVector
	if err := json.Unmarshal(fixture, &vector); err != nil {
		t.Fatalf("decode vector: %v", err)
	}
	if vector.Protocol != "GC-SYNC-RECORD/1" {
		t.Fatalf("protocol = %q", vector.Protocol)
	}

	seed, err := hex.DecodeString(vector.TestOnlyPrivateKeySeedHex)
	if err != nil || len(seed) != ed25519.SeedSize {
		t.Fatalf("invalid test-only seed: %v", err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	updatedAt, err := time.Parse(time.RFC3339Nano, vector.Record.UpdatedAt)
	if err != nil {
		t.Fatalf("parse vector timestamp: %v", err)
	}
	record := datasets.RecordEnvelope{
		Dataset: vector.Record.Dataset, SchemaVersion: vector.Record.SchemaVersion,
		RecordID: vector.Record.RecordID, Revision: vector.Record.Revision,
		UpdatedAt: updatedAt, OriginDevice: vector.Record.OriginDevice,
		Deleted: vector.Record.Deleted, Payload: vector.Record.Payload,
	}

	payloadJSON, err := json.Marshal(record.Payload)
	if err != nil {
		t.Fatalf("marshal vector payload: %v", err)
	}
	if string(payloadJSON) != vector.CanonicalPayloadJSON {
		t.Fatalf("canonical payload = %q", payloadJSON)
	}
	digest := sha256.Sum256(payloadJSON)
	if hex.EncodeToString(digest[:]) != vector.PayloadSHA256 {
		t.Fatalf("payload digest = %s", hex.EncodeToString(digest[:]))
	}
	message, err := recordProofMessage(record)
	if err != nil {
		t.Fatalf("recordProofMessage: %v", err)
	}
	if string(message) != vector.Message {
		t.Fatalf("proof message drifted:\n%s", message)
	}
	proof, err := NewRecordProof(record, publicKey, privateKey)
	if err != nil {
		t.Fatalf("NewRecordProof(vector): %v", err)
	}
	if proof != vector.Proof {
		t.Fatalf("proof drifted: got=%+v want=%+v", proof, vector.Proof)
	}
	if _, err := vector.Proof.Verify(record); err != nil {
		t.Fatalf("vector proof no longer verifies: %v", err)
	}
}
