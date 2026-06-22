package core_http_middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_response "go-chat/internal/core/transport/http/response"
)

type contextKey string

const (
	userIDContextKey   contextKey = "userID"
	usernameContextKey contextKey = "username"
)

// TokenValidator — интерфейс валидации access token
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string) (userID string, username string, err error)
}

// Auth — middleware для защищённых роутов.
// Читает "Authorization: Bearer <token>", кладёт userID и username в context.
func Auth(validator TokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := core_logger.FromContext(ctx)
			responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				responseHandler.ErrorResponse(
					fmt.Errorf("missing authorization header: %w", core_error.ErrUnauthorized),
					"unauthorized",
				)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				responseHandler.ErrorResponse(
					fmt.Errorf("malformed authorization header: %w", core_error.ErrUnauthorized),
					"unauthorized",
				)
				return
			}

			userID, username, err := validator.ValidateAccessToken(ctx, parts[1])
			if err != nil {
				responseHandler.ErrorResponse(err, "unauthorized")
				return
			}

			ctx = context.WithValue(ctx, userIDContextKey, userID)
			ctx = context.WithValue(ctx, usernameContextKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("userID not found in context: %w", core_error.ErrUnauthorized)
	}
	return userID, nil
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameContextKey).(string)
	return username, ok && username != ""
}
