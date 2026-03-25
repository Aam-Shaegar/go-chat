package handler

import (
	"encoding/json"
	"go-chat/internal/domain"
	"go-chat/internal/service"
	"net/http"
	"time"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Username == "" || input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email and password are required")
		return
	}
	if len(input.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters length")
		return
	}
	resp, refreshToken, err := h.authService.Register(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	setRefreshCookie(w, refreshToken)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	resp, refreshToken, err := h.authService.Login(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	setRefreshCookie(w, refreshToken)
	writeJSON(w, http.StatusOK, resp)
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/auth/refresh",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}
