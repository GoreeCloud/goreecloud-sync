package app

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/transport"
)

func TestSecurePeerFactoryRevalidatesBoundCurrentTrust(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer clientPeer.Close()
	defer serverPeer.Close()

	serverDevice, serverKey, err := client.trust.Resolve(secureFactoryAccountID, secureFactoryServerID)
	if err != nil {
		t.Fatal(err)
	}
	if clientPeer.AuthenticatedDeviceID() != secureFactoryServerID {
		t.Fatalf("authenticated device = %q", clientPeer.AuthenticatedDeviceID())
	}
	if clientPeer.AuthenticatedKeyFingerprint() != serverDevice.Fingerprint {
		t.Fatalf("bound fingerprint = %q, trust fingerprint = %q", clientPeer.AuthenticatedKeyFingerprint(), serverDevice.Fingerprint)
	}
	if clientPeer.AuthenticatedKeyFingerprint() != identity.Fingerprint(serverKey) {
		t.Fatal("bound fingerprint does not match pinned public key")
	}
	if err := client.factory.RevalidatePeer(clientPeer); err != nil {
		t.Fatalf("revalidate current trusted peer: %v", err)
	}
	if clientPeer.IsClosed() {
		t.Fatal("current trusted peer must remain open")
	}
}

func TestSecurePeerFactoryClosesPeerWhenTrustIsRevoked(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer serverPeer.Close()

	if _, err := client.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := client.factory.RevalidatePeer(clientPeer); !errors.Is(err, ErrSecurePeerTrustNotCurrent) {
		t.Fatalf("revalidate revoked peer error = %v", err)
	}
	if !clientPeer.IsClosed() {
		t.Fatal("revoked peer must be closed by trust revalidation")
	}
	if err := clientPeer.Close(); err != nil {
		t.Fatalf("repeated close must remain safe: %v", err)
	}
}

func TestRunWithCurrentTrustExecutesOnlyForCurrentPeer(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer clientPeer.Close()
	defer serverPeer.Close()

	called := 0
	if err := client.factory.RunWithCurrentTrust(clientPeer, func(peer *transport.PeerConn) error {
		called++
		if peer != clientPeer {
			t.Fatal("operation received unexpected peer")
		}
		return nil
	}); err != nil {
		t.Fatalf("run trusted operation: %v", err)
	}
	if called != 1 || clientPeer.IsClosed() {
		t.Fatalf("trusted operation called=%d closed=%v", called, clientPeer.IsClosed())
	}
}

func TestRunWithCurrentTrustBlocksOperationAfterRevocation(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer serverPeer.Close()

	if _, err := client.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now()); err != nil {
		t.Fatal(err)
	}
	called := false
	err := client.factory.RunWithCurrentTrust(clientPeer, func(*transport.PeerConn) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrSecurePeerTrustNotCurrent) {
		t.Fatalf("run revoked operation error = %v", err)
	}
	if called {
		t.Fatal("operation must not execute after trust revocation")
	}
	if !clientPeer.IsClosed() {
		t.Fatal("revoked peer must be closed before operation returns")
	}
}

func TestRunWithCurrentTrustRejectsMissingOperation(t *testing.T) {
	if err := (SecurePeerFactory{}).RunWithCurrentTrust(nil, nil); !errors.Is(err, ErrSecurePeerOperationRequired) {
		t.Fatalf("missing operation error = %v", err)
	}
}

func TestSecurePeerFactoryRejectsUnauthenticatedPeerAtRevalidation(t *testing.T) {
	local, remote := net.Pipe()
	defer remote.Close()
	peer, err := transport.AcceptPeer(local)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &SecurePeerFactory{AccountID: secureFactoryAccountID}
	if err := fixture.RevalidatePeer(peer); !errors.Is(err, ErrSecurePeerFactoryUnavailable) {
		t.Fatalf("revalidate unavailable factory error = %v", err)
	}
	if !peer.IsClosed() {
		t.Fatal("peer must close when revalidation authority is unavailable")
	}
}

func establishSecureFactoryPeers(t *testing.T) (secureFactoryFixture, secureFactoryFixture, *transport.PeerConn, *transport.PeerConn) {
	t.Helper()
	clientPublic, clientPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	serverPublic, serverPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	client := newSecureFactoryFixture(t, "revalidate-client", secureFactoryAccountID, secureFactoryClientID, clientPublic, clientPrivate, secureFactoryServerID, serverPublic, serverPrivate)
	server := newSecureFactoryFixture(t, "revalidate-server", secureFactoryAccountID, secureFactoryServerID, serverPublic, serverPrivate, secureFactoryClientID, clientPublic, clientPrivate)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverResult := make(chan secureFactoryResult, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- secureFactoryResult{err: acceptErr}
			return
		}
		peer, acceptErr := server.factory.AcceptSecurePeer(ctx, conn, 2*time.Second, secureFactoryClientID)
		serverResult <- secureFactoryResult{peer: peer, err: acceptErr}
	}()

	clientPeer, err := client.factory.DialSecurePeer(ctx, listener.Addr().String(), 2*time.Second, secureFactoryServerID)
	if err != nil {
		t.Fatalf("dial secure peer through factory: %v", err)
	}
	result := <-serverResult
	if result.err != nil {
		_ = clientPeer.Close()
		t.Fatalf("accept secure peer through factory: %v", result.err)
	}
	return client, server, clientPeer, result.peer
}
