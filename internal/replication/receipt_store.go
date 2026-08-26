package replication

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ReceiptStore durably records payload-free peer observation evidence.
type ReceiptStore struct {
	mu   sync.Mutex
	path string
}

func NewReceiptStore(path string) *ReceiptStore { return &ReceiptStore{path: path} }

func (s *ReceiptStore) Append(receipt ObservationReceipt) error {
	if receipt.Dataset == "" || receipt.RecordID == "" || receipt.Revision == 0 || receipt.RecordDigest == "" || receipt.ObserverDevice == "" || receipt.ObservedAt.IsZero() {
		return ErrInvalidObservationReceipt
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	receipts, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range receipts {
		if sameReceipt(existing, receipt) {
			return nil
		}
	}
	receipts = append(receipts, receipt)
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].ObservedAt.Equal(receipts[j].ObservedAt) {
			if receipts[i].RecordID == receipts[j].RecordID {
				return receipts[i].ObserverDevice < receipts[j].ObserverDevice
			}
			return receipts[i].RecordID < receipts[j].RecordID
		}
		return receipts[i].ObservedAt.Before(receipts[j].ObservedAt)
	})
	return s.save(receipts)
}

func (s *ReceiptStore) ForRecord(dataset, recordID string) ([]ObservationReceipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipts, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]ObservationReceipt, 0)
	for _, receipt := range receipts {
		if receipt.Dataset == dataset && receipt.RecordID == recordID {
			out = append(out, receipt)
		}
	}
	return out, nil
}

func (s *ReceiptStore) load() ([]ObservationReceipt, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var receipts []ObservationReceipt
	if err := json.Unmarshal(data, &receipts); err != nil {
		return nil, err
	}
	return receipts, nil
}

func (s *ReceiptStore) save(receipts []ObservationReceipt) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func sameReceipt(a, b ObservationReceipt) bool {
	return a.Dataset == b.Dataset && a.RecordID == b.RecordID && a.Revision == b.Revision && a.RecordDigest == b.RecordDigest && a.ObserverDevice == b.ObserverDevice
}
