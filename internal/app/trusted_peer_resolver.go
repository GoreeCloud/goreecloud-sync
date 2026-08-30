package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrPeerNotTrusted = errors.New("authenticated sync peer is not trusted for this account")

type TrustedDeviceChecker interface {
	IsTrusted(accountID, deviceID, fingerprint string) (bool, error)
}

// TrustedPeerResolver composes transport/session authentication with explicit
// account-scoped device authorization. A valid bearer session is insufficient
// when the durable trust registry does not authorize the exact device key.
type TrustedPeerResolver struct {
	Inner     PeerResolver
	AccountID string
	Trust     TrustedDeviceChecker
}

func (r TrustedPeerResolver) ResolvePeer(ctx context.Context, request *http.Request) (session.AuthenticatedPeer, error) {
	if r.Inner == nil || r.Trust == nil || r.AccountID == "" {
		return session.AuthenticatedPeer{}, ErrPeerResolutionFailed
	}
	peer, err := r.Inner.ResolvePeer(ctx, request)
	if err != nil {
		return session.AuthenticatedPeer{}, err
	}
	trusted, err := r.Trust.IsTrusted(r.AccountID, peer.DeviceID, peer.KeyFingerprint)
	if err != nil {
		return session.AuthenticatedPeer{}, ErrPeerResolutionFailed
	}
	if !trusted {
		return session.AuthenticatedPeer{}, ErrPeerNotTrusted
	}
	return peer, nil
}
