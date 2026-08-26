package replication

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
)

func TestBrowserStoresEnforceDatasetBoundary(t *testing.T) {
	tabs := NewBrowserTabStore(filepath.Join(t.TempDir(), "tabs.json"))
	record, peer, privacy, trust := authorizedInputs()
	record.Dataset = "browser.history"
	if _, err := tabs.AcceptAndPersist(record, peer, privacy, trust, time.Unix(300, 0).UTC()); !errors.Is(err, ErrBrowserDatasetMismatch) {
		t.Fatalf("error = %v, want %v", err, ErrBrowserDatasetMismatch)
	}
}

func TestBrowserTabStorePersistsAuthorizedRecord(t *testing.T) {
	store := NewBrowserTabStore(filepath.Join(t.TempDir(), "tabs.json"))
	record, peer, privacy, trust := authorizedInputs()
	record.Dataset = "browser.tabs"
	record.RecordID = "tab-1"
	record.Payload = map[string]any{"url": "https://goreecloud.com", "title": "GoreeCloud"}
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(300, 0).UTC()); err != nil {
		t.Fatalf("AcceptAndPersist: %v", err)
	}
	records, err := store.Records()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Dataset != "browser.tabs" || records[0].RecordID != "tab-1" {
		t.Fatalf("unexpected records: %+v", records)
	}
}

func TestBrowserHistoryStorePersistsAuthorizedRecord(t *testing.T) {
	store := NewBrowserHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	record, peer, privacy, trust := authorizedInputs()
	record.Dataset = "browser.history"
	record.RecordID = "history-1"
	record.Payload = map[string]any{"url": "https://goreecloud.com", "visitedAt": "2026-08-26T23:00:00Z"}
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(300, 0).UTC()); err != nil {
		t.Fatalf("AcceptAndPersist: %v", err)
	}
	records, err := store.Records()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Dataset != "browser.history" || records[0].RecordID != "history-1" {
		t.Fatalf("unexpected records: %+v", records)
	}
}

func TestBrowserStoresUseDeterministicConflictResolution(t *testing.T) {
	store := NewBrowserTabStore(filepath.Join(t.TempDir(), "tabs.json"))
	record, peer, privacy, trust := authorizedInputs()
	record.Dataset = "browser.tabs"
	record.RecordID = "tab-1"
	record.Revision = 2
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(300, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	older := record
	older.Revision = 1
	older.Payload = map[string]any{"url": "https://older.example"}
	if _, err := store.AcceptAndPersist(older, peer, privacy, trust, time.Unix(301, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	records, err := store.Records()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Revision != 2 {
		t.Fatalf("older revision replaced current record: %+v", records)
	}
}

var _ = datasets.RecordEnvelope{}
