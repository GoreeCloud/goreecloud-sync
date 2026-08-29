package datasets

import (
	"errors"
	"strings"
	"time"
)

const MaxRecordIDBytes = 512

// RecordEnvelope is the transport-neutral unit replicated by GoreeCloud Sync.
// Payload interpretation remains owned by the first-party application named by
// the negotiated dataset capability.
type RecordEnvelope struct {
	Dataset       string         `json:"dataset"`
	SchemaVersion int            `json:"schemaVersion"`
	RecordID      string         `json:"recordId"`
	Revision      uint64         `json:"revision"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	OriginDevice  string         `json:"originDevice"`
	Deleted       bool           `json:"deleted"`
	Payload       map[string]any `json:"payload,omitempty"`
}

var (
	ErrUnknownDataset      = errors.New("unknown sync dataset")
	ErrSchemaNotNegotiated = errors.New("record schema is not negotiated")
	ErrInvalidRecord       = errors.New("invalid sync record")
)

// ValidateRecord checks a record against an already-negotiated capability.
// Record IDs are bounded because they are persisted, compared for deterministic
// ordering, signed, and used as exclusive retrieval cursors.
// Tombstones intentionally carry no payload so deleted application data is not
// retained merely to communicate deletion.
func ValidateRecord(record RecordEnvelope, capability Capability) error {
	if strings.TrimSpace(record.Dataset) == "" || record.Dataset != capability.Dataset {
		return ErrUnknownDataset
	}
	if record.SchemaVersion < 1 || record.SchemaVersion != capability.SchemaVersion {
		return ErrSchemaNotNegotiated
	}
	if strings.TrimSpace(record.RecordID) == "" || len(record.RecordID) > MaxRecordIDBytes || record.Revision == 0 || record.UpdatedAt.IsZero() || strings.TrimSpace(record.OriginDevice) == "" {
		return ErrInvalidRecord
	}
	if record.Deleted {
		if !capability.Delete || len(record.Payload) != 0 {
			return ErrInvalidRecord
		}
		return nil
	}
	if !capability.Write || record.Payload == nil {
		return ErrInvalidRecord
	}
	return nil
}

// NewTombstone creates the minimal deletion record used to replicate removal
// without retaining the deleted application's payload.
func NewTombstone(capability Capability, recordID, originDevice string, revision uint64, updatedAt time.Time) (RecordEnvelope, error) {
	record := RecordEnvelope{
		Dataset: capability.Dataset, SchemaVersion: capability.SchemaVersion,
		RecordID: recordID, Revision: revision, UpdatedAt: updatedAt,
		OriginDevice: originDevice, Deleted: true,
	}
	if err := ValidateRecord(record, capability); err != nil {
		return RecordEnvelope{}, err
	}
	return record, nil
}
