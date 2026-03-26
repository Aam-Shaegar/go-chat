package handler

import (
	"encoding/json"
	"go-chat/internal/service"
	"log"
	"net/http"
	"strings"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateRoomInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	room, err := h.roomService.Create(r.Context(), input, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, room)
}

func (h *RoomHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomService.ListPublic(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rooms")
		return
	}

	writeJSON(w, http.StatusOK, rooms)
}

func (h *RoomHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rooms, err := h.roomService.ListMy(r.Context(), userID)
	if err != nil {
		log.Printf("ListMy error: %s", err)
		writeError(w, http.StatusInternalServerError, "failed to list rooms")
		return
	}

	writeJSON(w, http.StatusOK, rooms)
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	roomID = strings.TrimSuffix(roomID, "/join")

	if roomID == "" {
		writeError(w, http.StatusBadRequest, "room id is required")
		return
	}

	if err := h.roomService.Join(r.Context(), roomID, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *RoomHandler) GetRoomByID(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")

	if roomID == "" {
		writeError(w, http.StatusBadRequest, "room id is required")
		return
	}
	room, err := h.roomService.GetRoomByID(r.Context(), roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	writeJSON(w, http.StatusOK, room)

}
