package replication

import (
	"errors"
	"sync"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrReplayRecord = errors.New("replication record revision already observed")

type replayKey struct {
	deviceID string
	dataset  string
	recordID string
}

// ReplayGuard rejects an already-observed or older revision from the same
// authenticated device. Conflict resolution still handles independent updates
// from different devices after they pass this transport-level replay check.
type ReplayGuard struct {
	mu       sync.Mutex
	revision map[replayKey]uint64
}

func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{revision: make(map[replayKey]uint64)}
}

func (g *ReplayGuard) Accept(peer session.AuthenticatedPeer, record datasets.RecordEnvelope) error {
	if g == nil {
		return nil
	}
	key := replayKey{deviceID: peer.DeviceID, dataset: record.Dataset, recordID: record.RecordID}
	g.mu.Lock()
	defer g.mu.Unlock()
	if current, ok := g.revision[key]; ok && record.Revision <= current {
		return ErrReplayRecord
	}
	g.revision[key] = record.Revision
	return nil
}
