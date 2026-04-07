package ws

import (
	"context"
	"go-chat/internal/domain"
	"go-chat/internal/repository"
	"go-chat/internal/service"
	"log"
)

type ClientMessage struct {
	Client  *Client
	Message IncomingMessage
}

type Hub struct {
	rooms       map[string]map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	process     chan ClientMessage
	messageRepo *repository.MessageRepository
	userRepo    *repository.UserRepository
	roomService *service.RoomService
}

func NewHub(
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	roomService *service.RoomService,
) *Hub {
	return &Hub{
		rooms:       make(map[string]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		process:     make(chan ClientMessage),
		messageRepo: messageRepo,
		userRepo:    userRepo,
		roomService: roomService,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		case cm := <-h.process:
			h.handleMessage(cm)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
	if h.rooms[client.roomID] == nil {
		h.rooms[client.roomID] = make(map[*Client]bool)
	}
	h.rooms[client.roomID][client] = true

	h.broadcastToRoom(client.roomID, OutgoingMessage{
		Type: TypeUserJoined,
		Payload: UserEventPayload{
			UserID:   client.userID,
			Username: client.username,
		},
	}, client)

	h.broadcastRoomStats(client.roomID)

	log.Printf("client joined room %s (total: %d)", client.roomID, len(h.rooms[client.roomID]))
}

func (h *Hub) handleUnregister(client *Client) {
	if clients, ok := h.rooms[client.roomID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(h.rooms, client.roomID)
			} else {
				h.broadcastToRoom(client.roomID, OutgoingMessage{
					Type: TypeUserLeft,
					Payload: UserEventPayload{
						UserID:   client.userID,
						Username: client.username,
					},
				}, nil)
			}
		}
	}

	h.broadcastRoomStats(client.roomID)

	log.Printf("client left room %s", client.roomID)
}

func (h *Hub) handleMessage(cm ClientMessage) {
	switch cm.Message.Type {
	case TypeSendMessage:
		h.handleSendMessage(cm.Client, cm.Message.Content)
	case TypeTyping:
		h.handleTyping(cm.Client)
	default:
		cm.Client.send <- OutgoingMessage{Type: TypeError, Payload: "unknown message type"}
	}
}

func (h *Hub) handleSendMessage(client *Client, content string) {
	if content == "" {
		client.send <- OutgoingMessage{Type: TypeError, Payload: "content cannot be empty"}
		return
	}

	if len(content) > 4000 {
		client.send <- OutgoingMessage{Type: TypeError, Payload: "message too long"}
		return
	}
	msg, err := h.messageRepo.Create(context.Background(), domain.Message{
		RoomID:  client.roomID,
		UserID:  client.userID,
		Content: content,
	})
	if err != nil {
		log.Printf("save message error: %v", err)
		client.send <- OutgoingMessage{Type: TypeError, Payload: "failed to save message"}
		return
	}

	h.broadcastToRoom(client.roomID, OutgoingMessage{
		Type: TypeNewMessage,
		Payload: MessagePayload{
			Message:  msg,
			Username: client.username,
		},
	}, nil)
}

func (h *Hub) handleTyping(client *Client) {
	h.broadcastToRoom(client.roomID, OutgoingMessage{
		Type: TypeUserTyping,
		Payload: TypingPayload{
			UserID:   client.userID,
			Username: client.username,
		},
	}, client)
}

func (h *Hub) broadcastToRoom(roomID string, msg OutgoingMessage, exclude *Client) {
	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}
	for client := range clients {
		if client == exclude {
			continue
		}
		select {
		case client.send <- msg:
		default:
			delete(clients, client)
			close(client.send)
		}
	}
}

func (h *Hub) BroadcastMessageDeleted(roomID, messageID string) {
	h.broadcastToRoom(roomID, OutgoingMessage{
		Type: TypeMessageDeleted,
		Payload: MessageDeletedPayload{
			MessageID: messageID,
			RoomID:    roomID,
		},
	}, nil)
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) broadcastRoomStats(roomID string) {
	count := len(h.rooms[roomID])
	h.broadcastToRoom(roomID, OutgoingMessage{
		Type:    TypeRoomStats,
		Payload: RoomStatsPayload{OnlineCount: count},
	}, nil)
}
