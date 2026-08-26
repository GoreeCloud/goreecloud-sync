package replication

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestCompactTombstonesRequiresConvergenceAndRetention(t *testing.T) {
	store := NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	record, peer, privacy, trust := authorizedInputs()
	record.Deleted = true
	record.Payload = nil
	if _, err := store.AcceptAndPersist(record, peer, privacy, trust, time.Unix(201, 0).UTC()); err != nil {
		t.Fatalf("AcceptAndPersist: %v", err)
	}

	if _, err := store.CompactTombstones(time.Unix(500, 0).UTC(), TombstoneCompactionPolicy{MinimumRetention: time.Second}); !errors.Is(err, ErrCompactionNotConfirmed) {
		t.Fatalf("error = %v, want %v", err, ErrCompactionNotConfirmed)
	}
	removed, err := store.CompactTombstones(time.Unix(500, 0).UTC(), TombstoneCompactionPolicy{MinimumRetention: time.Second, ConvergenceConfirmed: true})
	if err != nil {
		t.Fatalf("CompactTombstones: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	records, err := store.Records()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("compacted tombstone remained: %+v", records)
	}
}
