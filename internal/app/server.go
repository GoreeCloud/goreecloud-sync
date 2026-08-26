package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/replication"
	"github.com/GoreeCloud/goreecloud-sync/internal/version"
)

const defaultShutdownTimeout = 10 * time.Second

type ServerOptions struct {
	Ingestor     *replication.Ingestor
	PeerResolver PeerResolver
}

type Server struct {
	httpServer   *http.Server
	logger       *slog.Logger
	startedAt    time.Time
	ingestor     *replication.Ingestor
	peerResolver PeerResolver
}

type statusResponse struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	State     string `json:"state"`
	StartedAt string `json:"started_at"`
}

func NewServer(addr string, logger *slog.Logger) *Server {
	return NewServerWithOptions(addr, logger, ServerOptions{})
}

func NewServerWithOptions(addr string, logger *slog.Logger, options ServerOptions) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		logger:       logger,
		startedAt:    time.Now().UTC(),
		ingestor:     options.Ingestor,
		peerResolver: options.PeerResolver,
	}

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	if s.ingestor != nil && s.peerResolver != nil {
		mux.HandleFunc("POST /api/v1/sync/search/history", s.handleSearchHistoryIngest)
	}
	return securityHeaders(mux)
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("GoreeCloud Sync service starting", "address", s.httpServer.Addr)
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(statusResponse{
		Name:      "GoreeCloud Sync",
		Version:   version.Version,
		Commit:    version.Commit,
		State:     "development",
		StartedAt: s.startedAt.Format(time.RFC3339),
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
