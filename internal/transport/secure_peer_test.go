package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"
)

const (
	testClientDeviceID = "11111111-1111-1111-1111-111111111111"
	testServerDeviceID = "22222222-2222-2222-2222-222222222222"
	testOtherDeviceID  = "33333333-3333-3333-3333-333333333333"
)

type securePeerResult struct {
	peer *PeerConn
	err  error
}

func TestSecurePeerTLS13RoundTripAndHandshakeBinding(t *testing.T) {
	client, server := newSecurePeerPair(t)
	defer client.Close()
	defer server.Close()

	if client.AuthenticatedDeviceID() != testServerDeviceID {
		t.Fatalf("client authenticated device = %q, want %q", client.AuthenticatedDeviceID(), testServerDeviceID)
	}
	if server.AuthenticatedDeviceID() != testClientDeviceID {
		t.Fatalf("server authenticated device = %q, want %q", server.AuthenticatedDeviceID(), testClientDeviceID)
	}
	clientTLS, ok := client.conn.(*tls.Conn)
	if !ok {
		t.Fatal("client connection is not TLS")
	}
	state := clientTLS.ConnectionState()
	if state.Version != tls.VersionTLS13 {
		t.Fatalf("TLS version = %x, want TLS 1.3", state.Version)
	}
	if state.NegotiatedProtocol != securePeerALPN {
		t.Fatalf("ALPN = %q, want %q", state.NegotiatedProtocol, securePeerALPN)
	}

	clientHandshake := Handshake{Protocol: Protocol, DeviceID: testClientDeviceID, Features: []string{"resume"}}
	if err := client.SendHandshake(clientHandshake); err != nil {
		t.Fatalf("send client handshake: %v", err)
	}
	gotClient, err := server.ReceiveHandshake()
	if err != nil {
		t.Fatalf("receive client handshake: %v", err)
	}
	if gotClient.DeviceID != testClientDeviceID {
		t.Fatalf("client handshake device = %q", gotClient.DeviceID)
	}

	serverHandshake := Handshake{Protocol: Protocol, DeviceID: testServerDeviceID, Features: []string{"resume"}}
	if err := server.SendHandshake(serverHandshake); err != nil {
		t.Fatalf("send server handshake: %v", err)
	}
	gotServer, err := client.ReceiveHandshake()
	if err != nil {
		t.Fatalf("receive server handshake: %v", err)
	}
	if gotServer.DeviceID != testServerDeviceID {
		t.Fatalf("server handshake device = %q", gotServer.DeviceID)
	}
}

func TestSecurePeerRejectsPinnedKeyMismatch(t *testing.T) {
	clientPublic, clientPrivate := newTestEd25519Key(t)
	serverPublic, serverPrivate := newTestEd25519Key(t)
	wrongServerPublic, _ := newTestEd25519Key(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverResult := make(chan securePeerResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- securePeerResult{err: acceptErr}
			return
		}
		peer, acceptErr := AcceptSecurePeer(ctx, conn, 2*time.Second,
			SecurePeerIdentity{DeviceID: testServerDeviceID, PrivateKey: serverPrivate},
			TrustedPeerIdentity{DeviceID: testClientDeviceID, PublicKey: clientPublic})
		serverResult <- securePeerResult{peer: peer, err: acceptErr}
	}()

	client, err := DialSecurePeer(ctx, listener.Addr().String(), 2*time.Second,
		SecurePeerIdentity{DeviceID: testClientDeviceID, PrivateKey: clientPrivate},
		TrustedPeerIdentity{DeviceID: testServerDeviceID, PublicKey: wrongServerPublic})
	if client != nil {
		_ = client.Close()
	}
	if err == nil || !errors.Is(err, ErrSecurePeerAuthentication) {
		t.Fatalf("client error = %v, want secure peer authentication failure", err)
	}
	result := <-serverResult
	if result.peer != nil {
		_ = result.peer.Close()
	}
	if result.err == nil {
		t.Fatal("server unexpectedly accepted client after pinned-key failure")
	}
	_ = serverPublic
}

