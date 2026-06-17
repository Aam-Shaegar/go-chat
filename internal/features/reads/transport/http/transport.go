package reads_transport_http

import (
	"context"
	"fmt"
	"net/http"

	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_response "go-chat/internal/core/transport/http/response"
	core_http_server "go-chat/internal/core/transport/http/server"
)

type ReadsHandler struct {
	service ReadsService
}

type ReadsService interface {
	MarkRead(ctx context.Context, roomID, userID string) error
	GetUnreadCounts(ctx context.Context, userID string) (map[string]int64, error)
	GetUnreadCount(ctx context.Context, roomID, userID string) (int64, error)
}

func NewReadsHandler(service ReadsService) *ReadsHandler {
	return &ReadsHandler{service: service}
}

func (h *ReadsHandler) Routes(auth core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:     http.MethodPost,
			Path:       "/rooms/{roomId}/read",
			Handler:    h.MarkRead,
			Middleware: []core_http_middleware.Middleware{auth},
		},
		{
			Method:     http.MethodGet,
			Path:       "/rooms/{roomId}/unread",
			Handler:    h.GetUnreadCount,
			Middleware: []core_http_middleware.Middleware{auth},
		},
		{
			Method:     http.MethodGet,
			Path:       "/reads/unread",
			Handler:    h.GetUnreadCounts,
			Middleware: []core_http_middleware.Middleware{auth},
		},
	}
}

func (h *ReadsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
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
		resp.ErrorResponse(fmt.Errorf("roomId required: %w", core_error.ErrInvalidArgument), "bad request")
		return
	}

	if err := h.service.MarkRead(ctx, roomID, userID); err != nil {
		resp.ErrorResponse(err, "failed to mark as read")
		return
	}
	resp.NoContentResponse()
}

func (h *ReadsHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
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
		resp.ErrorResponse(fmt.Errorf("roomId required: %w", core_error.ErrInvalidArgument), "bad request")
		return
	}

	count, err := h.service.GetUnreadCount(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get unread count")
		return
	}
	resp.JSONResponse(map[string]int64{"unread": count}, http.StatusOK)
}

func (h *ReadsHandler) GetUnreadCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	counts, err := h.service.GetUnreadCounts(ctx, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get unread counts")
		return
	}
	resp.JSONResponse(counts, http.StatusOK)
}
