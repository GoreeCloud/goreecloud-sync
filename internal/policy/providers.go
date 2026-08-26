package policy

import (
	"context"
	"errors"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrDecisionProviderUnavailable = errors.New("required policy decision provider is unavailable")

// PrivacyProvider obtains a Privacy Shield decision for the exact record and
// authenticated peer. Callers cannot self-assert consent or purpose approval.
type PrivacyProvider interface {
	EvaluatePrivacy(context.Context, datasets.RecordEnvelope, session.AuthenticatedPeer) (PrivacyDecision, error)
}

// TrustProvider obtains Wardveil Security trust evidence for the authenticated
// peer and record acceptance attempt.
type TrustProvider interface {
	EvaluateTrust(context.Context, datasets.RecordEnvelope, session.AuthenticatedPeer) (TrustEvidence, error)
}

// DecisionProviders groups the two platform policy dependencies required by
// synchronized record ingestion.
type DecisionProviders struct {
	Privacy PrivacyProvider
	Trust   TrustProvider
}

func (p DecisionProviders) Validate() error {
	if p.Privacy == nil || p.Trust == nil {
		return ErrDecisionProviderUnavailable
	}
	return nil
}
