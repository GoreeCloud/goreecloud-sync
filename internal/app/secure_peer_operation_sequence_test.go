package app

import (
	"errors"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/transport"
)

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
