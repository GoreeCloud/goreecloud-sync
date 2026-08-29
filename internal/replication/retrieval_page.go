package replication

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
)

var ErrPersistedRecordKeyMismatch = errors.New("persisted sync record key does not match recordId")

// validatedPersistedPage scans the complete persisted JSON object so corruption
// outside the requested page still fails closed. It retains only the smallest
// limit+1 records after the exclusive cursor, bounding retrieval-page memory by
// the negotiated page size rather than the full dataset size.
func validatedPersistedPage(path string, capability datasets.Capability, after string, limit int) ([]datasets.RecordEnvelope, string, error) {
	if limit < 1 {
		return nil, "", ErrInvalidRetrievalPageSize
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []datasets.RecordEnvelope{}, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	opening, err := decoder.Token()
	if errors.Is(err, io.EOF) {
		return []datasets.RecordEnvelope{}, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if delimiter, ok := opening.(json.Delim); !ok || delimiter != '{' {
		return nil, "", fmt.Errorf("persisted sync store must be a JSON object")
	}

	candidateLimit := limit + 1
	candidates := make([]datasets.RecordEnvelope, 0, candidateLimit)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, "", err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, "", fmt.Errorf("persisted sync store contains a non-string key")
		}

		var record persistedRecord
		if err := decoder.Decode(&record); err != nil {
			return nil, "", err
		}
		if key != record.Envelope.RecordID {
			return nil, "", ErrPersistedRecordKeyMismatch
		}
		if err := datasets.ValidateRecord(record.Envelope, capability); err != nil {
			return nil, "", err
		}
		if record.Envelope.RecordID <= after {
			continue
		}
		candidates = retainPageCandidate(candidates, record.Envelope, candidateLimit)
	}

	closing, err := decoder.Token()
	if err != nil {
		return nil, "", err
	}
	if delimiter, ok := closing.(json.Delim); !ok || delimiter != '}' {
		return nil, "", fmt.Errorf("persisted sync store has an invalid object terminator")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, "", fmt.Errorf("persisted sync store contains trailing JSON")
		}
		return nil, "", err
	}

	if len(candidates) <= limit {
		return candidates, "", nil
	}
	page := candidates[:limit]
	return page, page[len(page)-1].RecordID, nil
}

var ErrInvalidRetrievalPageSize = errors.New("invalid sync retrieval page size")

func retainPageCandidate(candidates []datasets.RecordEnvelope, candidate datasets.RecordEnvelope, limit int) []datasets.RecordEnvelope {
	position := sort.Search(len(candidates), func(index int) bool {
		return candidates[index].RecordID >= candidate.RecordID
	})
	if position < len(candidates) && candidates[position].RecordID == candidate.RecordID {
		candidates[position] = candidate
		return candidates
	}
	if len(candidates) >= limit && position == len(candidates) {
		return candidates
	}
	candidates = append(candidates, datasets.RecordEnvelope{})
	copy(candidates[position+1:], candidates[position:])
	candidates[position] = candidate
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}
