package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	return &DeviceKeyStore{path: path, protector: protector}, nil
}

func (s *DeviceKeyStore) Save(deviceID string, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, now time.Time) (StoredDeviceKey, error) {
	if deviceID == "" || len(publicKey) != ed25519.PublicKeySize || len(privateKey) != ed25519.PrivateKeySize {
		return StoredDeviceKey{}, ErrInvalidDeviceKey
	}
	sealed, err := s.protector.Seal("goreecloud-sync-device-key:"+deviceID, privateKey)
	if err != nil {
		return StoredDeviceKey{}, err
	}
	stored := StoredDeviceKey{
		DeviceID: deviceID, PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
		ProtectedKey: base64.RawURLEncoding.EncodeToString(sealed), CreatedAt: now.UTC(),
		KeyFingerprint: Fingerprint(publicKey),
	}
	if err := s.write(stored); err != nil {
		return StoredDeviceKey{}, err
	}
	return stored, nil
}

func (s *DeviceKeyStore) Load() (StoredDeviceKey, ed25519.PrivateKey, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	var stored StoredDeviceKey
	if err := json.Unmarshal(data, &stored); err != nil {
		return StoredDeviceKey{}, nil, err
	}
	sealed, err := base64.RawURLEncoding.DecodeString(stored.ProtectedKey)
	if err != nil {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	plain, err := s.protector.Open("goreecloud-sync-device-key:"+stored.DeviceID, sealed)
	if err != nil {
		return StoredDeviceKey{}, nil, err
	}
	if len(plain) != ed25519.PrivateKeySize {
		return StoredDeviceKey{}, nil, ErrInvalidDeviceKey
	}
	return stored, ed25519.PrivateKey(plain), nil
}

func (s *DeviceKeyStore) Rotate(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, now time.Time) (StoredDeviceKey, error) {
	current, _, err := s.Load()
	if err != nil {
		return StoredDeviceKey{}, err
	}
	stored, err := s.Save(current.DeviceID, publicKey, privateKey, current.CreatedAt)
	if err != nil {
		return StoredDeviceKey{}, err
	}
	stored.RotatedAt = now.UTC()
	if err := s.write(stored); err != nil {
		return StoredDeviceKey{}, err
	}
	return stored, nil
}

func (s *DeviceKeyStore) write(stored StoredDeviceKey) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
