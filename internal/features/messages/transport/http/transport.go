package messages_transport_http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_response "go-chat/internal/core/transport/http/response"
	core_http_server "go-chat/internal/core/transport/http/server"

	messages_service "go-chat/internal/features/messages/service"
)

type MessagesHandler struct {
	service MessagesService
}

type MessagesService interface {
	GetMessages(ctx context.Context, roomID, userID string, before *time.Time, limit int) (messages_service.GetMessagesResult, error)
}

func NewMessagesHandler(service MessagesService) *MessagesHandler {
	return &MessagesHandler{service: service}
}

func (h *MessagesHandler) Routes(auth core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:     http.MethodGet,
			Path:       "/rooms/{roomId}/messages",
			Handler:    h.GetMessages,
			Middleware: []core_http_middleware.Middleware{auth},
		},
	}
}

func (h *MessagesHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	if roomID == "" {
		resp.ErrorResponse(
			fmt.Errorf("roomId required: %w", core_error.ErrInvalidArgument),
			"bad request",
		)
		return
	}

	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		ms, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			resp.ErrorResponse(
				fmt.Errorf("invalid before cursor: %w", core_error.ErrInvalidArgument),
				"bad request",
			)
			return
		}
		t := time.UnixMilli(ms).UTC()
		before = &t
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		l, err := strconv.Atoi(raw)
		if err != nil || l <= 0 {
			resp.ErrorResponse(
				fmt.Errorf("invalid limit: %w", core_error.ErrInvalidArgument),
				"bad request",
			)
			return
		}
		limit = l
	}

	result, err := h.service.GetMessages(ctx, roomID, userID, before, limit)
	if err != nil {
		resp.ErrorResponse(err, "failed to get messages")
		return
	}

	type response struct {
		Messages   interface{} `json:"messages"`
		NextCursor *int64      `json:"next_cursor,omitempty"`
		HasMore    bool        `json:"has_more"`
	}

	var nextCursor *int64
	if result.NextCursor != nil {
		ms := result.NextCursor.UnixMilli()
		nextCursor = &ms
	}

	resp.JSONResponse(response{
		Messages:   result.Messages,
		NextCursor: nextCursor,
		HasMore:    result.HasMore,
	}, http.StatusOK)
}
