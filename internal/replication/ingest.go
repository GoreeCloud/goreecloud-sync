package replication

import (
	"context"
	"errors"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/policy"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrUnsupportedIngestDataset = errors.New("unsupported ingestion dataset")

// Ingestor is the trusted boundary between authenticated transport and durable
// replication. Privacy Shield and Wardveil decisions are obtained internally.
type Ingestor struct {
	Providers      policy.DecisionProviders
	History        *SearchHistoryStore
	Bookmarks      *BookmarkItemStore
	BrowserTabs    *BrowserRecordStore
	BrowserHistory *BrowserRecordStore
	Receipts       *ReceiptStore
	Replay         *ReplayGuard
	Now            func() time.Time
}

func (i *Ingestor) Ingest(ctx context.Context, peer session.AuthenticatedPeer, record datasets.RecordEnvelope, proof identity.RecordProof) (policy.AcceptanceEvidence, ObservationReceipt, error) {
	if err := i.Providers.Validate(); err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}
	if err := session.VerifySignedRecord(peer, record, proof); err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}
	if err := i.Replay.Accept(peer, record); err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}

	privacy, err := i.Providers.Privacy.EvaluatePrivacy(ctx, record, peer)
	if err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}
	trust, err := i.Providers.Trust.EvaluateTrust(ctx, record, peer)
	if err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}

	now := time.Now().UTC()
	if i.Now != nil {
		now = i.Now().UTC()
	}

	var evidence policy.AcceptanceEvidence
	switch record.Dataset {
	case "search.history":
		if i.History == nil {
			return policy.AcceptanceEvidence{}, ObservationReceipt{}, ErrUnsupportedIngestDataset
		}
		evidence, err = i.History.AcceptAndPersist(record, peer, privacy, trust, now)
	case "bookmarks.items":
		if i.Bookmarks == nil {
			return policy.AcceptanceEvidence{}, ObservationReceipt{}, ErrUnsupportedIngestDataset
		}
		evidence, err = i.Bookmarks.AcceptAndPersist(record, peer, privacy, trust, now)
	case "browser.tabs":
		if i.BrowserTabs == nil {
			return policy.AcceptanceEvidence{}, ObservationReceipt{}, ErrUnsupportedIngestDataset
		}
		evidence, err = i.BrowserTabs.AcceptAndPersist(record, peer, privacy, trust, now)
	case "browser.history":
		if i.BrowserHistory == nil {
			return policy.AcceptanceEvidence{}, ObservationReceipt{}, ErrUnsupportedIngestDataset
		}
		evidence, err = i.BrowserHistory.AcceptAndPersist(record, peer, privacy, trust, now)
	default:
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, ErrUnsupportedIngestDataset
	}
	if err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}

	receipt, err := NewObservationReceipt(record, peer, now)
	if err != nil {
		return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
	}
	if i.Receipts != nil {
		if err := i.Receipts.Append(receipt); err != nil {
			return policy.AcceptanceEvidence{}, ObservationReceipt{}, err
		}
	}
	return evidence, receipt, nil
}
