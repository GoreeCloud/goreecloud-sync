package transport

import (
	"context"
	"fmt"
	"net"
	"time"
)

// PeerConn owns one bounded control-plane connection to a synchronization peer.
type PeerConn struct{ conn net.Conn }

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

func (p *PeerConn) Close() error {
	if p == nil || p.conn == nil {
		return nil
	}
	return p.conn.Close()
}

// SendHandshake writes one GC-SYNC/1 capability handshake using the bounded codec.
func (p *PeerConn) SendHandshake(handshake Handshake) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("peer connection is unavailable")
	}
	if err := handshake.Validate(); err != nil {
		return fmt.Errorf("validate local handshake: %w", err)
	}
	if err := WriteJSONFrame(p.conn, handshake); err != nil {
		return fmt.Errorf("send peer handshake: %w", err)
	}
	return nil
}

// ReceiveHandshake reads and validates one peer capability handshake. Successful
// validation confirms protocol shape only; it does not authenticate the peer.
func (p *PeerConn) ReceiveHandshake() (Handshake, error) {
	if p == nil || p.conn == nil {
		return Handshake{}, fmt.Errorf("peer connection is unavailable")
	}
	var handshake Handshake
	if err := ReadJSONFrame(p.conn, &handshake); err != nil {
		return Handshake{}, fmt.Errorf("receive peer handshake: %w", err)
	}
	if err := handshake.Validate(); err != nil {
		return Handshake{}, fmt.Errorf("validate peer handshake: %w", err)
	}
	return handshake, nil
}
