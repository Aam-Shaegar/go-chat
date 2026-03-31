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
	hub          *ws.Hub
	roomService  *service.RoomService
	tokenService *service.TokenService
}

type WSHandlerDeps struct {
	Hub          *ws.Hub
	RoomService  *service.RoomService
	TokenService *service.TokenService
}

func NewWSHandler(deps WSHandlerDeps) *WSHandler {
	return &WSHandler{
		hub:          deps.Hub,
		roomService:  deps.RoomService,
		tokenService: deps.TokenService,
	}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	var userID string

	if tokenStr != "" {
		claims, err := h.tokenService.Parse(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		userID = claims.UserID
	} else {
		id, ok := GetUserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID = id
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
