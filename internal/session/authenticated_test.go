package session

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
)

func TestAuthenticatePeerVerifiesProofAndNegotiatesDatasets(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	deviceID := "123e4567-e89b-12d3-a456-426614174000"
	proof, err := identity.NewPairingProof(deviceID, "nonce-1", publicKey, privateKey)
	if err != nil {
		t.Fatalf("NewPairingProof: %v", err)
	}
	local := []datasets.Capability{{Dataset: "browser.tabs", Application: "browser", SchemaVersion: 1, Read: true, Write: true}}
	remote := []datasets.Capability{{Dataset: "browser.tabs", Application: "browser", SchemaVersion: 1, Read: true, Write: true}}
	peer, err := AuthenticatePeer(deviceID, proof, local, remote)
	if err != nil {
		t.Fatalf("AuthenticatePeer: %v", err)
	}
	if peer.KeyFingerprint == "" || len(peer.NegotiatedDatasets) != 1 {
		t.Fatalf("unexpected authenticated peer: %+v", peer)
	}
}

func TestAuthenticatePeerRejectsDeviceMismatch(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	proof, err := identity.NewPairingProof("123e4567-e89b-12d3-a456-426614174000", "nonce-2", publicKey, privateKey)
	if err != nil {
		t.Fatalf("NewPairingProof: %v", err)
	}
	if _, err := AuthenticatePeer("123e4567-e89b-12d3-a456-426614174001", proof, nil, nil); err != ErrDeviceMismatch {
		t.Fatalf("error = %v, want %v", err, ErrDeviceMismatch)
	}
}
