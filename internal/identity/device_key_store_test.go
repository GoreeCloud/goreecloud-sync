package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

type protectorStub struct{}

func (protectorStub) Seal(_ string, plaintext []byte) ([]byte, error) {
	out := append([]byte("sealed:"), plaintext...)
	return out, nil
}

func (protectorStub) Open(_ string, ciphertext []byte) ([]byte, error) {
	return append([]byte(nil), ciphertext[len("sealed:"):]...), nil
}

func TestDeviceKeyStoreProtectsAndRotatesKeys(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewDeviceKeyStore(t.TempDir()+"/device-key.json", protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	created := time.Unix(100, 0).UTC()
	stored, err := store.Save("device-a", publicKey, privateKey, created)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ProtectedKey == "" || stored.KeyFingerprint != Fingerprint(publicKey) {
		t.Fatalf("unexpected stored key metadata: %+v", stored)
	}
	loaded, loadedPrivate, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceID != "device-a" || !loadedPrivate.Equal(privateKey) {
		t.Fatal("loaded key does not match saved device key")
	}

	newPublic, newPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rotatedAt := time.Unix(200, 0).UTC()
	rotated, err := store.Rotate(newPublic, newPrivate, rotatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !rotated.RotatedAt.Equal(rotatedAt) || rotated.KeyFingerprint != Fingerprint(newPublic) {
		t.Fatalf("rotation metadata mismatch: %+v", rotated)
	}
}
