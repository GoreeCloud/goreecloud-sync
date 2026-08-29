package datasets

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateRecordRequiresNegotiatedDatasetAndSchema(t *testing.T) {
	capability := Capability{Dataset: "browser.tabs", SchemaVersion: 1, Write: true, Delete: true}
	record := RecordEnvelope{
		Dataset: "browser.tabs", SchemaVersion: 1, RecordID: "tab-1", Revision: 3,
		UpdatedAt: time.Unix(10, 0).UTC(), OriginDevice: "device-a",
		Payload: map[string]any{"url": "https://example.com"},
	}
	if err := ValidateRecord(record, capability); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	record.SchemaVersion = 2
	if err := ValidateRecord(record, capability); !errors.Is(err, ErrSchemaNotNegotiated) {
		t.Fatalf("schema error = %v, want %v", err, ErrSchemaNotNegotiated)
	}
}

func TestValidateRecordBoundsRecordIDForCursorSafety(t *testing.T) {
	capability := Capability{Dataset: "search.history", SchemaVersion: 1, Write: true, Delete: true}
	record := RecordEnvelope{
		Dataset: "search.history", SchemaVersion: 1,
		RecordID: strings.Repeat("r", MaxRecordIDBytes+1), Revision: 1,
		UpdatedAt: time.Unix(15, 0).UTC(), OriginDevice: "device-a",
		Payload: map[string]any{"query": "goreecloud"},
	}
	if err := ValidateRecord(record, capability); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("record ID bound error = %v, want %v", err, ErrInvalidRecord)
	}
}

func TestTombstoneContainsNoDeletedPayload(t *testing.T) {
	capability := Capability{Dataset: "bookmarks.items", SchemaVersion: 1, Write: true, Delete: true}
	when := time.Unix(20, 0).UTC()
	record, err := NewTombstone(capability, "bookmark-1", "device-b", 4, when)
	if err != nil {
		t.Fatalf("NewTombstone: %v", err)
	}
	if !record.Deleted || record.Payload != nil || record.RecordID != "bookmark-1" {
		t.Fatalf("unexpected tombstone: %+v", record)
	}
}

func TestTombstoneRequiresDeleteCapability(t *testing.T) {
	capability := Capability{Dataset: "search.preferences", SchemaVersion: 1, Write: true, Delete: false}
	_, err := NewTombstone(capability, "prefs", "device-c", 1, time.Unix(30, 0).UTC())
	if !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidRecord)
	}
}
