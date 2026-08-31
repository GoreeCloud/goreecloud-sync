package identity

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrKeyProtectorRequired = errors.New("device key protector is required")
	ErrInvalidDeviceKey     = errors.New("invalid device key material")
)

// KeyProtector is implemented by a platform secret facility such as a system
// keychain, TPM-backed service, GoreeCloud Vault adapter, or equivalent. Sync
// never writes plaintext private-key bytes to disk.
type KeyProtector interface {
	Seal(label string, plaintext []byte) ([]byte, error)
	Open(label string, ciphertext []byte) ([]byte, error)
}

type StoredDeviceKey struct {
	DeviceID       string    `json:"deviceId"`
	PublicKey      string    `json:"publicKey"`
	ProtectedKey   string    `json:"protectedPrivateKey"`
	CreatedAt      time.Time `json:"createdAt"`
	RotatedAt      time.Time `json:"rotatedAt,omitempty"`
	KeyFingerprint string    `json:"keyFingerprint"`
}

type DeviceKeyStore struct {
	path      string
	protector KeyProtector
}

func NewDeviceKeyStore(path string, protector KeyProtector) (*DeviceKeyStore, error) {
	if protector == nil {
		return nil, ErrKeyProtectorRequired
	}
	if path == "" {
		return nil, ErrInvalidDeviceKey
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve device key store path: %w", err)
	}
	return &DeviceKeyStore{path: filepath.Clean(absolute), protector: protector}, nil
}

func (s *DeviceKeyStore) Save(deviceID string, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, now time.Time) (StoredDeviceKey, error) {
	if s == nil || s.protector == nil || now.IsZero() || !validDeviceKeyMaterial(deviceID, publicKey, privateKey) {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	sealed, err := s.protector.Seal("goreecloud-sync-device-key:"+deviceID, privateKey)
	if err != nil {
		return StoredDeviceKey{}, err
	}
	if len(sealed) == 0 {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	stored := StoredDeviceKey{
		DeviceID:       deviceID,
		PublicKey:      base64.RawURLEncoding.EncodeToString(publicKey),
		ProtectedKey:   base64.RawURLEncoding.EncodeToString(sealed),
		CreatedAt:      now.UTC(),
		KeyFingerprint: Fingerprint(publicKey),
	}
	if err := s.write(stored); err != nil {
		return StoredDeviceKey{}, err
	}
	return stored, nil
}

func (s *DeviceKeyStore) Load() (StoredDeviceKey, ed25519.PrivateKey, error) {
	if s == nil || s.protector == nil || s.path == "" {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return StoredDeviceKey{}, nil, fmt.Errorf("%w: permissions must not grant group or other access", ErrInvalidDeviceKey)
	}
	var stored StoredDeviceKey
	if err := json.Unmarshal(data, &stored); err != nil {
		return StoredDeviceKey{}, nil, fmt.Errorf("%w: decode stored device key: %v", ErrInvalidDeviceKey, err)
	}
	publicKey, err := validateStoredDeviceKey(stored)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	sealed, err := base64.RawURLEncoding.DecodeString(stored.ProtectedKey)
	if err != nil || len(sealed) == 0 {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	plain, err := s.protector.Open("goreecloud-sync-device-key:"+stored.DeviceID, sealed)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	if len(plain) != ed25519.PrivateKeySize {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	privateKey := ed25519.PrivateKey(plain)
	derivedPublicKey := privateKey.Public().(ed25519.PublicKey)
	if !bytes.Equal(derivedPublicKey, publicKey) {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	return stored, privateKey, nil
}

func (s *DeviceKeyStore) Rotate(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, now time.Time) (StoredDeviceKey, error) {
	if now.IsZero() {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	current, _, err := s.Load()
	if err != nil {
		return StoredDeviceKey{}, err
	}
	if !validDeviceKeyMaterial(current.DeviceID, publicKey, privateKey) {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	stored, err := s.Save(current.DeviceID, publicKey, privateKey, current.CreatedAt)
	if err != nil {
		return StoredDeviceKey{}, err
	}
	stored.RotatedAt = now.UTC()
	if stored.RotatedAt.Before(stored.CreatedAt) {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	if err := s.write(stored); err != nil {
		return StoredDeviceKey{}, err
	}
	return stored, nil
}

func validateStoredDeviceKey(stored StoredDeviceKey) (ed25519.PublicKey, error) {
	if !validDeviceID(stored.DeviceID) || stored.CreatedAt.IsZero() || stored.KeyFingerprint == "" || stored.ProtectedKey == "" {
		return nil, ErrInvalidDeviceKey
	}
	if !stored.RotatedAt.IsZero() && stored.RotatedAt.Before(stored.CreatedAt) {
		return nil, ErrInvalidDeviceKey
	}
	publicKeyBytes, err := base64.RawURLEncoding.DecodeString(stored.PublicKey)
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, ErrInvalidDeviceKey
	}
	publicKey := ed25519.PublicKey(publicKeyBytes)
	if Fingerprint(publicKey) != stored.KeyFingerprint {
		return nil, ErrInvalidDeviceKey
	}
	return publicKey, nil
}

func validDeviceKeyMaterial(deviceID string, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) bool {
	if !validDeviceID(deviceID) || len(publicKey) != ed25519.PublicKeySize || len(privateKey) != ed25519.PrivateKeySize {
		return false
	}
	derivedPublicKey := privateKey.Public().(ed25519.PublicKey)
	return bytes.Equal(derivedPublicKey, publicKey)
}

func (s *DeviceKeyStore) write(stored StoredDeviceKey) error {
	if s == nil || s.path == "" {
		return ErrInvalidDeviceKey
	}
	if _, err := validateStoredDeviceKey(stored); err != nil {
		return err
	}
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(directory, ".device-key-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}
	keep = true
	return nil
}
