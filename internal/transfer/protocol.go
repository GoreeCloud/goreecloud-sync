package transfer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const PayloadProtocolVersion = 1

type PayloadKind string

const (
	PayloadKindFile PayloadKind = "file"
	PayloadKindText PayloadKind = "text"
)

type ReceiptStatus string

const ReceiptVerified ReceiptStatus = "verified"

// PayloadOffer is the bounded control-plane description for one Development-stage
// one-to-one payload transfer. The embedded manifest remains the integrity
// authority for the bytes that follow; this structure is not a Stable wire
// compatibility promise.
type PayloadOffer struct {
	Version    int         `json:"version"`
	TransferID string      `json:"transfer_id"`
	Kind       PayloadKind `json:"kind"`
	Manifest   Manifest    `json:"manifest"`
}

// PayloadDecision is sent by the receiver only after local application logic has
// explicitly approved or rejected the offered operation. Rejections intentionally
// carry no application-policy detail across the peer boundary.
type PayloadDecision struct {
	Version    int    `json:"version"`
	TransferID string `json:"transfer_id"`
	Accepted   bool   `json:"accepted"`
}

// PayloadReceipt is emitted only after the receiver has accepted every declared
// chunk and verified the complete payload digest.
type PayloadReceipt struct {
	Version    int           `json:"version"`
	TransferID string        `json:"transfer_id"`
	Status     ReceiptStatus `json:"status"`
	Size       int64         `json:"size"`
	Hash       string        `json:"hash"`
}

func NewTransferID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate transfer ID: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func NewPayloadOffer(kind PayloadKind, manifest Manifest) (PayloadOffer, error) {
	transferID, err := NewTransferID()
	if err != nil {
		return PayloadOffer{}, err
	}
	offer := PayloadOffer{
		Version:    PayloadProtocolVersion,
		TransferID: transferID,
		Kind:       kind,
		Manifest:   manifest,
	}
	if err := offer.Validate(); err != nil {
		return PayloadOffer{}, err
	}
	return offer, nil
}

func (o PayloadOffer) Validate() error {
	if o.Version != PayloadProtocolVersion {
		return fmt.Errorf("unsupported payload protocol version %d", o.Version)
	}
	if err := validateTransferID(o.TransferID); err != nil {
		return err
	}
	switch o.Kind {
	case PayloadKindFile, PayloadKindText:
	default:
		return fmt.Errorf("unsupported payload kind %q", o.Kind)
	}
	if err := o.Manifest.Validate(); err != nil {
		return fmt.Errorf("validate payload manifest: %w", err)
	}
	return nil
}

func (d PayloadDecision) ValidateFor(offer PayloadOffer) error {
	if d.Version != PayloadProtocolVersion {
		return fmt.Errorf("unsupported payload decision version %d", d.Version)
	}
	if err := validateTransferID(d.TransferID); err != nil {
		return err
	}
	if d.TransferID != offer.TransferID {
		return fmt.Errorf("payload decision transfer ID does not match offer")
	}
	return nil
}

func (r PayloadReceipt) ValidateFor(offer PayloadOffer) error {
	if r.Version != PayloadProtocolVersion {
		return fmt.Errorf("unsupported payload receipt version %d", r.Version)
	}
	if err := validateTransferID(r.TransferID); err != nil {
		return err
	}
	if r.TransferID != offer.TransferID {
		return fmt.Errorf("payload receipt transfer ID does not match offer")
	}
	if r.Status != ReceiptVerified {
		return fmt.Errorf("payload receipt status %q is not verified", r.Status)
	}
	if r.Size != offer.Manifest.Size {
		return fmt.Errorf("payload receipt size %d does not match manifest size %d", r.Size, offer.Manifest.Size)
	}
	if r.Hash != offer.Manifest.Hash {
		return fmt.Errorf("payload receipt hash does not match manifest hash")
	}
	return nil
}

func VerifiedPayloadReceipt(offer PayloadOffer) PayloadReceipt {
	return PayloadReceipt{
		Version:    PayloadProtocolVersion,
		TransferID: offer.TransferID,
		Status:     ReceiptVerified,
		Size:       offer.Manifest.Size,
		Hash:       offer.Manifest.Hash,
	}
}

func validateTransferID(transferID string) error {
	if len(transferID) != 32 {
		return fmt.Errorf("transfer ID must contain 32 lowercase hexadecimal characters")
	}
	if transferID != string([]byte(transferID)) {
		return fmt.Errorf("transfer ID is invalid")
	}
	if transferID != lowerASCII(transferID) {
		return fmt.Errorf("transfer ID must use lowercase hexadecimal")
	}
	decoded, err := hex.DecodeString(transferID)
	if err != nil || len(decoded) != 16 {
		return fmt.Errorf("transfer ID is not valid hexadecimal")
	}
	return nil
}

func lowerASCII(value string) string {
	out := []byte(value)
	for index, b := range out {
		if b >= 'A' && b <= 'Z' {
			out[index] = b + ('a' - 'A')
		}
	}
	return string(out)
}
