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

// VerifyAndConsume proves key possession and then consumes the exact challenge.
// A successful challenge cannot be replayed. Expired challenges are deleted.
func (s *ChallengeStore) VerifyAndConsume(proof PairingProof) (string, error) {
	if s == nil {
		return "", ErrPairingChallengeUnknown
	}
	fingerprint, err := proof.Verify()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.challenges[proof.Nonce]
	if !ok {
		return "", ErrPairingChallengeUnknown
	}
	delete(s.challenges, proof.Nonce)
	if !s.now().UTC().Before(expiresAt) {
		return "", ErrPairingChallengeExpired
	}
	return fingerprint, nil
}
