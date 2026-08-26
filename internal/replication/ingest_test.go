package replication

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/policy"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

type privacyStub struct {
	decision policy.PrivacyDecision
	err      error
}

func (s privacyStub) EvaluatePrivacy(context.Context, datasets.RecordEnvelope, session.AuthenticatedPeer) (policy.PrivacyDecision, error) {
	return s.decision, s.err
}

type trustStub struct {
	decision policy.TrustEvidence
	err      error
}

func (s trustStub) EvaluateTrust(context.Context, datasets.RecordEnvelope, session.AuthenticatedPeer) (policy.TrustEvidence, error) {
	return s.decision, s.err
}

func TestIngestRequiresProvidersAndSignedPeerBinding(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	deviceID := "11111111-1111-1111-1111-111111111111"
	when := time.Unix(300, 0).UTC()
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "h1", Revision: 1, UpdatedAt: when, OriginDevice: deviceID, Payload: map[string]any{"query": "goreecloud"}}
	proof, err := identity.NewRecordProof(record, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	peer := session.AuthenticatedPeer{DeviceID: deviceID, KeyFingerprint: identity.Fingerprint(publicKey), NegotiatedDatasets: []datasets.Capability{{Dataset: "search.history", SchemaVersion: 1, Read: true, Write: true, Delete: true}}}

	ingestor := &Ingestor{History: NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))}
	if _, _, err := ingestor.Ingest(context.Background(), peer, record, proof); !errors.Is(err, policy.ErrDecisionProviderUnavailable) {
		t.Fatalf("error = %v", err)
	}

	ingestor.Providers = policy.DecisionProviders{
		Privacy: privacyStub{decision: policy.PrivacyDecision{Purpose: "cross-device-search-history", PurposeAllowed: true, ConsentGranted: true, DecidedAt: when, EvidenceID: "privacy-1"}},
		Trust: trustStub{decision: policy.TrustEvidence{DeviceID: deviceID, KeyFingerprint: peer.KeyFingerprint, Trusted: true, EvaluatedAt: when, EvidenceID: "trust-1"}},
	}
	ingestor.Now = func() time.Time { return time.Unix(301, 0).UTC() }
	evidence, receipt, err := ingestor.Ingest(context.Background(), peer, record, proof)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if evidence.RecordID != "h1" || receipt.RecordID != "h1" || receipt.ObserverDevice != deviceID {
		t.Fatalf("unexpected outputs: evidence=%+v receipt=%+v", evidence, receipt)
	}
}

func TestIngestRejectsTamperedRecordBeforePolicy(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	deviceID := "22222222-2222-2222-2222-222222222222"
	when := time.Unix(400, 0).UTC()
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "h2", Revision: 1, UpdatedAt: when, OriginDevice: deviceID, Payload: map[string]any{"query": "one"}}
	proof, err := identity.NewRecordProof(record, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	record.Payload = map[string]any{"query": "tampered"}
	peer := session.AuthenticatedPeer{DeviceID: deviceID, KeyFingerprint: identity.Fingerprint(publicKey), NegotiatedDatasets: []datasets.Capability{{Dataset: "search.history", SchemaVersion: 1, Write: true}}}
	ingestor := &Ingestor{
		History: NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json")),
		Providers: policy.DecisionProviders{
			Privacy: privacyStub{err: errors.New("must not reach privacy provider")},
			Trust:   trustStub{err: errors.New("must not reach trust provider")},
		},
	}
	if _, _, err := ingestor.Ingest(context.Background(), peer, record, proof); !errors.Is(err, session.ErrRecordIdentityMismatch) {
		t.Fatalf("error = %v", err)
	}
}
