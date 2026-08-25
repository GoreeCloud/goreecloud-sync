package transport

import (
	"context"
	"fmt"
	"net"
	"time"
)

// PeerConn owns one bounded control-plane connection to a synchronization peer.
type PeerConn struct {
	conn net.Conn
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
	if conn == nil { return nil, fmt.Errorf("peer connection must not be nil") }
	return &PeerConn{conn: conn}, nil
}

func (p *PeerConn) Close() error {
	if p == nil || p.conn == nil { return nil }
	return p.conn.Close()
}

// SendFrame writes one GC-SYNC/1 control frame using the existing bounded codec.
func (p *PeerConn) SendFrame(frame Frame) error {
	if p == nil || p.conn == nil { return fmt.Errorf("peer connection is unavailable") }
	if err := WriteFrame(p.conn, frame); err != nil { return fmt.Errorf("send peer frame: %w", err) }
	return nil
}

// ReceiveFrame reads one bounded GC-SYNC/1 control frame.
func (p *PeerConn) ReceiveFrame() (Frame, error) {
	if p == nil || p.conn == nil { return Frame{}, fmt.Errorf("peer connection is unavailable") }
	frame, err := ReadFrame(p.conn)
	if err != nil { return Frame{}, fmt.Errorf("receive peer frame: %w", err) }
	return frame, nil
}
