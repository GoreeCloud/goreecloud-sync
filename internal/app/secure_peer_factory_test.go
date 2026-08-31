package app

import (
	"context"
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

func newSecureFactoryFixture(t *testing.T, name, accountID, localDeviceID string, localPublic, localPrivate []byte, remoteDeviceID string, remotePublic, remotePrivate []byte) secureFactoryFixture {
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
