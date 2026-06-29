package jwt_transport_http

import (
	"errors"
	"net/http"
	"time"

	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_response "go-chat/internal/core/transport/http/response"
)

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *JwtHTTPHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			err = core_error.ErrUnauthorized
		}
		responseHandler.ErrorResponse(err, "failed to get refresh token")
		return
	}

	accessToken, newRefreshToken, err := h.jwtService.Refresh(ctx, cookie.Value)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to refresh token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		HttpOnly: true,
		Secure:   h.secureRefreshCookie,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(h.refreshTTL),
	})

	responseHandler.JSONResponse(RefreshResponse{AccessToken: accessToken}, http.StatusOK)
}
