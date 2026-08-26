package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/replication"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

type retrievalPeerResolver struct {
	peer session.AuthenticatedPeer
	err  error
}

func (r retrievalPeerResolver) ResolvePeer(context.Context, *http.Request) (session.AuthenticatedPeer, error) {
	return r.peer, r.err
}

func TestDatasetRetrievalRequiresNegotiatedRead(t *testing.T) {
	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))}
	resolver := retrievalPeerResolver{peer: session.AuthenticatedPeer{
		DeviceID: "device-1",
		NegotiatedDatasets: []datasets.Capability{{
			Dataset: "search.history", Application: "search", SchemaVersion: 1, Write: true,
		}},
	}}
	server := NewServerWithOptions("127.0.0.1:0", nil, ServerOptions{Ingestor: ingestor, PeerResolver: resolver})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestDatasetRetrievalReturnsOnlyNegotiatedDatasetRecords(t *testing.T) {
	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))}
	resolver := retrievalPeerResolver{peer: session.AuthenticatedPeer{
		DeviceID: "device-1",
		NegotiatedDatasets: []datasets.Capability{{
			Dataset: "search.history", Application: "search", SchemaVersion: 1, Read: true,
		}},
	}}
	server := NewServerWithOptions("127.0.0.1:0", nil, ServerOptions{Ingestor: ingestor, PeerResolver: resolver})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if body := response.Body.String(); body != "{\"dataset\":\"search.history\",\"count\":0,\"records\":[]}\n" {
		t.Fatalf("unexpected response: %s", body)
	}
}

func TestDatasetRetrievalRequiresAuthenticatedPeer(t *testing.T) {
	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))}
	resolver := retrievalPeerResolver{err: errors.New("no session")}
	server := NewServerWithOptions("127.0.0.1:0", nil, ServerOptions{Ingestor: ingestor, PeerResolver: resolver})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}
