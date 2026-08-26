package replication

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/policy"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func authorizedInputs() (datasets.RecordEnvelope, session.AuthenticatedPeer, policy.PrivacyDecision, policy.TrustEvidence) {
	when := time.Unix(200, 0).UTC()
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "history-1", Revision: 1, UpdatedAt: when, OriginDevice: "device-a", Payload: map[string]any{"query": "goreecloud"}}
	peer := session.AuthenticatedPeer{DeviceID: "device-a", KeyFingerprint: "fingerprint-a", NegotiatedDatasets: []datasets.Capability{{Dataset: "search.history", SchemaVersion: 1, Read: true, Write: true, Delete: true}}}
	privacy := policy.PrivacyDecision{Purpose: "cross-device-search-history", PurposeAllowed: true, ConsentGranted: true, DecidedAt: when, EvidenceID: "privacy-evidence"}
	trust := policy.TrustEvidence{DeviceID: "device-a", KeyFingerprint: "fingerprint-a", Trusted: true, EvaluatedAt: when, EvidenceID: "trust-evidence"}
	return record, peer, privacy, trust
}

func TestAcceptAndPersistWritesOnlyAuthorizedHistory(t *testing.T) {
	store := NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	record, peer, privacy, trust := authorizedInputs()
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(201, 0).UTC()); err != nil {
		t.Fatalf("AcceptAndPersist: %v", err)
	}
	records, err := store.Records()
	if err != nil { t.Fatalf("Records: %v", err) }
	if len(records) != 1 || records[0].RecordID != "history-1" { t.Fatalf("unexpected records: %+v", records) }
}

func TestRejectedRecordNeverPersists(t *testing.T) {
	store := NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	record, peer, privacy, trust := authorizedInputs()
	privacy.ConsentGranted = false
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Time{}); !errors.Is(err, policy.ErrConsentDenied) {
		t.Fatalf("error = %v", err)
	}
	records, err := store.Records()
	if err != nil { t.Fatalf("Records: %v", err) }
	if len(records) != 0 { t.Fatalf("rejected record persisted: %+v", records) }
}

func TestHigherRevisionWinsPersistenceConflict(t *testing.T) {
	store := NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	record, peer, privacy, trust := authorizedInputs()
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(201, 0).UTC()); err != nil { t.Fatal(err) }
	record.Revision = 2
	record.Payload = map[string]any{"query": "goreecloud sync"}
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(202, 0).UTC()); err != nil { t.Fatal(err) }
	records, err := store.Records()
	if err != nil { t.Fatal(err) }
	if len(records) != 1 || records[0].Revision != 2 { t.Fatalf("unexpected conflict result: %+v", records) }
}
