package dm_transport_http

import (
	"context"
	"fmt"
	"net/http"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_response "go-chat/internal/core/transport/http/response"
	core_http_server "go-chat/internal/core/transport/http/server"
)

type DMHandler struct {
	service DMService
}

type DMService interface {
	OpenDM(ctx context.Context, requesterID, targetUserID string) (domain_models.Room, error)
	GetUserDMs(ctx context.Context, userID string) ([]domain_models.Room, error)
}

func NewDMHandler(service DMService) *DMHandler {
	return &DMHandler{service: service}
}

func (h *DMHandler) Routes(auth core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:     http.MethodPost,
			Path:       "/dm/{userId}",
			Handler:    h.OpenDM,
			Middleware: []core_http_middleware.Middleware{auth},
		},
		{
			Method:     http.MethodGet,
			Path:       "/dm",
			Handler:    h.GetUserDMs,
			Middleware: []core_http_middleware.Middleware{auth},
		},
	}
}

func (h *DMHandler) OpenDM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	requesterID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	targetUserID := r.PathValue("userId")
	if targetUserID == "" {
		resp.ErrorResponse(
			fmt.Errorf("userId required: %w", core_error.ErrInvalidArgument),
			"bad request",
		)
		return
	}

	room, err := h.service.OpenDM(ctx, requesterID, targetUserID)
	if err != nil {
		resp.ErrorResponse(err, "failed to open DM")
		return
	}
	resp.JSONResponse(room, http.StatusOK)
}

func (h *DMHandler) GetUserDMs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	rooms, err := h.service.GetUserDMs(ctx, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get DMs")
		return
	}
	resp.JSONResponse(rooms, http.StatusOK)
}
