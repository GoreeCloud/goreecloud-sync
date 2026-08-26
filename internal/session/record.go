package session

import (
	"errors"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
)

var ErrRecordIdentityMismatch = errors.New("record signature does not match authenticated peer")

// VerifySignedRecord requires the record-bound signature to resolve to the same
// device identity and key fingerprint established by the authenticated session.
func VerifySignedRecord(peer AuthenticatedPeer, record datasets.RecordEnvelope, proof identity.RecordProof) error {
	if record.OriginDevice != peer.DeviceID || proof.DeviceID != peer.DeviceID {
		return ErrRecordIdentityMismatch
	}
	fingerprint, err := proof.Verify(record)
	if err != nil || fingerprint != peer.KeyFingerprint {
		return ErrRecordIdentityMismatch
	}
	return nil
}
