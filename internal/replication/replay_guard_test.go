package replication

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func TestReplayGuardRejectsSameAndOlderRevision(t *testing.T) {
	guard := NewReplayGuard()
	peer := session.AuthenticatedPeer{DeviceID: "device-a"}
	record := datasets.RecordEnvelope{Dataset: "search.history", RecordID: "history-1", Revision: 4}

	if err := guard.Accept(peer, record); err != nil {
		t.Fatalf("first revision rejected: %v", err)
	}
	if err := guard.Accept(peer, record); !errors.Is(err, ErrReplayRecord) {
		t.Fatalf("same revision error = %v, want %v", err, ErrReplayRecord)
	}
	record.Revision = 3
	if err := guard.Accept(peer, record); !errors.Is(err, ErrReplayRecord) {
		t.Fatalf("older revision error = %v, want %v", err, ErrReplayRecord)
	}
	record.Revision = 5
	if err := guard.Accept(peer, record); err != nil {
		t.Fatalf("newer revision rejected: %v", err)
	}
}

func TestReplayGuardScopesRevisionByDevice(t *testing.T) {
	guard := NewReplayGuard()
	record := datasets.RecordEnvelope{Dataset: "bookmarks.items", RecordID: "bookmark-1", Revision: 2}
	if err := guard.Accept(session.AuthenticatedPeer{DeviceID: "device-a"}, record); err != nil {
		t.Fatal(err)
	}
	if err := guard.Accept(session.AuthenticatedPeer{DeviceID: "device-b"}, record); err != nil {
		t.Fatalf("independent device revision rejected: %v", err)
	}
}

func TestPersistentReplayGuardSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay", "state.json")
	peer := session.AuthenticatedPeer{DeviceID: "device-a"}
	record := datasets.RecordEnvelope{Dataset: "search.history", RecordID: "history-2", Revision: 7}

	guard, err := NewPersistentReplayGuard(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Accept(peer, record); err != nil {
		t.Fatal(err)
	}

	restarted, err := NewPersistentReplayGuard(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Accept(peer, record); !errors.Is(err, ErrReplayRecord) {
		t.Fatalf("replayed revision after restart error = %v, want %v", err, ErrReplayRecord)
	}
	record.Revision = 8
	if err := restarted.Accept(peer, record); err != nil {
		t.Fatalf("newer revision after restart rejected: %v", err)
	}
}
