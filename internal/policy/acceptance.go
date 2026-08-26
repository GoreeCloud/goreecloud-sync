package policy

import (
	"errors"
	"strings"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var (
	ErrPurposeDenied       = errors.New("privacy purpose is not authorized")
	ErrConsentDenied       = errors.New("privacy consent is not authorized")
	ErrPeerUntrusted       = errors.New("peer trust evidence is insufficient")
	ErrDatasetUnauthorized = errors.New("dataset is not negotiated for peer")
)

// PrivacyDecision is the Privacy Shield decision presented to Sync for a
// single record acceptance attempt. Sync does not infer consent or purpose.
type PrivacyDecision struct {
	Purpose        string    `json:"purpose"`
	PurposeAllowed bool      `json:"purposeAllowed"`
	ConsentGranted bool      `json:"consentGranted"`
	DecidedAt      time.Time `json:"decidedAt"`
	EvidenceID     string    `json:"evidenceId,omitempty"`
}

// TrustEvidence is the Wardveil Security decision for the authenticated peer.
// The cryptographic identity comes from the pairing/session layer; Wardveil
// supplies the higher-level trust authorization used for record acceptance.
type TrustEvidence struct {
	DeviceID       string    `json:"deviceId"`
	KeyFingerprint string    `json:"keyFingerprint"`
	Trusted        bool      `json:"trusted"`
	EvaluatedAt    time.Time `json:"evaluatedAt"`
	EvidenceID     string    `json:"evidenceId,omitempty"`
}

// AcceptanceEvidence records why a record was accepted without embedding the
// synchronized application payload itself.
type AcceptanceEvidence struct {
	Dataset           string    `json:"dataset"`
	RecordID          string    `json:"recordId"`
	PeerDeviceID      string    `json:"peerDeviceId"`
	PeerFingerprint   string    `json:"peerFingerprint"`
	PrivacyEvidenceID string    `json:"privacyEvidenceId,omitempty"`
	TrustEvidenceID   string    `json:"trustEvidenceId,omitempty"`
	AcceptedAt        time.Time `json:"acceptedAt"`
}

// AuthorizeRecord fails closed unless the dataset was negotiated, Privacy
// Shield explicitly authorized the purpose/consent, and Wardveil explicitly
// trusted the authenticated peer identity.
func AuthorizeRecord(record datasets.RecordEnvelope, peer session.AuthenticatedPeer, privacy PrivacyDecision, trust TrustEvidence, now time.Time) (AcceptanceEvidence, error) {
	var capability *datasets.Capability
	for i := range peer.NegotiatedDatasets {
		if peer.NegotiatedDatasets[i].Dataset == record.Dataset {
			capability = &peer.NegotiatedDatasets[i]
			break
		}
	}
	if capability == nil {
		return AcceptanceEvidence{}, ErrDatasetUnauthorized
	}
	if err := datasets.ValidateRecord(record, *capability); err != nil {
		return AcceptanceEvidence{}, err
	}
	if strings.TrimSpace(privacy.Purpose) == "" || !privacy.PurposeAllowed {
		return AcceptanceEvidence{}, ErrPurposeDenied
	}
	if !privacy.ConsentGranted || privacy.DecidedAt.IsZero() {
		return AcceptanceEvidence{}, ErrConsentDenied
	}
	if !trust.Trusted || trust.EvaluatedAt.IsZero() || trust.DeviceID != peer.DeviceID || trust.KeyFingerprint != peer.KeyFingerprint {
		return AcceptanceEvidence{}, ErrPeerUntrusted
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return AcceptanceEvidence{
		Dataset:           record.Dataset,
		RecordID:          record.RecordID,
		PeerDeviceID:      peer.DeviceID,
		PeerFingerprint:   peer.KeyFingerprint,
		PrivacyEvidenceID: privacy.EvidenceID,
		TrustEvidenceID:   trust.EvidenceID,
		AcceptedAt:        now.UTC(),
	}, nil
}
