package replication

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/policy"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrSearchHistoryOnly = errors.New("replication store accepts only search.history")

type persistedRecord struct {
	Envelope datasets.RecordEnvelope   `json:"envelope"`
	Evidence policy.AcceptanceEvidence `json:"acceptanceEvidence"`
}

type SearchHistoryStore struct {
	mu   sync.Mutex
	path string
}

func NewSearchHistoryStore(path string) *SearchHistoryStore {
	return &SearchHistoryStore{path: path}
}

// AcceptAndPersist authorizes before touching disk. Existing records converge
// through the shared deterministic conflict resolver.
func (s *SearchHistoryStore) AcceptAndPersist(record datasets.RecordEnvelope, peer session.AuthenticatedPeer, privacy policy.PrivacyDecision, trust policy.TrustEvidence, now time.Time) (policy.AcceptanceEvidence, error) {
	if record.Dataset != "search.history" {
		return policy.AcceptanceEvidence{}, ErrSearchHistoryOnly
	}
	evidence, err := policy.AuthorizeRecord(record, peer, privacy, trust, now)
	if err != nil {
		return policy.AcceptanceEvidence{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.load()
	if err != nil { return policy.AcceptanceEvidence{}, err }
	if current, ok := records[record.RecordID]; ok {
		winner, _ := datasets.ResolveConflict(current.Envelope, record)
		if winner.RecordID == current.Envelope.RecordID && winner.Revision == current.Envelope.Revision && winner.UpdatedAt.Equal(current.Envelope.UpdatedAt) && winner.OriginDevice == current.Envelope.OriginDevice && winner.Deleted == current.Envelope.Deleted {
			return current.Evidence, nil
		}
	}
	records[record.RecordID] = persistedRecord{Envelope: record, Evidence: evidence}
	if err := s.save(records); err != nil { return policy.AcceptanceEvidence{}, err }
	return evidence, nil
}

func (s *SearchHistoryStore) Records() ([]datasets.RecordEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil { return nil, err }
	out := make([]datasets.RecordEnvelope, 0, len(records))
	for _, record := range records { out = append(out, record.Envelope) }
	sort.Slice(out, func(i, j int) bool { return out[i].RecordID < out[j].RecordID })
	return out, nil
}

func (s *SearchHistoryStore) load() (map[string]persistedRecord, error) {
	result := map[string]persistedRecord{}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) { return result, nil }
	if err != nil { return nil, err }
	if len(data) == 0 { return result, nil }
	if err := json.Unmarshal(data, &result); err != nil { return nil, err }
	return result, nil
}

func (s *SearchHistoryStore) save(records map[string]persistedRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil { return err }
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil { return err }
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { return err }
	return os.Rename(tmp, s.path)
}
