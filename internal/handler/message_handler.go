package handler

import (
	"go-chat/internal/service"
	"go-chat/internal/ws"
	"net/http"
	"strings"
)

type MessageHandler struct {
	messageService *service.MessageService
	hub            *ws.Hub
}

func NewMessageHandler(messageService *service.MessageService, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		hub:            hub,
	}
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
	parts := strings.Split(path, "/messages/")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	roomID := parts[0]
	messageID := parts[1]
	if err := h.messageService.Delete(r.Context(), messageID, roomID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	h.hub.BroadcastMessageDeleted(roomID, messageID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
