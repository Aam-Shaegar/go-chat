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

// Request/Response types

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

type muteMemberRequest struct {
	MutedUntil string `json:"muted_until" validate:"required"`
}

type banMemberRequest struct {
	Reason     string  `json:"reason" validate:"max=255"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// Handlers

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

	// Get target member info before kicking
	targetMember, err := h.service.GetMember(ctx, roomID, targetID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get target member")
		return
	}

	// Get requester member info
	requesterMember, err := h.service.GetMember(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get requester member")
		return
	}

	if err := h.service.KickMember(ctx, roomID, userID, targetID); err != nil {
		resp.ErrorResponse(err, "failed to kick member")
		return
	}

	// Publish user_kicked event via WebSocket
	if h.wsSvc != nil {
		_ = h.wsSvc.PublishUserKicked(ctx, roomID, targetID, targetMember.Username, userID, requesterMember.Username)
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

func (h *RoomsHandler) MuteMember(w http.ResponseWriter, r *http.Request) {
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

	var req muteMemberRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		resp.ErrorResponse(err, "invalid request")
		return
	}

	mutedUntil, err := time.Parse(time.RFC3339, req.MutedUntil)
	if err != nil {
		resp.ErrorResponse(fmt.Errorf("invalid muted_until format, use RFC3339: %w", core_error.ErrInvalidArgument), "invalid request")
		return
	}

	if err := h.service.MuteMember(ctx, roomID, userID, targetID, mutedUntil); err != nil {
		resp.ErrorResponse(err, "failed to mute member")
		return
	}

	// Get target member for event payload
	targetMember, err := h.service.GetMember(ctx, roomID, targetID)
	if err == nil {
		// Get requester info
		requesterMember, _ := h.service.GetMember(ctx, roomID, userID)
		requesterName := ""
		if requesterMember.Username != "" {
			requesterName = requesterMember.Username
		}

		// Publish mute event via WebSocket service
		if h.wsSvc != nil {
			_ = h.wsSvc.PublishUserMuted(ctx, roomID, targetID, targetMember.Username, userID, requesterName, mutedUntil)
		}
	}

	resp.NoContentResponse()
}

func (h *RoomsHandler) UnmuteMember(w http.ResponseWriter, r *http.Request) {
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

	// Get target member for event payload
	targetMember, err := h.service.GetMember(ctx, roomID, targetID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get target member")
		return
	}

	if err := h.service.UnmuteMember(ctx, roomID, userID, targetID); err != nil {
		resp.ErrorResponse(err, "failed to unmute member")
		return
	}

	// Get requester info
	requesterMember, _ := h.service.GetMember(ctx, roomID, userID)
	requesterName := ""
	if requesterMember.Username != "" {
		requesterName = requesterMember.Username
	}

	// Publish unmute event via WebSocket service
	if h.wsSvc != nil {
		_ = h.wsSvc.PublishUserUnmuted(ctx, roomID, targetID, targetMember.Username, userID, requesterName)
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
	room, invite, err := h.service.AcceptInvite(ctx, token, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to accept invite")
		return
	}

	// Publish invite used event via WebSocket
	if h.wsSvc != nil {
		_ = h.wsSvc.PublishInviteUsed(ctx, invite.RoomID, invite.Token, invite.Uses, invite.MaxUses)
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

	// Get room ID from invite before deactivating
	invite, err := h.service.GetInviteByToken(ctx, token)
	if err != nil {
		resp.ErrorResponse(err, "invite not found")
		return
	}
	roomID := invite.RoomID

	if err := h.service.DeactivateInvite(ctx, token, userID); err != nil {
		resp.ErrorResponse(err, "failed to deactivate invite")
		return
	}

	// Publish invite deactivated event via WebSocket
	if h.wsSvc != nil {
		_ = h.wsSvc.PublishInviteDeactivated(ctx, roomID, token)
	}

	resp.NoContentResponse()
}

func (h *RoomsHandler) BanMember(w http.ResponseWriter, r *http.Request) {
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

	var req banMemberRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		resp.ErrorResponse(err, "invalid request")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			resp.ErrorResponse(fmt.Errorf("invalid expires_at format, use RFC3339: %w", core_error.ErrInvalidArgument), "invalid request")
			return
		}
		expiresAt = &t
	}

	targetMember, err := h.service.GetMember(ctx, roomID, targetID)
	if err != nil {
		resp.ErrorResponse(err, "target member not found")
		return
	}
	requesterMember, _ := h.service.GetMember(ctx, roomID, userID)

	if err := h.service.BanMember(ctx, roomID, userID, targetID, req.Reason, expiresAt); err != nil {
		resp.ErrorResponse(err, "failed to ban member")
		return
	}

	if h.wsSvc != nil {
		_ = h.wsSvc.PublishUserBanned(ctx, roomID, targetID, targetMember.Username, userID, requesterMember.Username, req.Reason, expiresAt)
	}

	resp.NoContentResponse()
}

func (h *RoomsHandler) UnbanMember(w http.ResponseWriter, r *http.Request) {
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

	// Get requester info for WS event
	requesterMember, _ := h.service.GetMember(ctx, roomID, userID)
	// Get ban info for target username
	bans, _ := h.service.GetRoomBans(ctx, roomID, userID)
	var targetUsername string
	for _, ban := range bans {
		if ban.UserID == targetID {
			targetUsername = ban.Username
			break
		}
	}
	if targetUsername == "" {
		targetUsername = targetID
	}

	if err := h.service.UnbanMember(ctx, roomID, userID, targetID); err != nil {
		resp.ErrorResponse(err, "failed to unban member")
		return
	}

	if h.wsSvc != nil {
		_ = h.wsSvc.PublishUserUnbanned(ctx, roomID, targetID, targetUsername, userID, requesterMember.Username)
	}

	resp.NoContentResponse()
}

func (h *RoomsHandler) GetBans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	resp := core_http_response.NewHTTPResponseHandler(log, w)

	userID, err := core_http_middleware.UserIDFromContext(ctx)
	if err != nil {
		resp.ErrorResponse(err, "unauthorized")
		return
	}

	roomID := r.PathValue("roomId")

	bans, err := h.service.GetRoomBans(ctx, roomID, userID)
	if err != nil {
		resp.ErrorResponse(err, "failed to get bans")
		return
	}

	resp.JSONResponse(bans, http.StatusOK)
}
