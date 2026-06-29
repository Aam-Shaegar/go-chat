package messages_transport_http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	domain_models "go-chat/internal/core/domain/models"
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
	GetMessages(ctx context.Context, roomID, userID string, before *domain_models.MessageCursor, limit int) (messages_service.GetMessagesResult, error)
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

	var before *domain_models.MessageCursor
	if raw := r.URL.Query().Get("before"); raw != "" {
		cursor, err := parseMessageCursor(raw)
		if err != nil {
			resp.ErrorResponse(
				fmt.Errorf("invalid before cursor: %w", core_error.ErrInvalidArgument),
				"bad request",
			)
			return
		}
		before = cursor
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
		NextCursor *string     `json:"next_cursor,omitempty"`
		HasMore    bool        `json:"has_more"`
	}

	var nextCursor *string
	if result.NextCursor != nil {
		nextCursor = encodeMessageCursor(result.NextCursor)
	}

	resp.JSONResponse(response{
		Messages:   result.Messages,
		NextCursor: nextCursor,
		HasMore:    result.HasMore,
	}, http.StatusOK)
}

func parseMessageCursor(raw string) (*domain_models.MessageCursor, error) {
	if createdRaw, id, ok := strings.Cut(raw, "|"); ok {
		createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse composite cursor: %w", err)
		}
		if id == "" {
			return nil, fmt.Errorf("parse composite cursor: empty message id")
		}
		return &domain_models.MessageCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
	}

	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return &domain_models.MessageCursor{CreatedAt: time.UnixMilli(ms).UTC()}, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	return &domain_models.MessageCursor{CreatedAt: createdAt.UTC()}, nil
}

func encodeMessageCursor(cursor *domain_models.MessageCursor) *string {
	if cursor == nil {
		return nil
	}
	encoded := cursor.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID
	return &encoded
}
