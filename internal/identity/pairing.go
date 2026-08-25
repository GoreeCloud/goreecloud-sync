package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

var ErrInvalidPairingProof = errors.New("invalid pairing proof")

// PairingProof binds a device identifier and one-time challenge to an Ed25519
// public key. It proves key possession but does not by itself authorize a device.
type PairingProof struct {
	DeviceID  string `json:"device_id"`
	Nonce     string `json:"nonce"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

func NewPairingProof(deviceID, nonce string, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) (PairingProof, error) {
	if !validDeviceID(deviceID) || nonce == "" || len(nonce) > 256 {
		return PairingProof{}, ErrInvalidPairingProof
	}
	if len(publicKey) != ed25519.PublicKeySize || len(privateKey) != ed25519.PrivateKeySize {
		return PairingProof{}, ErrInvalidPairingProof
	}
	message := pairingMessage(deviceID, nonce)
	signature := ed25519.Sign(privateKey, message)
	return PairingProof{
		DeviceID:  deviceID,
		Nonce:     nonce,
		PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		Signature: base64.RawURLEncoding.EncodeToString(signature),
	}, nil
}

// Verify validates shape, key encoding, and signature possession and returns the
// verified public key fingerprint for explicit trust approval by a higher layer.
func (p PairingProof) Verify() (string, error) {
	if !validDeviceID(p.DeviceID) || p.Nonce == "" || len(p.Nonce) > 256 {
		return "", ErrInvalidPairingProof
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(p.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return "", ErrInvalidPairingProof
	}
	signature, err := base64.RawURLEncoding.DecodeString(p.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return "", ErrInvalidPairingProof
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), pairingMessage(p.DeviceID, p.Nonce), signature) {
		return "", ErrInvalidPairingProof
	}
	return Fingerprint(ed25519.PublicKey(publicKey)), nil
}

func pairingMessage(deviceID, nonce string) []byte {
	return []byte(fmt.Sprintf("GC-SYNC-PAIRING/1\n%s\n%s", deviceID, nonce))
}

func validDeviceID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return false
			}
		}
	}
	return true
}
