package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidSessionToken = errors.New("invalid sync session token")
	ErrSessionExpired      = errors.New("sync session expired")
)

type PersistedSession struct {
	TokenHash string            `json:"tokenHash"`
	Peer      AuthenticatedPeer `json:"peer"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore(path string) *Store { return &Store{path: path} }

func (s *Store) Put(token string, peer AuthenticatedPeer, expiresAt time.Time) error {
	token = strings.TrimSpace(token)
	if token == "" || strings.TrimSpace(peer.DeviceID) == "" || strings.TrimSpace(peer.KeyFingerprint) == "" || expiresAt.IsZero() {
		return ErrInvalidSessionToken
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	hash := tokenHash(token)
	sessions[hash] = PersistedSession{TokenHash: hash, Peer: peer, ExpiresAt: expiresAt.UTC()}
	return s.save(sessions)
}

func (s *Store) Resolve(token string, now time.Time) (AuthenticatedPeer, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AuthenticatedPeer{}, ErrInvalidSessionToken
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return AuthenticatedPeer{}, err
	}
	hash := tokenHash(token)
	entry, ok := sessions[hash]
	if !ok {
		return AuthenticatedPeer{}, ErrInvalidSessionToken
	}
	if !now.UTC().Before(entry.ExpiresAt) {
		delete(sessions, hash)
		_ = s.save(sessions)
		return AuthenticatedPeer{}, ErrSessionExpired
	}
	return entry.Peer, nil
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (s *Store) load() (map[string]PersistedSession, error) {
	out := map[string]PersistedSession{}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) save(sessions map[string]PersistedSession) error {
	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
