package handler

import (
	"encoding/json"
	"go-chat/internal/service"
	"net/http"
	"strings"
)

type DMHandler struct {
	dmService *service.DMService
	wsHub     DMBroadcaster
}

type DMBroadcaster interface {
	BroadcastDM(toUserID string, msg interface{})
}

func NewDMHandler(dmService *service.DMService, wsHub DMBroadcaster) *DMHandler {
	return &DMHandler{
		dmService: dmService,
		wsHub:     wsHub,
	}
}

func (h *DMHandler) Send(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	toUserID := r.PathValue("userId")

	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	msg, err := h.dmService.Send(r.Context(), userID, toUserID, input.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.wsHub.BroadcastDM(toUserID, msg)
	h.wsHub.BroadcastDM(userID, msg)
	writeJSON(w, http.StatusCreated, msg)
}

func (h *DMHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	toUserID := r.PathValue("userId")
	msgs, err := h.dmService.GetHistory(r.Context(), userID, toUserID, 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get history")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *DMHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	msgs, err := h.dmService.GetConversations(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get conversations")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *DMHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	otherUserID := strings.TrimPrefix(r.URL.Path, "/api/dm/")
	otherUserID = strings.TrimSuffix(otherUserID, "/read")

	if err := h.dmService.MarkRead(r.Context(), userID, otherUserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark read")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *DMHandler) GetUnreadCounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	counts, err := h.dmService.GetAllUnreadCounts(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get unread counts")
		return
	}

	writeJSON(w, http.StatusOK, counts)
}

func (h *DMHandler) GetLastRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	otherUserID := strings.TrimPrefix(r.URL.Path, "/api/dm/")
	otherUserID = strings.TrimSuffix(otherUserID, "/read")

	t, err := h.dmService.GetLastReadAt(r.Context(), userID, otherUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get last read")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"last_read_at": t,
	})
}
