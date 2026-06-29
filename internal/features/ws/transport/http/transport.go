package ws_transport_http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_response "go-chat/internal/core/transport/http/response"
	core_http_server "go-chat/internal/core/transport/http/server"
	ws_client "go-chat/internal/features/ws/client"
	ws_domain "go-chat/internal/features/ws/domain"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSHandler struct {
	service        WSService
	hub            Hub
	roomRepo       RoomRepository
	tokenValidator TokenValidator
	allowedOrigins []string
}

type WSService interface {
	Handle(client *ws_client.Client, event ws_domain.IncomingEvent)
	OnClose(client *ws_client.Client)
}

type Hub interface {
	Register(client *ws_client.Client)
}

type RoomRepository interface {
	IsMember(ctx context.Context, roomID, userID string) (bool, error)
}

type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string) (string, string, error)
}

func NewWSHandler(service WSService, hub Hub, roomRepo RoomRepository, tokenValidator TokenValidator, allowedOrigins []string) *WSHandler {
	return &WSHandler{
		service:        service,
		hub:            hub,
		roomRepo:       roomRepo,
		tokenValidator: tokenValidator,
		allowedOrigins: allowedOrigins,
	}
}

func (h *WSHandler) Routes(authMiddleware core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		{
			// Глобальный сокет пользователя: один канал для всех чатов и уведомлений.
			Method:  http.MethodGet,
			Path:    "/ws",
			Handler: h.ServeUserWS,
		},
		{
			// Без authMiddleware — валидируем токен сами из query param
			Method:  http.MethodGet,
			Path:    "/ws/rooms/{roomId}",
			Handler: h.ServeRoomWS,
		},
	}
}

func (h *WSHandler) ServeUserWS(w http.ResponseWriter, r *http.Request) {
	h.serveWS(w, r, "")
}

func (h *WSHandler) ServeRoomWS(w http.ResponseWriter, r *http.Request) {
	h.serveWS(w, r, r.PathValue("roomId"))
}

func (h *WSHandler) serveWS(w http.ResponseWriter, r *http.Request, roomID string) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

	// Токен из query param ?token=... (браузерный WS не поддерживает заголовки)
	// Или из Authorization заголовка (для curl/wscat)
	token := extractToken(r)
	if token == "" {
		responseHandler.ErrorResponse(
			fmt.Errorf("token required: %w", core_error.ErrUnauthorized),
			"unauthorized",
		)
		return
	}

	userID, username, err := h.tokenValidator.ValidateAccessToken(ctx, token)
	if err != nil {
		responseHandler.ErrorResponse(err, "unauthorized")
		return
	}
	if username == "" {
		username = userID
	}

	if roomID != "" {
		log.Debug("ws: checking membership",
			zap.String("user_id", userID),
			zap.String("room_id", roomID),
		)

		isMember, err := h.roomRepo.IsMember(ctx, roomID, userID)
		if err != nil {
			log.Error("ws: failed to check membership", zap.Error(err))
			responseHandler.ErrorResponse(err, "failed to check room membership")
			return
		}
		if !isMember {
			responseHandler.ErrorResponse(
				fmt.Errorf("not a member of this room: %w", core_error.ErrUnauthorized),
				"forbidden",
			)
			return
		}
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("ws: failed to upgrade connection", zap.Error(err))
		return
	}

	client := ws_client.NewClient(userID, username, roomID, conn, log)
	client.Serve(func() {
		h.hub.Register(client)
	}, h.service.Handle, h.service.OnClose)
}

func extractToken(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}
	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return after
	}
	return ""
}

func (h *WSHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range h.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
