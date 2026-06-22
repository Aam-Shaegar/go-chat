package users_transport_http

import (
	"context"
	core_config "go-chat/internal/core/config"
	domain_dtos "go-chat/internal/core/domain/dtos"
	domain_models "go-chat/internal/core/domain/models"
	core_http_server "go-chat/internal/core/transport/http/server"
	"net/http"
)

type UsersHTTPHandler struct {
	usersService UsersService
	cfg          *core_config.Config
}

type UsersService interface {
	GetUsers(ctx context.Context, limit, offser *int) ([]domain_models.User, error)
	GetUser(ctx context.Context, userID string) (domain_models.User, error)
	Register(ctx context.Context, username, email, pasword string) (domain_dtos.AuthResponseDTO, string, error)
	Login(ctx context.Context, email, password string) (domain_dtos.AuthResponseDTO, string, error)
}

func NewUsersHTTPHandler(usersService UsersService, cfg *core_config.Config) *UsersHTTPHandler {
	return &UsersHTTPHandler{
		usersService: usersService,
		cfg:          cfg,
	}
}

func (h *UsersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodGet,
			Path:    "/users/{id}",
			Handler: h.GetUser,
		},
		{
			Method:  http.MethodGet,
			Path:    "/users/",
			Handler: h.GetUsers,
		},
		{
			Method:  http.MethodPost,
			Path:    "/auth/register",
			Handler: h.Register,
		},
		{
			Method:  http.MethodPost,
			Path:    "/auth/login",
			Handler: h.Login,
		},
	}
}
