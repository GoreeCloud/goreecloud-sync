package app

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/transport"
)

const (
	secureFactoryAccountID = "account-a"
	secureFactoryClientID  = "11111111-1111-1111-1111-111111111111"
	secureFactoryServerID  = "22222222-2222-2222-2222-222222222222"
)

type secureFactoryProtector struct{}

func (secureFactoryProtector) Seal(_ string, plaintext []byte) ([]byte, error) {
	return append([]byte("sealed:"), plaintext...), nil
}

func (secureFactoryProtector) Open(_ string, ciphertext []byte) ([]byte, error) {
	const prefix = "sealed:"
	if len(ciphertext) < len(prefix) || string(ciphertext[:len(prefix)]) != prefix {
		return nil, identity.ErrInvalidDeviceKey
	}
	return append([]byte(nil), ciphertext[len(prefix):]...), nil
}

type secureFactoryFixture struct {
	factory *SecurePeerFactory
	trust   *identity.TrustedDeviceStore
}

type secureFactoryResult struct {
	peer *transport.PeerConn
	err  error
}

func TestSecurePeerFactoryComposesCurrentTrustIntoTLS(t *testing.T) {
	clientPublic, clientPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	serverPublic, serverPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	client := newSecureFactoryFixture(t, "client", secureFactoryAccountID, secureFactoryClientID, clientPublic, clientPrivate, secureFactoryServerID, serverPublic, serverPrivate)
	server := newSecureFactoryFixture(t, "server", secureFactoryAccountID, secureFactoryServerID, serverPublic, serverPrivate, secureFactoryClientID, clientPublic, clientPrivate)

	local, remote, err := client.factory.ResolveIdentities(secureFactoryServerID)
	if err != nil {
		t.Fatal(err)
	}
	if local.DeviceID != secureFactoryClientID || remote.DeviceID != secureFactoryServerID || identity.Fingerprint(remote.PublicKey) != identity.Fingerprint(serverPublic) {
		t.Fatalf("resolved identities local=%q remote=%q fingerprint=%q", local.DeviceID, remote.DeviceID, identity.Fingerprint(remote.PublicKey))
	}

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
	defer clientPeer.Close()
	result := <-serverResult
	if result.err != nil {
		t.Fatalf("accept secure peer through factory: %v", result.err)
	}
	defer result.peer.Close()
	if clientPeer.AuthenticatedDeviceID() != secureFactoryServerID || result.peer.AuthenticatedDeviceID() != secureFactoryClientID {
		t.Fatalf("authenticated identities client=%q server=%q", clientPeer.AuthenticatedDeviceID(), result.peer.AuthenticatedDeviceID())
	}
}

