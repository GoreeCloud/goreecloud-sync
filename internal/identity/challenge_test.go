package identity

import (
	"errors"
	"testing"
	"time"
)

func TestChallengeStoreConsumesSuccessfulProofOnce(t *testing.T) {
	store := NewChallengeStore(time.Minute)
	nonce, err := store.Issue()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := NewPairingProof("11111111-1111-1111-1111-111111111111", nonce, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := store.VerifyAndConsume(proof)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != Fingerprint(publicKey) {
		t.Fatalf("fingerprint=%q", fingerprint)
	}
	if _, err := store.VerifyAndConsume(proof); !errors.Is(err, ErrPairingChallengeUnknown) {
		t.Fatalf("replay err=%v", err)
	}
}

func TestChallengeStoreRejectsExpiredProof(t *testing.T) {
	store := NewChallengeStore(time.Minute)
	now := time.Date(2026, 8, 29, 20, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	nonce, err := store.Issue()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, _ := NewKeyPair()
	proof, _ := NewPairingProof("11111111-1111-1111-1111-111111111111", nonce, publicKey, privateKey)
	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := store.VerifyAndConsume(proof); !errors.Is(err, ErrPairingChallengeExpired) {
		t.Fatalf("err=%v", err)
	}
}
