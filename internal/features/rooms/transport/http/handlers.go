package rooms_transport_http

import (
	"fmt"
	"net/http"
	"time"

	domain_models "go-chat/internal/core/domain/models"
	core_error "go-chat/internal/core/errors"
	core_logger "go-chat/internal/core/logger"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_request "go-chat/internal/core/transport/http/request"
	core_http_response "go-chat/internal/core/transport/http/response"
)

// --- Request/Response types ---

type createRoomRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=64"`
	Description string `json:"description" validate:"max=255"`
	IsPrivate   bool   `json:"is_private"`
}

type createInviteRequest struct {
	MaxUses  int  `json:"max_uses"`
	TTLHours *int `json:"ttl_hours"`
}

type updateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

// --- Handlers ---

func (h *RoomsHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	var req createRoomRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		resp.ErrorResponse(err, "invalid request")
		return
	}

	room, err := h.service.CreateRoom(ctx, req.Name, req.Description, userID, req.IsPrivate)
	if err != nil {
		resp.ErrorResponse(err, "failed to create room")
		return
	}
	resp.JSONResponse(room, http.StatusCreated)
}

func (h *RoomsHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
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

	room, err := h.service.GetRoom(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get room")
		return
	}
	resp.JSONResponse(room, http.StatusOK)
}

func (h *RoomsHandler) GetPublicRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	limit, err := core_http_request.GetIntQueryParam(r, "limit")
	if err != nil {
		resp.ErrorResponse(err, "invalid limit")
		return
	}
	offset, err := core_http_request.GetIntQueryParam(r, "offset")
	if err != nil {
		resp.ErrorResponse(err, "invalid offset")
		return
	}

	l, o := 20, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}

	rooms, err := h.service.GetPublicRooms(ctx, l, o)
	if err != nil {
		resp.ErrorResponse(err, "failed to get rooms")
		return
	}
	resp.JSONResponse(rooms, http.StatusOK)
}

func (h *RoomsHandler) GetUserRooms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	rooms, err := h.service.GetUserRooms(ctx, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get user rooms")
		return
	}
	resp.JSONResponse(rooms, http.StatusOK)
}

func (h *RoomsHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteRoom(ctx, roomID, userID); err != nil {
		resp.ErrorResponse(err, "failed to delete room")
		return
	}
	resp.NoContentResponse()
}

func (h *RoomsHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	if err := h.service.JoinPublicRoom(ctx, roomID, userID); err != nil {
		resp.ErrorResponse(err, "failed to join room")
		return
	}
	resp.NoContentResponse()
}

func (h *RoomsHandler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	if err := h.service.LeaveRoom(ctx, roomID, userID); err != nil {
		resp.ErrorResponse(err, "failed to leave room")
		return
	}
	resp.NoContentResponse()
}

func (h *RoomsHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	members, err := h.service.GetMembers(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get members")
		return
	}
	resp.JSONResponse(members, http.StatusOK)
}

func (h *RoomsHandler) KickMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	targetID := r.PathValue("userId")

	if err := h.service.KickMember(ctx, roomID, userID, targetID); err != nil {
		resp.ErrorResponse(err, "failed to kick member")
		return
	}
	resp.NoContentResponse()
}

func (h *RoomsHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	targetID := r.PathValue("userId")

	var req updateRoleRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		resp.ErrorResponse(err, "invalid request")
		return
	}

	if err := h.service.UpdateMemberRole(ctx, roomID, userID, targetID, domain_models.MemberRole(req.Role)); err != nil {
		resp.ErrorResponse(err, "failed to update role")
		return
	}
	resp.NoContentResponse()
}

func (h *RoomsHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")

	var req createInviteRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		resp.ErrorResponse(err, "invalid request")
		return
	}

	var ttl *time.Duration
	if req.TTLHours != nil {
		d := time.Duration(*req.TTLHours) * time.Hour
		ttl = &d
	}

	invite, err := h.service.CreateInvite(ctx, roomID, userID, req.MaxUses, ttl)
	if err != nil {
		resp.ErrorResponse(err, "failed to create invite")
		return
	}
	resp.JSONResponse(invite, http.StatusCreated)
}

func (h *RoomsHandler) GetInvites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")
	invites, err := h.service.GetRoomInvites(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get invites")
		return
	}
	resp.JSONResponse(invites, http.StatusOK)
}

func (h *RoomsHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	token := r.PathValue("token")
	room, err := h.service.AcceptInvite(ctx, token, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to accept invite")
		return
	}
	resp.JSONResponse(room, http.StatusOK)
}

func (h *RoomsHandler) DeactivateInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	token := r.PathValue("token")
	if err := h.service.DeactivateInvite(ctx, token, userID); err != nil {
		resp.ErrorResponse(err, "failed to deactivate invite")
		return
	}
	resp.NoContentResponse()
}
