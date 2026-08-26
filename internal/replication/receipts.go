package replication

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrInvalidObservationReceipt = errors.New("invalid peer observation receipt")

// ObservationReceipt records that an authenticated peer observed a specific
// record revision. It contains no application payload.
type ObservationReceipt struct {
	Dataset        string    `json:"dataset"`
	RecordID       string    `json:"recordId"`
	Revision       uint64    `json:"revision"`
	RecordDigest   string    `json:"recordDigest"`
	ObserverDevice string    `json:"observerDevice"`
	ObservedAt     time.Time `json:"observedAt"`
}

func NewObservationReceipt(record datasets.RecordEnvelope, observer session.AuthenticatedPeer, observedAt time.Time) (ObservationReceipt, error) {
	if strings.TrimSpace(observer.DeviceID) == "" || record.RecordID == "" || record.Revision == 0 {
		return ObservationReceipt{}, ErrInvalidObservationReceipt
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	return ObservationReceipt{
		Dataset: record.Dataset, RecordID: record.RecordID, Revision: record.Revision,
		RecordDigest: recordDigest(record), ObserverDevice: observer.DeviceID,
		ObservedAt: observedAt.UTC(),
	}, nil
}

func recordDigest(record datasets.RecordEnvelope) string {
	value := strings.Join([]string{
		record.Dataset,
		record.RecordID,
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		record.OriginDevice,
	}, "\n")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ConvergenceConfirmed returns true only when every required peer has an exact
// receipt for the tombstone revision and digest.
func ConvergenceConfirmed(record datasets.RecordEnvelope, requiredPeers []string, receipts []ObservationReceipt) bool {
	if !record.Deleted || len(requiredPeers) == 0 {
		return false
	}
	required := append([]string(nil), requiredPeers...)
	sort.Strings(required)
	seen := map[string]bool{}
	digest := recordDigest(record)
	for _, receipt := range receipts {
		if receipt.Dataset == record.Dataset && receipt.RecordID == record.RecordID && receipt.Revision == record.Revision && receipt.RecordDigest == digest {
			seen[receipt.ObserverDevice] = true
		}
	}
	for _, peer := range required {
		if !seen[peer] {
			return false
		}
	}
	return true
}
