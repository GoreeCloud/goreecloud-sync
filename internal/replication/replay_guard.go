package replication

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrReplayRecord = errors.New("replication record revision already observed")

type replayKey struct {
	DeviceID string `json:"deviceId"`
	Dataset  string `json:"dataset"`
	RecordID string `json:"recordId"`
}

type replayEntry struct {
	Key      replayKey `json:"key"`
	Revision uint64    `json:"revision"`
}

// ReplayGuard rejects an already-observed or older revision from the same
// authenticated device and can persist its high-water marks across restarts.
type ReplayGuard struct {
	mu       sync.Mutex
	path     string
	revision map[replayKey]uint64
}

func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{revision: make(map[replayKey]uint64)}
}

func NewPersistentReplayGuard(path string) (*ReplayGuard, error) {
	guard := &ReplayGuard{path: path, revision: make(map[replayKey]uint64)}
	if err := guard.load(); err != nil {
		return nil, err
	}
	return guard, nil
}

func (g *ReplayGuard) Accept(peer session.AuthenticatedPeer, record datasets.RecordEnvelope) error {
	if g == nil {
		return nil
	}
	key := replayKey{DeviceID: peer.DeviceID, Dataset: record.Dataset, RecordID: record.RecordID}
	g.mu.Lock()
	defer g.mu.Unlock()
	if current, ok := g.revision[key]; ok && record.Revision <= current {
		return ErrReplayRecord
	}
	g.revision[key] = record.Revision
	if g.path != "" {
		if err := g.save(); err != nil {
			delete(g.revision, key)
			return err
		}
	}
	return nil
}

func (g *ReplayGuard) load() error {
	data, err := os.ReadFile(g.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var entries []replayEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	for _, entry := range entries {
		g.revision[entry.Key] = entry.Revision
	}
	return nil
}

func (g *ReplayGuard) save() error {
	entries := make([]replayEntry, 0, len(g.revision))
	for key, revision := range g.revision {
		entries = append(entries, replayEntry{Key: key, Revision: revision})
	}
	if err := os.MkdirAll(filepath.Dir(g.path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	tmp := g.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, g.path)
}
