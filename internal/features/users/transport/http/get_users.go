package users_transport_http

import (
	"fmt"
	domain_dtos "go-chat/internal/core/domain/dtos"
	core_logger "go-chat/internal/core/logger"
	core_http_request "go-chat/internal/core/transport/http/request"
	core_http_response "go-chat/internal/core/transport/http/response"
	"net/http"
)

type GetUsersResponse []domain_dtos.UserResponseDTO

func (h *UsersHTTPHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHanler := core_http_response.NewHTTPResponseHandler(log, w)

	limit, offset, err := getLimitOffsetQueryParams(r)
	if err != nil {
		responseHanler.ErrorResponse(err, "failed to get 'limit', 'offset' query parameters")
		return
	}
	userDomains, err := h.usersService.GetUsers(ctx, limit, offset)
	if err != nil {
		responseHanler.ErrorResponse(
			err,
			"failed to get users",
		)
		return
	}
	response := GetUsersResponse(domain_dtos.UsersDTOFromDomains(userDomains))
	responseHanler.JSONResponse(response, http.StatusOK)
}

func getLimitOffsetQueryParams(r *http.Request) (*int, *int, error) {
	limit, err := core_http_request.GetIntQueryParam(r, "limit")
	if err != nil {
		return nil, nil, fmt.Errorf("get 'limit' query param: %w", err)
	}
	offset, err := core_http_request.GetIntQueryParam(r, "offset")
	if err != nil {
		return nil, nil, fmt.Errorf("get 'offset' query param: %w", err)
	}
	return limit, offset, nil

}
