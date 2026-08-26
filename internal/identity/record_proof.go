package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
)

var ErrInvalidRecordProof = errors.New("invalid replication record proof")

// RecordProof binds a synchronized record to an Ed25519 key. Unlike pairing
// proof, this signature covers the replicated record so a captured pairing
// proof cannot be replayed to authorize arbitrary record contents.
type RecordProof struct {
	DeviceID  string `json:"deviceId"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}

func NewRecordProof(record datasets.RecordEnvelope, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) (RecordProof, error) {
	if record.OriginDevice == "" || len(publicKey) != ed25519.PublicKeySize || len(privateKey) != ed25519.PrivateKeySize {
		return RecordProof{}, ErrInvalidRecordProof
	}
	message, err := recordProofMessage(record)
	if err != nil {
		return RecordProof{}, ErrInvalidRecordProof
	}
	return RecordProof{
		DeviceID:  record.OriginDevice,
		PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, message)),
	}, nil
}

// Verify proves that the key signed this exact record and returns the key
// fingerprint for Wardveil trust evaluation.
func (p RecordProof) Verify(record datasets.RecordEnvelope) (string, error) {
	if p.DeviceID == "" || p.DeviceID != record.OriginDevice {
		return "", ErrInvalidRecordProof
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(p.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return "", ErrInvalidRecordProof
	}
	signature, err := base64.RawURLEncoding.DecodeString(p.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return "", ErrInvalidRecordProof
	}
	message, err := recordProofMessage(record)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), message, signature) {
		return "", ErrInvalidRecordProof
	}
	return Fingerprint(ed25519.PublicKey(publicKey)), nil
}

func recordProofMessage(record datasets.RecordEnvelope) ([]byte, error) {
	payload, err := json.Marshal(record.Payload)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(payload)
	return []byte(fmt.Sprintf(
		"GC-SYNC-RECORD/1\n%s\n%d\n%s\n%d\n%s\n%s\n%t\n%s",
		record.Dataset,
		record.SchemaVersion,
		record.RecordID,
		record.Revision,
		record.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		record.OriginDevice,
		record.Deleted,
		hex.EncodeToString(digest[:]),
	)), nil
}
