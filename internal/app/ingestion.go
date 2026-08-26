package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/identity"
	"github.com/GoreeCloud/goreecloud-sync/internal/replication"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

const maxIngestBodyBytes = 256 << 10

var ErrPeerResolutionFailed = errors.New("authenticated sync peer could not be resolved")

// PeerResolver is supplied by the authenticated transport/session layer. The
// HTTP request body never carries authoritative peer identity or trust state.
type PeerResolver interface {
	ResolvePeer(context.Context, *http.Request) (session.AuthenticatedPeer, error)
}

type ingestRequest struct {
	Record datasets.RecordEnvelope `json:"record"`
	Proof  identity.RecordProof    `json:"proof"`
}

type ingestResponse struct {
	Accepted bool                           `json:"accepted"`
	Evidence any                            `json:"evidence"`
	Receipt  replication.ObservationReceipt `json:"receipt"`
}

func (s *Server) handleSearchHistoryIngest(w http.ResponseWriter, r *http.Request) {
	if s.ingestor == nil || s.peerResolver == nil {
		http.Error(w, "sync ingestion unavailable", http.StatusServiceUnavailable)
		return
	}
	peer, err := s.peerResolver.ResolvePeer(r.Context(), r)
	if err != nil {
		http.Error(w, ErrPeerResolutionFailed.Error(), http.StatusUnauthorized)
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxIngestBodyBytes)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request ingestRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid ingestion request", http.StatusBadRequest)
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		http.Error(w, "invalid ingestion request", http.StatusBadRequest)
		return
	}

	evidence, receipt, err := s.ingestor.Ingest(r.Context(), peer, request.Record, request.Proof)
	if err != nil {
		s.logger.Warn("sync record ingestion rejected", "dataset", request.Record.Dataset, "record_id", request.Record.RecordID, "error", err)
		http.Error(w, "record rejected", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(ingestResponse{Accepted: true, Evidence: evidence, Receipt: receipt})
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
