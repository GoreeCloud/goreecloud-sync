package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrTrustedDeviceNotFound     = errors.New("trusted device not found")
	ErrTrustedDeviceKeyMismatch  = errors.New("trusted device key mismatch")
	ErrInvalidTrustedDeviceStore = errors.New("invalid trusted device store")
)

type TrustedDevice struct {
	AccountID    string     `json:"accountId"`
	DeviceID     string     `json:"deviceId"`
	PublicKey    string     `json:"publicKey"`
	Fingerprint  string     `json:"fingerprint"`
	AuthorizedAt time.Time  `json:"authorizedAt"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
}

type trustedDeviceFile struct {
	Version int             `json:"version"`
	Devices []TrustedDevice `json:"devices"`
}

// TrustedDeviceStore persists explicit account-scoped device authorization.
// Pairing proof alone is insufficient: Authorize only accepts a VerifiedPairing
// produced after successful one-time challenge consumption.
type TrustedDeviceStore struct {
	path string
	mu   sync.Mutex
}

func NewTrustedDeviceStore(path string) (*TrustedDeviceStore, error) {
	if path == "" {
		return nil, fmt.Errorf("trusted device store path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve trusted device store path: %w", err)
	}
	return &TrustedDeviceStore{path: filepath.Clean(absolute)}, nil
}

func (s *TrustedDeviceStore) Authorize(accountID string, pairing VerifiedPairing, now time.Time) (TrustedDevice, error) {
	if s == nil || accountID == "" || pairing.DeviceID() == "" || pairing.PublicKey() == "" || pairing.Fingerprint() == "" {
		return TrustedDevice{}, ErrInvalidTrustedDeviceStore
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return TrustedDevice{}, err
	}
	for index, existing := range state.Devices {
		if existing.AccountID != accountID || existing.DeviceID != pairing.DeviceID() {
			continue
		}
		if existing.RevokedAt == nil {
			if existing.Fingerprint != pairing.Fingerprint() || existing.PublicKey != pairing.PublicKey() {
				return TrustedDevice{}, ErrTrustedDeviceKeyMismatch
			}
			return existing, nil
		}
		replacement := trustedDeviceFromPairing(accountID, pairing, now)
		state.Devices[index] = replacement
		if err := s.writeLocked(state); err != nil {
			return TrustedDevice{}, err
		}
		return replacement, nil
	}

	device := trustedDeviceFromPairing(accountID, pairing, now)
	state.Devices = append(state.Devices, device)
	if err := s.writeLocked(state); err != nil {
		return TrustedDevice{}, err
	}
	return device, nil
}

func (s *TrustedDeviceStore) IsTrusted(accountID, deviceID, fingerprint string) (bool, error) {
	if s == nil || accountID == "" || !validDeviceID(deviceID) || fingerprint == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return false, err
	}
	for _, device := range state.Devices {
		if device.AccountID == accountID && device.DeviceID == deviceID {
			return device.RevokedAt == nil && device.Fingerprint == fingerprint, nil
		}
	}
	return false, nil
}

func (s *TrustedDeviceStore) Revoke(accountID, deviceID string, now time.Time) (TrustedDevice, error) {
	if s == nil || accountID == "" || !validDeviceID(deviceID) {
		return TrustedDevice{}, ErrTrustedDeviceNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return TrustedDevice{}, err
	}
	for index, device := range state.Devices {
		if device.AccountID != accountID || device.DeviceID != deviceID {
			continue
		}
		if device.RevokedAt != nil {
			return device, nil
		}
		revokedAt := now.UTC()
		device.RevokedAt = &revokedAt
		state.Devices[index] = device
		if err := s.writeLocked(state); err != nil {
			return TrustedDevice{}, err
		}
		return device, nil
	}
	return TrustedDevice{}, ErrTrustedDeviceNotFound
}

func (s *TrustedDeviceStore) List(accountID string) ([]TrustedDevice, error) {
	if s == nil || accountID == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	devices := make([]TrustedDevice, 0, len(state.Devices))
	for _, device := range state.Devices {
		if device.AccountID == accountID {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

func trustedDeviceFromPairing(accountID string, pairing VerifiedPairing, now time.Time) TrustedDevice {
	return TrustedDevice{
		AccountID:    accountID,
		DeviceID:     pairing.DeviceID(),
		PublicKey:    pairing.PublicKey(),
		Fingerprint:  pairing.Fingerprint(),
		AuthorizedAt: now.UTC(),
	}
}

func (s *TrustedDeviceStore) loadLocked() (trustedDeviceFile, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return trustedDeviceFile{Version: 1}, nil
	}
	if err != nil {
		return trustedDeviceFile{}, fmt.Errorf("read trusted device store: %w", err)
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return trustedDeviceFile{}, fmt.Errorf("inspect trusted device store: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return trustedDeviceFile{}, fmt.Errorf("%w: permissions must not grant group or other access", ErrInvalidTrustedDeviceStore)
	}
	var state trustedDeviceFile
	if err := json.Unmarshal(data, &state); err != nil {
		return trustedDeviceFile{}, fmt.Errorf("%w: %v", ErrInvalidTrustedDeviceStore, err)
	}
	if state.Version != 1 {
		return trustedDeviceFile{}, fmt.Errorf("%w: unsupported version", ErrInvalidTrustedDeviceStore)
	}
	for _, device := range state.Devices {
		if err := validateTrustedDevice(device); err != nil {
			return trustedDeviceFile{}, err
		}
	}
	return state, nil
}

func validateTrustedDevice(device TrustedDevice) error {
	if device.AccountID == "" || !validDeviceID(device.DeviceID) || device.AuthorizedAt.IsZero() {
		return ErrInvalidTrustedDeviceStore
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(device.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return ErrInvalidTrustedDeviceStore
	}
	if Fingerprint(ed25519.PublicKey(publicKey)) != device.Fingerprint {
		return ErrInvalidTrustedDeviceStore
	}
	return nil
}

func (s *TrustedDeviceStore) writeLocked(state trustedDeviceFile) error {
	state.Version = 1
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create trusted device store directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure trusted device store directory: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trusted device store: %w", err)
	}
	tmp, err := os.CreateTemp(directory, ".trusted-devices-*.tmp")
	if err != nil {
		return fmt.Errorf("create trusted device store temp file: %w", err)
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
		return fmt.Errorf("secure trusted device store temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write trusted device store: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync trusted device store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close trusted device store: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("publish trusted device store: %w", err)
	}
	keep = true
	return nil
}
