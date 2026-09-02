package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/transport"
)

const maxSecurePeerOperationSequence = 64

var (
	ErrSecurePeerFactoryUnavailable      = errors.New("secure peer factory is unavailable")
	ErrSecurePeerTrustNotCurrent         = errors.New("secure peer trust is no longer current")
	ErrSecurePeerOperationRequired       = errors.New("secure peer operation is required")
	ErrSecurePeerOperationSequence       = errors.New("secure peer operation sequence is invalid")
)

// SecurePeerFactory resolves current durable remote-device trust and the local
// protected device identity immediately before secure transport admission. It
// deliberately does not own discovery, addressing, listener selection,
// reconnect policy, or broader connection lifecycle management.
type SecurePeerFactory struct {
	AccountID      string
	LocalKeys      *identity.DeviceKeyStore
	TrustedDevices *identity.TrustedDeviceStore
}

// ResolveIdentities returns the exact local and remote identities required by
// the TLS 1.3 secure-peer transport. Remote trust is resolved first so a
// revoked/unknown device fails before local private-key material is opened.
func (f SecurePeerFactory) ResolveIdentities(remoteDeviceID string) (transport.SecurePeerIdentity, transport.TrustedPeerIdentity, error) {
	if f.AccountID == "" || f.LocalKeys == nil || f.TrustedDevices == nil {
		return transport.SecurePeerIdentity{}, transport.TrustedPeerIdentity{}, ErrSecurePeerFactoryUnavailable
	}
	remote, remotePublicKey, err := f.TrustedDevices.Resolve(f.AccountID, remoteDeviceID)
	if err != nil {
		return transport.SecurePeerIdentity{}, transport.TrustedPeerIdentity{}, fmt.Errorf("resolve trusted remote device: %w", err)
	}
	local, localPrivateKey, err := f.LocalKeys.Load()
	if err != nil {
		return transport.SecurePeerIdentity{}, transport.TrustedPeerIdentity{}, fmt.Errorf("load local secure peer identity: %w", err)
	}
	return transport.SecurePeerIdentity{
			DeviceID:   local.DeviceID,
			PrivateKey: localPrivateKey,
		}, transport.TrustedPeerIdentity{
			DeviceID:  remote.DeviceID,
			PublicKey: remotePublicKey,
		}, nil
}

// DialSecurePeer resolves current identity/trust state and then establishes one
// authenticated TLS 1.3 peer session to an address selected by the caller. The
// durable fingerprint for the already-validated pinned key is bound to the
// returned PeerConn so later explicit trust checkpoints can revalidate it.
func (f SecurePeerFactory) DialSecurePeer(ctx context.Context, address string, timeout time.Duration, remoteDeviceID string) (*transport.PeerConn, error) {
	local, remote, err := f.ResolveIdentities(remoteDeviceID)
	if err != nil {
		return nil, err
	}
	peer, err := transport.DialSecurePeer(ctx, address, timeout, local, remote)
	if err != nil {
		return nil, err
	}
	if err := peer.BindAuthenticatedKeyFingerprint(identity.Fingerprint(remote.PublicKey)); err != nil {
		_ = peer.Close()
		return nil, fmt.Errorf("bind secure peer trust identity: %w", err)
	}
	return peer, nil
}

// AcceptSecurePeer takes ownership of conn. It resolves current identity/trust
// state before TLS admission and closes the accepted stream if resolution or
// durable trust-identity binding fails.
func (f SecurePeerFactory) AcceptSecurePeer(ctx context.Context, conn net.Conn, timeout time.Duration, remoteDeviceID string) (*transport.PeerConn, error) {
	local, remote, err := f.ResolveIdentities(remoteDeviceID)
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, err
	}
	peer, err := transport.AcceptSecurePeer(ctx, conn, timeout, local, remote)
	if err != nil {
		return nil, err
	}
	if err := peer.BindAuthenticatedKeyFingerprint(identity.Fingerprint(remote.PublicKey)); err != nil {
		_ = peer.Close()
		return nil, fmt.Errorf("bind secure peer trust identity: %w", err)
	}
	return peer, nil
}

// RevalidatePeer checks whether the exact account-scoped device ID and key
// fingerprint admitted for this secure session are still trusted now. A revoked,
// replaced, missing, malformed, or unreadable trust record fails closed and the
// local connection is closed before the error is returned.
//
// This is an explicit lifecycle/operation checkpoint; it does not imply that a
// session is asynchronously terminated at the instant trust changes. Runtime
// orchestration must invoke it at the boundaries where current authorization is
// required and before reusing long-lived sessions.
func (f SecurePeerFactory) RevalidatePeer(peer *transport.PeerConn) error {
	if f.AccountID == "" || f.TrustedDevices == nil {
		if peer != nil {
			_ = peer.Close()
		}
		return ErrSecurePeerFactoryUnavailable
	}
	if peer == nil {
		return ErrSecurePeerTrustNotCurrent
	}
	deviceID := peer.AuthenticatedDeviceID()
	fingerprint := peer.AuthenticatedKeyFingerprint()
	if deviceID == "" || fingerprint == "" {
		_ = peer.Close()
		return ErrSecurePeerTrustNotCurrent
	}
	trusted, err := f.TrustedDevices.IsTrusted(f.AccountID, deviceID, fingerprint)
	if err != nil {
		_ = peer.Close()
		return fmt.Errorf("revalidate secure peer trust: %w", err)
	}
	if !trusted {
		_ = peer.Close()
		return ErrSecurePeerTrustNotCurrent
	}
	return nil
}

// RunWithCurrentTrust is the operation-level guard for reusing an established
// secure peer. It revalidates the exact account/device/fingerprint immediately
// before invoking operation. If trust is not current, operation is never called
// and RevalidatePeer closes the local peer fail-closed.
//
// This helper makes explicit operation checkpoints reusable; it is not a
// background revocation scheduler and cannot make trust changes instantaneous
// between the checkpoint and an operation already in progress.
func (f SecurePeerFactory) RunWithCurrentTrust(peer *transport.PeerConn, operation func(*transport.PeerConn) error) error {
	if operation == nil {
		return ErrSecurePeerOperationRequired
	}
	if err := f.RevalidatePeer(peer); err != nil {
		return err
	}
	return operation(peer)
}

// RunOperationSequenceWithCurrentTrust executes a bounded sequence of peer
// operations and revalidates durable current trust before every individual step.
// All operation slots are validated before the first callback is invoked so a
// malformed sequence cannot partially execute. If trust changes between steps,
// the next checkpoint fails closed and the remaining operations are not run.
//
// This remains synchronous explicit orchestration. It does not claim background
// or instantaneous revocation, and it cannot revoke an operation already in
// progress after that operation's pre-execution checkpoint.
func (f SecurePeerFactory) RunOperationSequenceWithCurrentTrust(peer *transport.PeerConn, operations []func(*transport.PeerConn) error) error {
	if len(operations) == 0 || len(operations) > maxSecurePeerOperationSequence {
		return ErrSecurePeerOperationSequence
	}
	for _, operation := range operations {
		if operation == nil {
			return ErrSecurePeerOperationRequired
		}
	}
	for _, operation := range operations {
		if err := f.RunWithCurrentTrust(peer, operation); err != nil {
			return err
		}
	}
	return nil
}
