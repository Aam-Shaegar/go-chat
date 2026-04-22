package handler

import (
	"encoding/json"
	"go-chat/internal/service"
	"net/http"
	"strings"
)

type InviteHandler struct {
	inviteService *service.InviteService
}

func NewInviteHandler(inviteService *service.InviteService) *InviteHandler {
	return &InviteHandler{inviteService: inviteService}
}

func (h *InviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	roomID = strings.TrimSuffix(roomID, "/invites")
	invite, err := h.inviteService.Create(r.Context(), roomID, userID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, invite)
}

func (h *InviteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	roomID = strings.TrimSuffix(roomID, "/invites")
	invites, err := h.inviteService.ListByRoom(r.Context(), roomID, userID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, invites)
}

func (h *InviteHandler) Accept(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	inviteID := strings.TrimPrefix(r.URL.Path, "/api/invites/")
	inviteID = strings.TrimSuffix(inviteID, "/accept")
	room, err := h.inviteService.Accept(r.Context(), inviteID, userID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (h *InviteHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	inviteID := strings.TrimPrefix(r.URL.Path, "/api/invites/")
	var dummy struct{}
	json.NewDecoder(r.Body).Decode(&dummy)

	invite, err := h.inviteService.GetByID(r.Context(), inviteID)
	if err != nil {
		writeError(w, http.StatusNotFound, "invite not found")
		return
	}
	writeJSON(w, http.StatusOK, invite)
}

func (h *InviteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	inviteID := strings.TrimPrefix(r.URL.Path, "/api/invites/")
	if err := h.inviteService.Delete(r.Context(), inviteID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
