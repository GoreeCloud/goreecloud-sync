package transport

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PeerConn owns one bounded control-plane connection to a synchronization peer.
// Raw DialPeer/AcceptPeer connections leave authenticated identity fields empty.
// Secure peer constructors populate them after TLS authentication so the
// GC-SYNC/1 capability handshake and later trust revalidation can remain bound
// to the exact admitted device identity.
type PeerConn struct {
	conn                        net.Conn
	localDeviceID               string
	authenticatedDeviceID       string
	trustMu                     sync.RWMutex
	authenticatedKeyFingerprint string
	closed                      atomic.Bool
}

// DialPeer establishes a context-bound TCP connection. Authentication and
// encryption are intentionally layered above this primitive rather than implied.
func DialPeer(ctx context.Context, address string, timeout time.Duration) (*PeerConn, error) {
	if address == "" || timeout <= 0 {
		return nil, fmt.Errorf("peer address and timeout are required")
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial peer: %w", err)
	}
	return &PeerConn{conn: conn}, nil
}

// AcceptPeer wraps an accepted stream without assigning trust to it.
func AcceptPeer(conn net.Conn) (*PeerConn, error) {
	if conn == nil {
		return nil, fmt.Errorf("peer connection must not be nil")
	}
	return &PeerConn{conn: conn}, nil
}

// AuthenticatedDeviceID returns the peer identity cryptographically bound by a
// secure peer constructor. Raw peers return an empty string.
func (p *PeerConn) AuthenticatedDeviceID() string {
	if p == nil {
		return ""
	}
	return p.authenticatedDeviceID
}

// AuthenticatedKeyFingerprint returns the durable trusted-device fingerprint
// associated with the key pinned during secure admission. It is populated by
// the application trust factory; raw peers return an empty string.
func (p *PeerConn) AuthenticatedKeyFingerprint() string {
	if p == nil {
		return ""
	}
	p.trustMu.RLock()
	defer p.trustMu.RUnlock()
	return p.authenticatedKeyFingerprint
}

// BindAuthenticatedKeyFingerprint attaches the durable trust-store fingerprint
// for the key already authenticated by the secure transport. The binding is
// write-once: callers cannot replace it after admission. Raw unauthenticated
// peers cannot acquire a durable trust fingerprint through this method.
func (p *PeerConn) BindAuthenticatedKeyFingerprint(fingerprint string) error {
	if p == nil || p.conn == nil || p.IsClosed() || p.authenticatedDeviceID == "" {
		return fmt.Errorf("authenticated peer connection is unavailable")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return fmt.Errorf("authenticated key fingerprint is required")
	}
	p.trustMu.Lock()
	defer p.trustMu.Unlock()
	if p.authenticatedKeyFingerprint != "" {
		if p.authenticatedKeyFingerprint == fingerprint {
			return nil
		}
		return fmt.Errorf("authenticated key fingerprint is already bound")
	}
	p.authenticatedKeyFingerprint = fingerprint
	return nil
}

// IsClosed reports whether Close has been invoked on the PeerConn. It is a
// local lifecycle signal, not evidence that a remote peer observed closure.
func (p *PeerConn) IsClosed() bool {
	return p == nil || p.closed.Load()
}

func (p *PeerConn) Close() error {
	if p == nil || p.conn == nil {
		return nil
	}
	if p.closed.Swap(true) {
		return nil
	}
	return p.conn.Close()
}

// SendHandshake writes one GC-SYNC/1 capability handshake using the bounded codec.
func (p *PeerConn) SendHandshake(handshake Handshake) error {
	if p == nil || p.conn == nil || p.IsClosed() {
		return fmt.Errorf("peer connection is unavailable")
	}
	if err := handshake.Validate(); err != nil {
		return fmt.Errorf("validate local handshake: %w", err)
	}
	if p.localDeviceID != "" && handshake.DeviceID != p.localDeviceID {
		return fmt.Errorf("local handshake device ID does not match authenticated TLS identity")
	}
	if err := WriteJSONFrame(p.conn, handshake); err != nil {
		return fmt.Errorf("send peer handshake: %w", err)
	}
	return nil
}

// ReceiveHandshake reads and validates one peer capability handshake. Raw peers
// validate protocol shape only. Secure peers additionally require the claimed
// device ID to match the identity authenticated by TLS.
func (p *PeerConn) ReceiveHandshake() (Handshake, error) {
	if p == nil || p.conn == nil || p.IsClosed() {
		return Handshake{}, fmt.Errorf("peer connection is unavailable")
	}
	var handshake Handshake
	if err := ReadJSONFrame(p.conn, &handshake); err != nil {
		return Handshake{}, fmt.Errorf("receive peer handshake: %w", err)
	}
	if err := handshake.Validate(); err != nil {
		return Handshake{}, fmt.Errorf("validate peer handshake: %w", err)
	}
	if p.authenticatedDeviceID != "" && handshake.DeviceID != p.authenticatedDeviceID {
		return Handshake{}, fmt.Errorf("peer handshake device ID does not match authenticated TLS identity")
	}
	return handshake, nil
}
