package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var (
	ErrDatasetReadNotNegotiated = errors.New("dataset read capability was not negotiated")
	ErrDatasetReadUnavailable   = errors.New("sync dataset retrieval unavailable")
	ErrInvalidRetrievalPage     = errors.New("invalid sync retrieval page")
)

const (
	defaultRetrievalLimit = 256
	maxRetrievalLimit     = 1024
)

type retrievalResponse struct {
	Dataset   string                    `json:"dataset"`
	Count     int                       `json:"count"`
	Records   []datasets.RecordEnvelope `json:"records"`
	NextAfter string                    `json:"nextAfter,omitempty"`
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

	limit, after, err := retrievalPage(r)
	if err != nil {
		http.Error(w, ErrInvalidRetrievalPage.Error(), http.StatusBadRequest)
		return
	}

	// Retrieval must validate the complete persisted dataset even though only a
	// bounded page is retained. Corruption outside the visible page therefore
	// remains fail-closed without materializing every record in memory.
	validationCapability := readCapability
	validationCapability.Write = true
	validationCapability.Delete = true

	var page []datasets.RecordEnvelope
	var nextAfter string
	switch dataset {
	case "search.history":
		if s.ingestor.History == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		page, nextAfter, err = s.ingestor.History.ValidatedPage(validationCapability, after, limit)
	case "bookmarks.items":
		if s.ingestor.Bookmarks == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		page, nextAfter, err = s.ingestor.Bookmarks.ValidatedPage(validationCapability, after, limit)
	case "browser.tabs":
		if s.ingestor.BrowserTabs == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		page, nextAfter, err = s.ingestor.BrowserTabs.ValidatedPage(validationCapability, after, limit)
	case "browser.history":
		if s.ingestor.BrowserHistory == nil {
			http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusServiceUnavailable)
			return
		}
		page, nextAfter, err = s.ingestor.BrowserHistory.ValidatedPage(validationCapability, after, limit)
	default:
		http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("sync dataset retrieval failed", "dataset", dataset, "error", err)
		http.Error(w, ErrDatasetReadUnavailable.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		page = []datasets.RecordEnvelope{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(retrievalResponse{
		Dataset: dataset, Count: len(page), Records: page, NextAfter: nextAfter,
	})
}

func retrievalPage(r *http.Request) (int, string, error) {
	limit := defaultRetrievalLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxRetrievalLimit {
			return 0, "", ErrInvalidRetrievalPage
		}
		limit = parsed
	}
	after := r.URL.Query().Get("after")
	if len(after) > datasets.MaxRecordIDBytes {
		return 0, "", ErrInvalidRetrievalPage
	}
	return limit, after, nil
}

func peerReadCapability(peer session.AuthenticatedPeer, dataset string) (datasets.Capability, bool) {
	for _, capability := range peer.NegotiatedDatasets {
		if capability.Dataset == dataset && capability.Read {
			return capability, true
		}
	}
	return datasets.Capability{}, false
}
