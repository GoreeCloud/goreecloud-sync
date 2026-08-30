package identity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrPairingChallengeUnknown = errors.New("pairing challenge is unknown")
	ErrPairingChallengeExpired = errors.New("pairing challenge is expired")
)

// VerifiedPairing is produced only after a valid pairing proof consumes its
// exact one-time challenge. Callers can inspect the verified identity but cannot
// construct a valid value directly outside this package.
type VerifiedPairing struct {
	deviceID    string
	publicKey   string
	fingerprint string
}

func (p VerifiedPairing) DeviceID() string    { return p.deviceID }
func (p VerifiedPairing) PublicKey() string   { return p.publicKey }
func (p VerifiedPairing) Fingerprint() string { return p.fingerprint }

// ChallengeStore owns short-lived, one-time pairing challenges. It is an
// in-memory development implementation; durable trusted-device state is separate.
type ChallengeStore struct {
	mu         sync.Mutex
	challenges map[string]time.Time
	ttl        time.Duration
	now        func() time.Time
}

func NewChallengeStore(ttl time.Duration) *ChallengeStore {
	return &ChallengeStore{challenges: make(map[string]time.Time), ttl: ttl, now: time.Now}
}

func (s *ChallengeStore) Issue() (string, error) {
	if s == nil || s.ttl <= 0 {
		return "", fmt.Errorf("pairing challenge store unavailable")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate pairing challenge: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(raw[:])
	s.mu.Lock()
	s.challenges[nonce] = s.now().UTC().Add(s.ttl)
	s.mu.Unlock()
	return nonce, nil
}

// VerifyAndConsumePairing proves key possession, consumes the exact one-time
// challenge, and returns the verified device/key binding for explicit approval.
func (s *ChallengeStore) VerifyAndConsumePairing(proof PairingProof) (VerifiedPairing, error) {
	if s == nil {
		return VerifiedPairing{}, ErrPairingChallengeUnknown
	}
	fingerprint, err := proof.Verify()
	if err != nil {
		return VerifiedPairing{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.challenges[proof.Nonce]
	if !ok {
		return VerifiedPairing{}, ErrPairingChallengeUnknown
	}
	delete(s.challenges, proof.Nonce)
	if !s.now().UTC().Before(expiresAt) {
		return VerifiedPairing{}, ErrPairingChallengeExpired
	}
	return VerifiedPairing{
		deviceID:    proof.DeviceID,
		publicKey:   proof.PublicKey,
		fingerprint: fingerprint,
	}, nil
}

// VerifyAndConsume preserves the original fingerprint-only API for callers that
// do not need to persist an approved trust record.
func (s *ChallengeStore) VerifyAndConsume(proof PairingProof) (string, error) {
	verified, err := s.VerifyAndConsumePairing(proof)
	if err != nil {
		return "", err
	}
	return verified.Fingerprint(), nil
}
