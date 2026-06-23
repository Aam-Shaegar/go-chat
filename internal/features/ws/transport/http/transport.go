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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: в проде ограничить по origin
	},
}

type WSHandler struct {
	service        WSService
	hub            Hub
	roomRepo       RoomRepository
	tokenValidator TokenValidator
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

func NewWSHandler(service WSService, hub Hub, roomRepo RoomRepository, tokenValidator TokenValidator) *WSHandler {
	return &WSHandler{
		service:        service,
		hub:            hub,
		roomRepo:       roomRepo,
		tokenValidator: tokenValidator,
	}
}

func (h *WSHandler) Routes(authMiddleware core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		{
			// Без authMiddleware — валидируем токен сами из query param
			Method:  http.MethodGet,
			Path:    "/ws/rooms/{roomId}",
			Handler: h.ServeWS,
		},
	}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

	// Токен из query param ?token=... (браузерный WS не поддерживает заголовки)
	// Или из Authorization заголовка (для curl/wscat)
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
			token = after
		}
	}
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

	roomID := r.PathValue("roomId")
	if roomID == "" {
		responseHandler.ErrorResponse(
			fmt.Errorf("roomId is required: %w", core_error.ErrInvalidArgument),
			"bad request",
		)
		return
	}

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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("ws: failed to upgrade connection", zap.Error(err))
		return
	}

	client := ws_client.NewClient(userID, username, roomID, conn, log)
	h.hub.Register(client)

	go client.WritePump()
	client.ReadPump(h.service.Handle, h.service.OnClose)
}
