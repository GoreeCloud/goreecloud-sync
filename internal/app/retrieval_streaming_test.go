package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/replication"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func TestDatasetRetrievalRejectsCorruptionAfterRequestedPage(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "history.json")
	persisted := "{\"query-1\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":1,\"recordId\":\"query-1\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:01:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"one\"}}},\"query-z\":{\"envelope\":{\"dataset\":\"search.history\",\"schemaVersion\":2,\"recordId\":\"query-z\",\"revision\":1,\"updatedAt\":\"2026-08-28T19:02:00Z\",\"originDevice\":\"device-1\",\"deleted\":false,\"payload\":{\"query\":\"corrupt\"}}}}"
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

	request := httptest.NewRequest(http.MethodGet, "/api/v1/sync/search/history?limit=1", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}
