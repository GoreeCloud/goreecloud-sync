package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

var ErrInvalidAuthorizationHeader = errors.New("invalid sync authorization header")

type SessionPeerResolver struct {
	Sessions *session.Store
	Now      func() time.Time
}

func (r SessionPeerResolver) ResolvePeer(_ context.Context, request *http.Request) (session.AuthenticatedPeer, error) {
	if r.Sessions == nil || request == nil {
		return session.AuthenticatedPeer{}, ErrPeerResolutionFailed
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return session.AuthenticatedPeer{}, ErrInvalidAuthorizationHeader
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if token == "" {
		return session.AuthenticatedPeer{}, ErrInvalidAuthorizationHeader
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	return r.Sessions.Resolve(token, now)
}
