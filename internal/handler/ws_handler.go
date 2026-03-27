package handler

import (
	"go-chat/internal/service"
	"go-chat/internal/ws"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub         *ws.Hub
	roomService *service.RoomService
	userRepo    interface {
		GetByID(ctx interface{}, id string) (interface{}, error)
	}
}

type WSHandlerDeps struct {
	Hub         *ws.Hub
	RoomService *service.RoomService
}

func NewWSHandler(deps WSHandlerDeps) *WSHandler {
	return &WSHandler{
		hub:         deps.Hub,
		roomService: deps.RoomService,
	}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = "anonymous"
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/rooms/")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, "room id us required")
		return
	}

	isMember, err := h.roomService.IsMember(r.Context(), roomID, userID)
	if err != nil || !isMember {
		writeError(w, http.StatusForbidden, "you are not a member of this room")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, roomID, userID, username)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
