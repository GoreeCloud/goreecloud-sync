package app

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

type peerResolverFunc func(context.Context, *http.Request) (session.AuthenticatedPeer, error)

func (f peerResolverFunc) ResolvePeer(ctx context.Context, request *http.Request) (session.AuthenticatedPeer, error) {
	return f(ctx, request)
}

type trustCheckerFunc func(string, string, string) (bool, error)

func (f trustCheckerFunc) IsTrusted(accountID, deviceID, fingerprint string) (bool, error) {
	return f(accountID, deviceID, fingerprint)
}

func TestTrustedPeerResolverRequiresExactTrustedDeviceKey(t *testing.T) {
	peer := session.AuthenticatedPeer{DeviceID: "11111111-1111-1111-1111-111111111111", KeyFingerprint: "fingerprint"}
	resolver := TrustedPeerResolver{
		Inner:     peerResolverFunc(func(context.Context, *http.Request) (session.AuthenticatedPeer, error) { return peer, nil }),
		AccountID: "account-a",
		Trust: trustCheckerFunc(func(accountID, deviceID, fingerprint string) (bool, error) {
			return accountID == "account-a" && deviceID == peer.DeviceID && fingerprint == peer.KeyFingerprint, nil
		}),
	}
	resolved, err := resolver.ResolvePeer(context.Background(), &http.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DeviceID != peer.DeviceID || resolved.KeyFingerprint != peer.KeyFingerprint {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestTrustedPeerResolverRejectsRevokedOrUntrustedPeer(t *testing.T) {
	peer := session.AuthenticatedPeer{DeviceID: "11111111-1111-1111-1111-111111111111", KeyFingerprint: "fingerprint"}
	resolver := TrustedPeerResolver{
		Inner:     peerResolverFunc(func(context.Context, *http.Request) (session.AuthenticatedPeer, error) { return peer, nil }),
		AccountID: "account-a",
		Trust:     trustCheckerFunc(func(string, string, string) (bool, error) { return false, nil }),
	}
	if _, err := resolver.ResolvePeer(context.Background(), &http.Request{}); !errors.Is(err, ErrPeerNotTrusted) {
		t.Fatalf("err=%v", err)
	}
}

func TestTrustedPeerResolverFailsClosedOnTrustStoreError(t *testing.T) {
	peer := session.AuthenticatedPeer{DeviceID: "11111111-1111-1111-1111-111111111111", KeyFingerprint: "fingerprint"}
	resolver := TrustedPeerResolver{
		Inner:     peerResolverFunc(func(context.Context, *http.Request) (session.AuthenticatedPeer, error) { return peer, nil }),
		AccountID: "account-a",
		Trust:     trustCheckerFunc(func(string, string, string) (bool, error) { return false, errors.New("store unavailable") }),
	}
	if _, err := resolver.ResolvePeer(context.Background(), &http.Request{}); !errors.Is(err, ErrPeerResolutionFailed) {
		t.Fatalf("err=%v", err)
	}
}
