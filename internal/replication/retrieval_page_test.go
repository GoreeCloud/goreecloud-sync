package replication

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
)

func TestValidatedPersistedPageKeepsOrderedWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	writeRetrievalPageFixture(t, path, map[string]persistedRecord{
		"query-4": {Envelope: retrievalPageRecord("query-4", 1)},
		"query-1": {Envelope: retrievalPageRecord("query-1", 1)},
		"query-3": {Envelope: retrievalPageRecord("query-3", 1)},
		"query-2": {Envelope: retrievalPageRecord("query-2", 1)},
	})

	page, nextAfter, err := validatedPersistedPage(path, retrievalPageCapability(), "query-1", 2)
	if err != nil {
		t.Fatalf("validatedPersistedPage: %v", err)
	}
	if len(page) != 2 || page[0].RecordID != "query-2" || page[1].RecordID != "query-3" {
		t.Fatalf("page = %#v, want query-2/query-3", page)
	}
	if nextAfter != "query-3" {
		t.Fatalf("nextAfter = %q, want query-3", nextAfter)
	}
}

func TestValidatedPersistedPageStillValidatesRecordsOutsidePage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	invalid := retrievalPageRecord("query-z", 2)
	writeRetrievalPageFixture(t, path, map[string]persistedRecord{
		"query-1": {Envelope: retrievalPageRecord("query-1", 1)},
		"query-z": {Envelope: invalid},
	})

	if _, _, err := validatedPersistedPage(path, retrievalPageCapability(), "", 1); !errors.Is(err, datasets.ErrSchemaNotNegotiated) {
		t.Fatalf("error = %v, want %v", err, datasets.ErrSchemaNotNegotiated)
	}
}

func TestValidatedPersistedPageRejectsKeyRecordIDMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	writeRetrievalPageFixture(t, path, map[string]persistedRecord{
		"stored-key": {Envelope: retrievalPageRecord("different-record-id", 1)},
	})

	if _, _, err := validatedPersistedPage(path, retrievalPageCapability(), "", 1); !errors.Is(err, ErrPersistedRecordKeyMismatch) {
		t.Fatalf("error = %v, want %v", err, ErrPersistedRecordKeyMismatch)
	}
}

func retrievalPageCapability() datasets.Capability {
	return datasets.Capability{
		Dataset: "search.history", Application: "search", SchemaVersion: 1,
		Read: true, Write: true, Delete: true,
	}
}

func retrievalPageRecord(recordID string, schemaVersion int) datasets.RecordEnvelope {
	return datasets.RecordEnvelope{
		Dataset: "search.history", SchemaVersion: schemaVersion, RecordID: recordID, Revision: 1,
		UpdatedAt: time.Date(2026, 8, 28, 20, 0, 0, 0, time.UTC), OriginDevice: "device-1",
		Payload: map[string]any{"query": recordID},
	}
}

func writeRetrievalPageFixture(t *testing.T, path string, records map[string]persistedRecord) {
	t.Helper()
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