func TestSecurePeerServerRejectsPinnedClientKeyMismatch(t *testing.T) {
	clientPublic, clientPrivate := newTestEd25519Key(t)
	serverPublic, serverPrivate := newTestEd25519Key(t)
	wrongClientPublic, _ := newTestEd25519Key(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverResult := make(chan securePeerResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- securePeerResult{err: acceptErr}
			return
		}
		peer, acceptErr := AcceptSecurePeer(ctx, conn, 2*time.Second,
			SecurePeerIdentity{DeviceID: testServerDeviceID, PrivateKey: serverPrivate},
			TrustedPeerIdentity{DeviceID: testClientDeviceID, PublicKey: wrongClientPublic})
		serverResult <- securePeerResult{peer: peer, err: acceptErr}
	}()

	client, dialErr := DialSecurePeer(ctx, listener.Addr().String(), 2*time.Second,
		SecurePeerIdentity{DeviceID: testClientDeviceID, PrivateKey: clientPrivate},
		TrustedPeerIdentity{DeviceID: testServerDeviceID, PublicKey: serverPublic})
	if client != nil {
		_ = client.Close()
	}
	if dialErr == nil {
		t.Fatal("client unexpectedly completed TLS after server rejected its key")
	}
	result := <-serverResult
	if result.peer != nil {
		_ = result.peer.Close()
	}
	if result.err == nil || !errors.Is(result.err, ErrSecurePeerAuthentication) {
		t.Fatalf("server error = %v, want secure peer authentication failure", result.err)
	}
	_ = clientPublic
}

func TestSecurePeerRejectsPinnedDeviceIDMismatch(t *testing.T) {
	clientPublic, clientPrivate := newTestEd25519Key(t)
	serverPublic, serverPrivate := newTestEd25519Key(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverResult := make(chan securePeerResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- securePeerResult{err: acceptErr}
			return
		}
		peer, acceptErr := AcceptSecurePeer(ctx, conn, 2*time.Second,
			SecurePeerIdentity{DeviceID: testServerDeviceID, PrivateKey: serverPrivate},
			TrustedPeerIdentity{DeviceID: testClientDeviceID, PublicKey: clientPublic})
		serverResult <- securePeerResult{peer: peer, err: acceptErr}
	}()

	client, err := DialSecurePeer(ctx, listener.Addr().String(), 2*time.Second,
		SecurePeerIdentity{DeviceID: testClientDeviceID, PrivateKey: clientPrivate},
		TrustedPeerIdentity{DeviceID: testOtherDeviceID, PublicKey: serverPublic})
	if client != nil {
		_ = client.Close()
	}
	if err == nil || !errors.Is(err, ErrSecurePeerAuthentication) {
		t.Fatalf("client error = %v, want secure peer authentication failure", err)
	}
	result := <-serverResult
	if result.peer != nil {
		_ = result.peer.Close()
	}
	if result.err == nil {
		t.Fatal("server unexpectedly accepted client after device-ID pin failure")
	}
}

func TestSecurePeerRejectsHandshakeIdentityMismatch(t *testing.T) {
	client, server := newSecurePeerPair(t)
	defer client.Close()
	defer server.Close()

	if err := client.SendHandshake(Handshake{Protocol: Protocol, DeviceID: testOtherDeviceID}); err == nil {
		t.Fatal("expected local TLS/handshake identity mismatch")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- WriteJSONFrame(client.conn, Handshake{Protocol: Protocol, DeviceID: testOtherDeviceID})
	}()
	if _, err := server.ReceiveHandshake(); err == nil {
		t.Fatal("expected authenticated peer/handshake identity mismatch")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("write mismatched handshake: %v", err)
	}
}

func TestSecurePeerRejectsInvalidIdentityBeforeDial(t *testing.T) {
	publicKey, _ := newTestEd25519Key(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := DialSecurePeer(ctx, "127.0.0.1:1", time.Second,
		SecurePeerIdentity{DeviceID: testClientDeviceID, PrivateKey: ed25519.PrivateKey("short")},
		TrustedPeerIdentity{DeviceID: testServerDeviceID, PublicKey: publicKey}); err == nil {
		t.Fatal("expected invalid local identity rejection")
	}
}

func newSecurePeerPair(t *testing.T) (*PeerConn, *PeerConn) {
	t.Helper()
	clientPublic, clientPrivate := newTestEd25519Key(t)
	serverPublic, serverPrivate := newTestEd25519Key(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	serverResult := make(chan securePeerResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- securePeerResult{err: acceptErr}
			return
		}
		peer, acceptErr := AcceptSecurePeer(ctx, conn, 2*time.Second,
			SecurePeerIdentity{DeviceID: testServerDeviceID, PrivateKey: serverPrivate},
			TrustedPeerIdentity{DeviceID: testClientDeviceID, PublicKey: clientPublic})
		serverResult <- securePeerResult{peer: peer, err: acceptErr}
	}()

	client, err := DialSecurePeer(ctx, listener.Addr().String(), 2*time.Second,
		SecurePeerIdentity{DeviceID: testClientDeviceID, PrivateKey: clientPrivate},
		TrustedPeerIdentity{DeviceID: testServerDeviceID, PublicKey: serverPublic})
	if err != nil {
		t.Fatalf("dial secure peer: %v", err)
	}
	result := <-serverResult
	if result.err != nil {
		_ = client.Close()
		t.Fatalf("accept secure peer: %v", result.err)
	}
	return client, result.peer
}

func newTestEd25519Key(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return publicKey, privateKey
}
