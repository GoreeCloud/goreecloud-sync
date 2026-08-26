package session

import (
	"errors"
	"strings"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
)

var (
	ErrUnauthenticatedPeer = errors.New("sync peer is not authenticated")
	ErrDeviceMismatch      = errors.New("pairing proof device does not match session device")
)

// AuthenticatedPeer is the trust-bearing session descriptor created only after
// a pairing proof has verified key possession. Authorization remains a policy
// decision for the higher GoreeCloud trust layer.
type AuthenticatedPeer struct {
	DeviceID           string                `json:"deviceId"`
	KeyFingerprint     string                `json:"keyFingerprint"`
	Capabilities       []datasets.Capability `json:"capabilities"`
	NegotiatedDatasets []datasets.Capability `json:"negotiatedDatasets"`
}

// AuthenticatePeer verifies the signed pairing proof and negotiates only the
// datasets mutually supported by the local and remote peers.
func AuthenticatePeer(deviceID string, proof identity.PairingProof, local, remote []datasets.Capability) (AuthenticatedPeer, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || proof.DeviceID != deviceID {
		return AuthenticatedPeer{}, ErrDeviceMismatch
	}
	fingerprint, err := proof.Verify()
	if err != nil {
		return AuthenticatedPeer{}, ErrUnauthenticatedPeer
	}
	remoteCopy := append([]datasets.Capability(nil), remote...)
	return AuthenticatedPeer{
		DeviceID:           deviceID,
		KeyFingerprint:     fingerprint,
		Capabilities:       remoteCopy,
		NegotiatedDatasets: datasets.Negotiate(local, remoteCopy),
	}, nil
}
