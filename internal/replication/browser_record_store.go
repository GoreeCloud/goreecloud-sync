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

var ErrBrowserDatasetMismatch = errors.New("browser replication store dataset mismatch")

type BrowserRecordStore struct {
	mu      sync.Mutex
	path    string
	dataset string
}

func NewBrowserTabStore(path string) *BrowserRecordStore {
	return &BrowserRecordStore{path: path, dataset: "browser.tabs"}
}

func NewBrowserHistoryStore(path string) *BrowserRecordStore {
	return &BrowserRecordStore{path: path, dataset: "browser.history"}
}

func (s *BrowserRecordStore) AcceptAndPersist(record datasets.RecordEnvelope, peer session.AuthenticatedPeer, privacy policy.PrivacyDecision, trust policy.TrustEvidence, now time.Time) (policy.AcceptanceEvidence, error) {
	if record.Dataset != s.dataset {
		return policy.AcceptanceEvidence{}, ErrBrowserDatasetMismatch
	}
	evidence, err := policy.AuthorizeRecord(record, peer, privacy, trust, now)
	if err != nil {
		return policy.AcceptanceEvidence{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil {
		return policy.AcceptanceEvidence{}, err
	}
	if current, ok := records[record.RecordID]; ok {
		switch datasets.ResolveConflict(current.Envelope, record) {
		case datasets.ResolutionLocal, datasets.ResolutionConverged:
			return current.Evidence, nil
		case datasets.ResolutionRemote:
		}
	}
	records[record.RecordID] = persistedRecord{Envelope: record, Evidence: evidence}
	if err := s.save(records); err != nil {
		return policy.AcceptanceEvidence{}, err
	}
	return evidence, nil
}

func (s *BrowserRecordStore) Records() ([]datasets.RecordEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]datasets.RecordEnvelope, 0, len(records))
	for _, record := range records {
		out = append(out, record.Envelope)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordID < out[j].RecordID })
	return out, nil
}

func (s *BrowserRecordStore) load() (map[string]persistedRecord, error) {
	result := map[string]persistedRecord{}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *BrowserRecordStore) save(records map[string]persistedRecord) error {
	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