func TestSecurePeerFactoryRejectsRevokedRemoteBeforeTransport(t *testing.T) {
	clientPublic, clientPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	serverPublic, serverPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	fixture := newSecureFactoryFixture(t, "client", secureFactoryAccountID, secureFactoryClientID, clientPublic, clientPrivate, secureFactoryServerID, serverPublic, serverPrivate)
	if _, err := fixture.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fixture.factory.ResolveIdentities(secureFactoryServerID); !errors.Is(err, identity.ErrTrustedDeviceNotFound) {
		t.Fatalf("resolve revoked remote error=%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := fixture.factory.DialSecurePeer(ctx, "127.0.0.1:1", time.Second, secureFactoryServerID); !errors.Is(err, identity.ErrTrustedDeviceNotFound) {
		t.Fatalf("dial revoked remote error=%v", err)
	}
}

func TestSecurePeerFactoryRejectsCrossAccountTrust(t *testing.T) {
	clientPublic, clientPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	serverPublic, serverPrivate, err := identity.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	fixture := newSecureFactoryFixture(t, "client", secureFactoryAccountID, secureFactoryClientID, clientPublic, clientPrivate, secureFactoryServerID, serverPublic, serverPrivate)
	fixture.factory.AccountID = "account-b"
	if _, _, err := fixture.factory.ResolveIdentities(secureFactoryServerID); !errors.Is(err, identity.ErrTrustedDeviceNotFound) {
		t.Fatalf("resolve cross-account remote error=%v", err)
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

func TestRunWithCurrentTrustRejectsMissingOperation(t *testing.T) {
	if err := (SecurePeerFactory{}).RunWithCurrentTrust(nil, nil); !errors.Is(err, ErrSecurePeerOperationRequired) {
		t.Fatalf("missing operation error = %v", err)
	}
}

func TestRunOperationSequenceRevalidatesTrustBeforeEveryStep(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer serverPeer.Close()

	called := 0
	err := client.factory.RunOperationSequenceWithCurrentTrust(clientPeer, []func(*transport.PeerConn) error{
		func(*transport.PeerConn) error {
			called++
			_, revokeErr := client.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now())
			return revokeErr
		},
		func(*transport.PeerConn) error {
			called++
			return nil
		},
	})
	if !errors.Is(err, ErrSecurePeerTrustNotCurrent) {
		t.Fatalf("sequence after revocation error = %v", err)
	}
	if called != 1 {
		t.Fatalf("operations called=%d, want 1", called)
	}
	if !clientPeer.IsClosed() {
		t.Fatal("peer must be closed when trust is revoked between operations")
	}
}

func TestRunOperationSequenceValidatesAllSlotsBeforeExecution(t *testing.T) {
	called := false
	err := (SecurePeerFactory{}).RunOperationSequenceWithCurrentTrust(nil, []func(*transport.PeerConn) error{
		func(*transport.PeerConn) error {
			called = true
			return nil
		},
		nil,
	})
	if !errors.Is(err, ErrSecurePeerOperationRequired) {
		t.Fatalf("invalid operation slot error = %v", err)
	}
	if called {
		t.Fatal("malformed operation sequence must not partially execute")
	}
}

func TestRunOperationSequenceRejectsInvalidBounds(t *testing.T) {
	factory := SecurePeerFactory{}
	if err := factory.RunOperationSequenceWithCurrentTrust(nil, nil); !errors.Is(err, ErrSecurePeerOperationSequence) {
		t.Fatalf("empty sequence error = %v", err)
	}
	tooMany := make([]func(*transport.PeerConn) error, maxSecurePeerOperationSequence+1)
	if err := factory.RunOperationSequenceWithCurrentTrust(nil, tooMany); !errors.Is(err, ErrSecurePeerOperationSequence) {
		t.Fatalf("oversized sequence error = %v", err)
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
	client := newSecureFactoryFixture(t, "client-live", secureFactoryAccountID, secureFactoryClientID, clientPublic, clientPrivate, secureFactoryServerID, serverPublic, serverPrivate)
	server := newSecureFactoryFixture(t, "server-live", secureFactoryAccountID, secureFactoryServerID, serverPublic, serverPrivate, secureFactoryClientID, clientPublic, clientPrivate)

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
		t.Fatalf("dial secure peer: %v", err)
	}
	result := <-serverResult
	if result.err != nil {
		_ = clientPeer.Close()
		t.Fatalf("accept secure peer: %v", result.err)
	}
	return client, server, clientPeer, result.peer
}

func newSecureFactoryFixture(t *testing.T, name, accountID, localDeviceID string, localPublic ed25519.PublicKey, localPrivate ed25519.PrivateKey, remoteDeviceID string, remotePublic ed25519.PublicKey, remotePrivate ed25519.PrivateKey) secureFactoryFixture {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	localStore, err := identity.NewDeviceKeyStore(filepath.Join(root, "device-key.json"), secureFactoryProtector{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := localStore.Save(localDeviceID, localPublic, localPrivate, time.Now()); err != nil {
		t.Fatal(err)
	}
	trustStore, err := identity.NewTrustedDeviceStore(filepath.Join(root, "trusted-devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	challenges := identity.NewChallengeStore(time.Minute)
	nonce, err := challenges.Issue()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := identity.NewPairingProof(remoteDeviceID, nonce, remotePublic, remotePrivate)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := challenges.VerifyAndConsumePairing(proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := trustStore.Authorize(accountID, verified, time.Now()); err != nil {
		t.Fatal(err)
	}
	return secureFactoryFixture{
		factory: &SecurePeerFactory{AccountID: accountID, LocalKeys: localStore, TrustedDevices: trustStore},
		trust:   trustStore,
	}
}
