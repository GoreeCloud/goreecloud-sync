package replication

import (
	"errors"
	"time"
)

var (
	ErrCompactionNotConfirmed = errors.New("tombstone convergence is not confirmed")
	ErrInvalidRetention       = errors.New("tombstone retention must be positive")
	ErrReceiptStoreRequired   = errors.New("receipt store is required")
)

type TombstoneCompactionPolicy struct {
	MinimumRetention     time.Duration
	ConvergenceConfirmed bool
}

func (s *SearchHistoryStore) CompactTombstones(now time.Time, policy TombstoneCompactionPolicy) (int, error) {
	if !policy.ConvergenceConfirmed {
		return 0, ErrCompactionNotConfirmed
	}
	if policy.MinimumRetention <= 0 {
		return 0, ErrInvalidRetention
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.UTC().Add(-policy.MinimumRetention)

	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil {
		return 0, err
	}
	removed := 0
	for id, record := range records {
		if record.Envelope.Deleted && !record.Envelope.UpdatedAt.After(cutoff) {
			delete(records, id)
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}
	if err := s.save(records); err != nil {
		return 0, err
	}
	return removed, nil
}

func (s *SearchHistoryStore) CompactTombstonesWithReceipts(now time.Time, minimumRetention time.Duration, requiredPeers []string, receiptStore *ReceiptStore) (int, error) {
	if minimumRetention <= 0 {
		return 0, ErrInvalidRetention
	}
	if receiptStore == nil {
		return 0, ErrReceiptStoreRequired
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.UTC().Add(-minimumRetention)

	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil {
		return 0, err
	}
	removed := 0
	for id, persisted := range records {
		record := persisted.Envelope
		if !record.Deleted || record.UpdatedAt.After(cutoff) {
			continue
		}
		receipts, err := receiptStore.ForRecord(record.Dataset, record.RecordID)
		if err != nil {
			return 0, err
		}
		if !ConvergenceConfirmed(record, requiredPeers, receipts) {
			continue
		}
		delete(records, id)
		removed++
	}
	if removed == 0 {
		return 0, nil
	}
	if err := s.save(records); err != nil {
		return 0, err
	}
	return removed, nil
}

// CompactTombstonesWithReceipts applies the same exact-revision convergence
// requirement to bookmarks.items. Bookmark payload is already absent from the
// tombstone; compaction removes only the deletion marker after every required
// peer has durably observed it and retention has elapsed.
func (s *BookmarkItemStore) CompactTombstonesWithReceipts(now time.Time, minimumRetention time.Duration, requiredPeers []string, receiptStore *ReceiptStore) (int, error) {
	if minimumRetention <= 0 {
		return 0, ErrInvalidRetention
	}
	if receiptStore == nil {
		return 0, ErrReceiptStoreRequired
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.UTC().Add(-minimumRetention)

	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.load()
	if err != nil {
		return 0, err
	}
	removed := 0
	for id, persisted := range records {
		record := persisted.Envelope
		if !record.Deleted || record.UpdatedAt.After(cutoff) {
			continue
		}
		receipts, err := receiptStore.ForRecord(record.Dataset, record.RecordID)
		if err != nil {
			return 0, err
		}
		if !ConvergenceConfirmed(record, requiredPeers, receipts) {
			continue
		}
		delete(records, id)
		removed++
	}
	if removed == 0 {
		return 0, nil
	}
	if err := s.save(records); err != nil {
		return 0, err
	}
	return removed, nil
}
