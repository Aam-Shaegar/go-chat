package jwt_transport_http

import (
	"context"
	"time"

	core_http_server "go-chat/internal/core/transport/http/server"
	"net/http"
)

type JwtHTTPHandler struct {
	jwtService          JwtService
	refreshTTL          time.Duration
	secureRefreshCookie bool
}

type JwtService interface {
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
}

func NewJwtHTTPHandler(jwtService JwtService, refreshTTL time.Duration, secureRefreshCookie bool) *JwtHTTPHandler {
	return &JwtHTTPHandler{
		jwtService:          jwtService,
		refreshTTL:          refreshTTL,
		secureRefreshCookie: secureRefreshCookie,
	}
}

func (h *JwtHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodPost,
			Path:    "/jwt/refresh",
			Handler: h.Refresh,
		},
	}
}
