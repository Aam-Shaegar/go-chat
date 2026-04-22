package handler

import (
	"encoding/json"
	"go-chat/internal/repository"
	"net/http"
	"strings"
)

type ReadsHandler struct {
	readsRepo *repository.ReadsRepository
}

func NewReadsHandler(readsRepo *repository.ReadsRepository) *ReadsHandler {
	return &ReadsHandler{readsRepo: readsRepo}
}

func (h *ReadsHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	roomID = strings.TrimSuffix(roomID, "/read")

	var input struct {
		MessageID string `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.MessageID == "" {
		writeError(w, http.StatusBadRequest, "message_id is required")
		return
	}
	if err := h.readsRepo.Upsert(r.Context(), roomID, userID, input.MessageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update read status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ReadsHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	roomID = strings.TrimSuffix(roomID, "/read")

	msgID, err := h.readsRepo.Get(r.Context(), roomID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get read position")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"last_read_message_id": msgID})
}

func (h *ReadsHandler) GetUnreadCounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	counts, err := h.readsRepo.GetUnreadCounts(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to ger inread counts")
		return
	}

	writeJSON(w, http.StatusOK, counts)
}
