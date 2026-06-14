package rooms_transport_http

import (
	"context"
	"net/http"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_server "go-chat/internal/core/transport/http/server"
)

type RoomsHandler struct {
	service RoomsService
}

type RoomsService interface {
	CreateRoom(ctx context.Context, name, description, ownerID string, isPrivate bool) (domain_models.Room, error)
	GetRoom(ctx context.Context, roomID, userID string) (domain_models.Room, error)
	GetPublicRooms(ctx context.Context, limit, offset int) ([]domain_models.Room, error)
	GetUserRooms(ctx context.Context, userID string) ([]domain_models.Room, error)
	DeleteRoom(ctx context.Context, roomID, userID string) error

	JoinPublicRoom(ctx context.Context, roomID, userID string) error
	LeaveRoom(ctx context.Context, roomID, userID string) error
	KickMember(ctx context.Context, roomID, requesterID, targetUserID string) error
	UpdateMemberRole(ctx context.Context, roomID, requesterID, targetUserID string, role domain_models.MemberRole) error
	GetMembers(ctx context.Context, roomID, userID string) ([]domain_models.RoomMember, error)

	CreateInvite(ctx context.Context, roomID, userID string, maxUses int, ttl *time.Duration) (domain_models.RoomInvite, error)
	AcceptInvite(ctx context.Context, token, userID string) (domain_models.Room, error)
	GetRoomInvites(ctx context.Context, roomID, userID string) ([]domain_models.RoomInvite, error)
	DeactivateInvite(ctx context.Context, token, userID string) error
}

func NewRoomsHandler(service RoomsService) *RoomsHandler {
	return &RoomsHandler{service: service}
}

func (h *RoomsHandler) Routes(auth core_http_middleware.Middleware) []core_http_server.Route {
	return []core_http_server.Route{
		// Rooms
		{Method: http.MethodPost, Path: "/rooms", Handler: h.CreateRoom, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodGet, Path: "/rooms", Handler: h.GetPublicRooms, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodGet, Path: "/rooms/my", Handler: h.GetUserRooms, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodGet, Path: "/rooms/{roomId}", Handler: h.GetRoom, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodDelete, Path: "/rooms/{roomId}", Handler: h.DeleteRoom, Middleware: []core_http_middleware.Middleware{auth}},

		// Members
		{Method: http.MethodPost, Path: "/rooms/{roomId}/join", Handler: h.JoinRoom, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodPost, Path: "/rooms/{roomId}/leave", Handler: h.LeaveRoom, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodGet, Path: "/rooms/{roomId}/members", Handler: h.GetMembers, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodDelete, Path: "/rooms/{roomId}/members/{userId}", Handler: h.KickMember, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodPatch, Path: "/rooms/{roomId}/members/{userId}/role", Handler: h.UpdateMemberRole, Middleware: []core_http_middleware.Middleware{auth}},

		// Invites
		{Method: http.MethodPost, Path: "/rooms/{roomId}/invites", Handler: h.CreateInvite, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodGet, Path: "/rooms/{roomId}/invites", Handler: h.GetInvites, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodDelete, Path: "/invites/{token}", Handler: h.DeactivateInvite, Middleware: []core_http_middleware.Middleware{auth}},
		{Method: http.MethodPost, Path: "/invites/{token}/accept", Handler: h.AcceptInvite, Middleware: []core_http_middleware.Middleware{auth}},
	}
}
