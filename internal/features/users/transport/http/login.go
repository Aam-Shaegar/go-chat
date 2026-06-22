package users_transport_http

import (
	domain_dtos "go-chat/internal/core/domain/dtos"
	core_logger "go-chat/internal/core/logger"
	core_http_request "go-chat/internal/core/transport/http/request"
	core_http_response "go-chat/internal/core/transport/http/response"
	"net/http"
	"time"
)

type LoginRequest domain_dtos.LoginInputDTO

type LoginResponse domain_dtos.AuthResponseDTO

func (h *UsersHTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

	var req LoginRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}
	authResp, refreshToken, err := h.usersService.Login(ctx, req.Email, req.Password)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to login")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(h.cfg.JwtRefreshTTL),
	})
	response := LoginResponse(authResp)
	responseHandler.JSONResponse(response, http.StatusOK)
}
