package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var (
	ErrDatasetReadNotNegotiated = errors.New("dataset read capability was not negotiated")
	ErrDatasetReadUnavailable   = errors.New("sync dataset retrieval unavailable")
)

type retrievalResponse struct {
	Dataset string                    `json:"dataset"`
	Count   int                       `json:"count"`
	Records []datasets.RecordEnvelope `json:"records"`
}

func (s *Server) handleSearchHistoryRetrieve(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetRetrieve(w, r, "search.history")
}

func (s *Server) handleBookmarkItemRetrieve(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetRetrieve(w, r, "bookmarks.items")
}

func (s *Server) handleBrowserTabRetrieve(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetRetrieve(w, r, "browser.tabs")
}

func (s *Server) handleBrowserHistoryRetrieve(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetRetrieve(w, r, "browser.history")
}

func (s *Server) handleDatasetRetrieve(w http.ResponseWriter, r *http.Request, dataset string) {
	if s.ingestor == nil || s.peerResolver == nil {
		http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
		return
	}
	peer, err := s.peerResolver.ResolvePeer(r.Context(), r)
	if err != nil {
		http.Error(w, ErrPeerResolutionFailed.Error(), http.StatusUnauthorized)
		return
	}
	readCapability, ok := peerReadCapability(peer, dataset)
	if !ok {
		http.Error(w, ErrDatasetReadNotNegotiated.Error(), http.StatusForbidden)
		return
	}

	var records []datasets.RecordEnvelope
	switch dataset {
	case "search.history":
		if s.ingestor.History == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		records, err = s.ingestor.History.Records()
	case "bookmarks.items":
		if s.ingestor.Bookmarks == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		records, err = s.ingestor.Bookmarks.Records()
	case "browser.tabs":
		if s.ingestor.BrowserTabs == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		records, err = s.ingestor.BrowserTabs.Records()
	case "browser.history":
		if s.ingestor.BrowserHistory == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		records, err = s.ingestor.BrowserHistory.Records()
	default:
		http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("sync dataset retrieval failed", "dataset", dataset, "error", err)
		http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []datasets.RecordEnvelope{}
	}

	// Reads are validated again at the storage boundary before serialization.
	// This prevents stale, corrupted, or manually altered persisted state from
	// bypassing schema negotiation or Privacy Shield tombstone minimization.
	validationCapability := readCapability
	validationCapability.Write = true
	validationCapability.Delete = true
	for _, record := range records {
		if err := datasets.ValidateRecord(record, validationCapability); err != nil {
			s.logger.Error("sync dataset retrieval record validation failed", "dataset", dataset, "record_id", record.RecordID, "error", err)
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(retrievalResponse{Dataset: dataset, Count: len(records), Records: records})
}

func peerReadCapability(peer session.AuthenticatedPeer, dataset string) (datasets.Capability, bool) {
	for _, capability := range peer.NegotiatedDatasets {
		if capability.Dataset == dataset && capability.Read {
			return capability, true
		}
	}
	return datasets.Capability{}, false
}
