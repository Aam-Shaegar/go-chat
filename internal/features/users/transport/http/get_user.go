package users_transport_http

import (
	"errors"
	"fmt"

	domain_dtos "go-chat/internal/core/domain/dtos"
	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_postgres_pool "go-chat/internal/core/repository/postgres/pool"
	core_http_response "go-chat/internal/core/transport/http/response"
	"net/http"
)

type GetUserResponse domain_dtos.UserResponseDTO

func (h *UsersHTTPHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(logger, w)

	userID := r.PathValue("id")
	if userID == "" {
		responseHandler.ErrorResponse(
			fmt.Errorf("id is required: %w", core_error.ErrInvalidArgument),
			"failed to get ID path value",
		)
		return
	}

	user, err := h.usersService.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			responseHandler.ErrorResponse(
				fmt.Errorf("user not found: %w", core_error.ErrNotFound),
				"user not found",
			)
			return
		}
		responseHandler.ErrorResponse(err, "failed to get user")
		return
	}

	response := GetUserResponse(domain_dtos.UserDTOFromDomain(user))
	responseHandler.JSONResponse(response, http.StatusOK)
}
