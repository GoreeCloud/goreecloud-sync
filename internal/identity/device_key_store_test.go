package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testStoredDeviceID = "11111111-1111-1111-1111-111111111111"

type protectorStub struct{}

func (protectorStub) Seal(_ string, plaintext []byte) ([]byte, error) {
	out := append([]byte("sealed:"), plaintext...)
	return out, nil
}

func (protectorStub) Open(_ string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < len("sealed:") || string(ciphertext[:len("sealed:")]) != "sealed:" {
		return nil, ErrInvalidDeviceKey
	}
	return append([]byte(nil), ciphertext[len("sealed:"):]...), nil
}

func TestDeviceKeyStoreProtectsAndRotatesKeys(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	path := filepath.Join(t.TempDir(), "device-key.json")
	store, err := NewDeviceKeyStore(path, protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	created := time.Unix(100, 0).UTC()
	stored, err := store.Save(testStoredDeviceID, publicKey, privateKey, created)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ProtectedKey == "" || stored.KeyFingerprint != Fingerprint(publicKey) {
		t.Fatalf("unexpected stored key metadata: %+v", stored)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("device key store permissions = %o, want 600", info.Mode().Perm())
	}
	loaded, loadedPrivate, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceID != testStoredDeviceID || !loadedPrivate.Equal(privateKey) {
		t.Fatal("loaded key does not match saved device key")
	}

	newPublic, newPrivate := newDeviceKeyPair(t)
	rotatedAt := time.Unix(200, 0).UTC()
	rotated, err := store.Rotate(newPublic, newPrivate, rotatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !rotated.RotatedAt.Equal(rotatedAt) || rotated.KeyFingerprint != Fingerprint(newPublic) {
		t.Fatalf("rotation metadata mismatch: %+v", rotated)
	}
	_, rotatedPrivate, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !rotatedPrivate.Equal(newPrivate) {
		t.Fatal("loaded key does not match rotated private key")
	}
}

func TestDeviceKeyStoreRejectedRotationDoesNotMutateKey(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	path := filepath.Join(t.TempDir(), "device-key.json")
	store, err := NewDeviceKeyStore(path, protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	created := time.Unix(200, 0).UTC()
	if _, err := store.Save(testStoredDeviceID, publicKey, privateKey, created); err != nil {
		t.Fatal(err)
	}
	newPublic, newPrivate := newDeviceKeyPair(t)
	if _, err := store.Rotate(newPublic, newPrivate, time.Unix(100, 0).UTC()); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("rotate error = %v, want ErrInvalidDeviceKey", err)
	}
	stored, loadedPrivate, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if stored.KeyFingerprint != Fingerprint(publicKey) || !stored.RotatedAt.IsZero() || !loadedPrivate.Equal(privateKey) {
		t.Fatalf("rejected rotation mutated stored identity: %+v", stored)
	}
}

func TestDeviceKeyStoreRejectsMismatchedKeyPair(t *testing.T) {
	publicKey, _ := newDeviceKeyPair(t)
	_, otherPrivate := newDeviceKeyPair(t)
	store, err := NewDeviceKeyStore(filepath.Join(t.TempDir(), "device-key.json"), protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(testStoredDeviceID, publicKey, otherPrivate, time.Unix(100, 0).UTC()); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("save error = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestDeviceKeyStoreRejectsInvalidDeviceID(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	store, err := NewDeviceKeyStore(filepath.Join(t.TempDir(), "device-key.json"), protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("device-a", publicKey, privateKey, time.Unix(100, 0).UTC()); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("save error = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestDeviceKeyStoreRejectsStoredPublicPrivateMismatch(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	path := filepath.Join(t.TempDir(), "device-key.json")
	store, err := NewDeviceKeyStore(path, protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(testStoredDeviceID, publicKey, privateKey, time.Unix(100, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	otherPublic, _ := newDeviceKeyPair(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored StoredDeviceKey
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.PublicKey = encodeDevicePublicKey(otherPublic)
	stored.KeyFingerprint = Fingerprint(otherPublic)
	data, err = json.MarshalIndent(stored, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("load error = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestDeviceKeyStoreRejectsFingerprintTampering(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	path := filepath.Join(t.TempDir(), "device-key.json")
	store, err := NewDeviceKeyStore(path, protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(testStoredDeviceID, publicKey, privateKey, time.Unix(100, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored StoredDeviceKey
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.KeyFingerprint = "sha256:tampered"
	data, err = json.MarshalIndent(stored, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("load error = %v, want ErrInvalidDeviceKey", err)
	}
}

func TestDeviceKeyStoreRejectsLoosePermissions(t *testing.T) {
	publicKey, privateKey := newDeviceKeyPair(t)
	path := filepath.Join(t.TempDir(), "device-key.json")
	store, err := NewDeviceKeyStore(path, protectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(testStoredDeviceID, publicKey, privateKey, time.Unix(100, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(); !errors.Is(err, ErrInvalidDeviceKey) {
		t.Fatalf("load error = %v, want ErrInvalidDeviceKey", err)
	}
}

func newDeviceKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return publicKey, privateKey
}

func encodeDevicePublicKey(publicKey ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(publicKey)
}
