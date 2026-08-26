package policy

import (
	"errors"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func testInputs() (datasets.RecordEnvelope, session.AuthenticatedPeer, PrivacyDecision, TrustEvidence) {
	when := time.Unix(100, 0).UTC()
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "h1", Revision: 1, UpdatedAt: when, OriginDevice: "peer-a", Payload: map[string]any{"query": "weather"}}
	peer := session.AuthenticatedPeer{DeviceID: "peer-a", KeyFingerprint: "fp-a", NegotiatedDatasets: []datasets.Capability{{Dataset: "search.history", SchemaVersion: 1, Read: true, Write: true, Delete: true}}}
	privacy := PrivacyDecision{Purpose: "cross-device-search-history", PurposeAllowed: true, ConsentGranted: true, DecidedAt: when, EvidenceID: "privacy-1"}
	trust := TrustEvidence{DeviceID: "peer-a", KeyFingerprint: "fp-a", Trusted: true, EvaluatedAt: when, EvidenceID: "trust-1"}
	return record, peer, privacy, trust
}

func TestAuthorizeRecordRequiresPrivacyAndTrust(t *testing.T) {
	record, peer, privacy, trust := testInputs()
	evidence, err := AuthorizeRecord(record, peer, privacy, trust, time.Unix(101, 0).UTC())
	if err != nil { t.Fatalf("AuthorizeRecord: %v", err) }
	if evidence.PrivacyEvidenceID != "privacy-1" || evidence.TrustEvidenceID != "trust-1" { t.Fatalf("unexpected evidence: %+v", evidence) }
}

func TestAuthorizeRecordFailsClosed(t *testing.T) {
	record, peer, privacy, trust := testInputs()
	privacy.ConsentGranted = false
	if _, err := AuthorizeRecord(record, peer, privacy, trust, time.Time{}); !errors.Is(err, ErrConsentDenied) { t.Fatalf("error = %v", err) }
	privacy.ConsentGranted = true
	trust.Trusted = false
	if _, err := AuthorizeRecord(record, peer, privacy, trust, time.Time{}); !errors.Is(err, ErrPeerUntrusted) { t.Fatalf("error = %v", err) }
}

func TestAuthorizeRecordRequiresNegotiatedDataset(t *testing.T) {
	record, peer, privacy, trust := testInputs()
	peer.NegotiatedDatasets = nil
	if _, err := AuthorizeRecord(record, peer, privacy, trust, time.Time{}); !errors.Is(err, ErrDatasetUnauthorized) { t.Fatalf("error = %v", err) }
}
