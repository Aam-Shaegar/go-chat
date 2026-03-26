package handler

import (
	"context"
	"go-chat/internal/service"
	"net/http"
	"strings"
)

type contextKey string

const ContextUserID contextKey = "user_id"

func AuthMiddleware(tokenService *service.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			claims, err := tokenService.Parse(parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
