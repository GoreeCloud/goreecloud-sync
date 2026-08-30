package identity

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrustedDeviceStoreAuthorizesConsumedPairingAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity", "trusted-devices.json")
	store, err := NewTrustedDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifiedPairingForTest(t, "11111111-1111-1111-1111-111111111111")
	now := time.Date(2026, 8, 29, 20, 30, 0, 0, time.UTC)
	device, err := store.Authorize("account-a", verified, now)
	if err != nil {
		t.Fatal(err)
	}
	if device.DeviceID != verified.DeviceID() || device.Fingerprint != verified.Fingerprint() {
		t.Fatalf("device=%+v", device)
	}
	trusted, err := store.IsTrusted("account-a", verified.DeviceID(), verified.Fingerprint())
	if err != nil || !trusted {
		t.Fatalf("trusted=%v err=%v", trusted, err)
	}

	reloaded, err := NewTrustedDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	trusted, err = reloaded.IsTrusted("account-a", verified.DeviceID(), verified.Fingerprint())
	if err != nil || !trusted {
		t.Fatalf("reloaded trusted=%v err=%v", trusted, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("permissions=%#o", info.Mode().Perm())
	}
}

func TestTrustedDeviceStoreScopesTrustByAccountAndRevokes(t *testing.T) {
	store, err := NewTrustedDeviceStore(filepath.Join(t.TempDir(), "trusted-devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	verified := verifiedPairingForTest(t, "11111111-1111-1111-1111-111111111111")
	if _, err := store.Authorize("account-a", verified, time.Now()); err != nil {
		t.Fatal(err)
	}
	trusted, err := store.IsTrusted("account-b", verified.DeviceID(), verified.Fingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if trusted {
		t.Fatal("trust must not cross account boundaries")
	}
	if _, err := store.Revoke("account-a", verified.DeviceID(), time.Now()); err != nil {
		t.Fatal(err)
	}
	trusted, err = store.IsTrusted("account-a", verified.DeviceID(), verified.Fingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if trusted {
		t.Fatal("revoked device must not remain trusted")
	}
}

func TestTrustedDeviceStoreRejectsSilentActiveKeyReplacement(t *testing.T) {
	store, err := NewTrustedDeviceStore(filepath.Join(t.TempDir(), "trusted-devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	first := verifiedPairingForTest(t, "11111111-1111-1111-1111-111111111111")
	if _, err := store.Authorize("account-a", first, time.Now()); err != nil {
		t.Fatal(err)
	}
	second := verifiedPairingForTest(t, "11111111-1111-1111-1111-111111111111")
	if second.Fingerprint() == first.Fingerprint() {
		t.Fatal("test expected a different key")
	}
	if _, err := store.Authorize("account-a", second, time.Now()); !errors.Is(err, ErrTrustedDeviceKeyMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func verifiedPairingForTest(t *testing.T, deviceID string) VerifiedPairing {
	t.Helper()
	challenges := NewChallengeStore(time.Minute)
	nonce, err := challenges.Issue()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := NewPairingProof(deviceID, nonce, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := challenges.VerifyAndConsumePairing(proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := challenges.VerifyAndConsumePairing(proof); !errors.Is(err, ErrPairingChallengeUnknown) {
		t.Fatalf("replay err=%v", err)
	}
	return verified
}
