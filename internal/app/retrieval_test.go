package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDatasetRetrievalPaginatesByRecordID(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "history.json")
	persisted := "{\"query-3\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":1,\"recordId\":\"query-3\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:03:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"three\"}}},\"query-1\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":1,\"recordId\":\"query-1\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:01:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"one\"}}},\"query-2\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":1,\"recordId\":\"query-2\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:02:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"two\"}}}}"
	if err := os.WriteFile(historyPath, []byte(persisted), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(historyPath)}
	resolver := retrievalPeerResolver{peer: session.AuthenticatedPeer{
		DeviceID: "device-1",
		NegotiatedDatasets: []datasets.Capability{{
			Dataset: "search.history", Application: "search", SchemaVersion: 1, Read: true,
		}},
	}}
	server := NewServerWithOptions("127.0.0.1:0", nil, ServerOptions{Ingestor: ingestor, PeerResolver: resolver})

	firstRequest := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history?limit=2", nil)
	first := httptest.NewRecorder()
	server.Handler().ServeHTTP(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d; body=%s", first.Code, first.Body.String())
	}
	firstBody := first.Body.String()
	if !strings.Contains(firstBody, `"count":2`) || !strings.Contains(firstBody, `"nextAfter":"query-2"`) || !strings.Contains(firstBody, `"recordId":"query-1"`) || !strings.Contains(firstBody, `"recordId":"query-2"`) {
		t.Fatalf("unexpected first page: %s", firstBody)
	}
	if strings.Index(firstBody, `"recordId":"query-1"`) > strings.Index(firstBody, `"recordId":"query-2"`) {
		t.Fatalf("first page is not record-id ordered: %s", firstBody)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history?limit=2&after=query-2", nil)
	second := httptest.NewRecorder()
	server.Handler().ServeHTTP(second, secondRequest)
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d; body=%s", second.Code, second.Body.String())
	}
	if body := second.Body.String(); !strings.Contains(body, `"count":1`) || !strings.Contains(body, `"recordId":"query-3"`) || strings.Contains(body, "nextAfter") {
		t.Fatalf("unexpected second page: %s", body)
	}
}

func TestDatasetRetrievalRejectsInvalidPageParameters(t *testing.T) {
	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(filepath.Join(t.TempDir(), "history.json"))}
	resolver := retrievalPeerResolver{peer: session.AuthenticatedPeer{
		DeviceID: "device-1",
		NegotiatedDatasets: []datasets.Capability{{
			Dataset: "search.history", Application: "search", SchemaVersion: 1, Read: true,
		}},
	}}
	server := NewServerWithOptions("127.0.0.1:0", nil, ServerOptions{Ingestor: ingestor, PeerResolver: resolver})

	for _, target := range []string{
		"/api/v1/sync/search/history?limit=0",
		"/api/v1/sync/search/history?limit=1025",
		"/api/v1/sync/search/history?after=" + strings.Repeat("x", datasets.MaxRecordIDBytes+1),
	} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusBadRequest)
		}
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

func TestDatasetRetrievalRejectsStoredSchemaOutsideNegotiation(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "history.json")
	persisted := "{\"query-1\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":2,\"recordId\":\"query-1\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:00:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"goreecloud\"}}}}"
	if err := os.WriteFile(historyPath, []byte(persisted), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ingestor := &replication.Ingestor{History: replication.NewSearchHistoryStore(historyPath)}
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

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}
