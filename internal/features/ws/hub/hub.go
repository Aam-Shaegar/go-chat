package ws_hub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	ws_client "go-chat/internal/features/ws/client"
	ws_domain "go-chat/internal/features/ws/domain"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	redisRoomChannelPrefix = "gochat:room:"
	redisUserChannelPrefix = "gochat:user:"
)

type Hub struct {
	rooms map[string]map[string]*ws_client.Client
	users map[string]map[string]*ws_client.Client
	mu    sync.RWMutex
	redis *redis.Client
	log   Logger
}

type Logger interface {
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

func NewHub(redisClient *redis.Client, log Logger) *Hub {
	return &Hub{
		rooms: make(map[string]map[string]*ws_client.Client),
		users: make(map[string]map[string]*ws_client.Client),
		redis: redisClient,
		log:   log,
	}
}

// Run подписывается на Redis и раздаёт события клиентам.
// Блокирует до отмены ctx.
func (h *Hub) Run(ctx context.Context) {
	pubsub := h.redis.PSubscribe(ctx, redisRoomChannelPrefix+"*", redisUserChannelPrefix+"*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var event ws_domain.OutgoingEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				h.log.Error("hub: unmarshal redis message", zap.Error(err))
				continue
			}
			switch {
			case strings.HasPrefix(msg.Channel, redisRoomChannelPrefix):
				roomID := msg.Channel[len(redisRoomChannelPrefix):]
				h.broadcastRoom(roomID, event)
			case strings.HasPrefix(msg.Channel, redisUserChannelPrefix):
				userID := msg.Channel[len(redisUserChannelPrefix):]
				h.broadcastUser(userID, event)
			}
		}
	}
}

// Shutdown закрывает всех клиентов и ждёт завершения их горутин.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	clients := make([]*ws_client.Client, 0)
	for _, room := range h.rooms {
		for _, c := range room {
			clients = append(clients, c)
		}
	}
	for _, userClients := range h.users {
		for _, c := range userClients {
			clients = append(clients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range clients {
		c.Close()
	}
	for _, c := range clients {
		c.Wait()
	}
	h.log.Debug("hub: shutdown complete")
}

func (h *Hub) Register(client *ws_client.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.RoomID == "" {
		if h.users[client.ID] == nil {
			h.users[client.ID] = make(map[string]*ws_client.Client)
		}
		h.users[client.ID][client.ConnectionID] = client
		h.log.Debug("hub: user client registered",
			zap.String("connection_id", client.ConnectionID),
			zap.String("user_id", client.ID),
		)
		return
	}

	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[string]*ws_client.Client)
	}
	h.rooms[client.RoomID][client.ConnectionID] = client
	h.log.Debug("hub: room client registered",
		zap.String("connection_id", client.ConnectionID),
		zap.String("user_id", client.ID),
		zap.String("room_id", client.RoomID),
	)
}

func (h *Hub) Unregister(client *ws_client.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.RoomID == "" {
		if clients, ok := h.users[client.ID]; ok {
			delete(clients, client.ConnectionID)
			if len(clients) == 0 {
				delete(h.users, client.ID)
			}
		}
		h.log.Debug("hub: user client unregistered",
			zap.String("connection_id", client.ConnectionID),
			zap.String("user_id", client.ID),
		)
		return
	}

	if clients, ok := h.rooms[client.RoomID]; ok {
		delete(clients, client.ConnectionID)
		if len(clients) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}
	h.log.Debug("hub: room client unregistered",
		zap.String("connection_id", client.ConnectionID),
		zap.String("user_id", client.ID),
		zap.String("room_id", client.RoomID),
	)
}

func (h *Hub) Publish(ctx context.Context, roomID string, event ws_domain.OutgoingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := h.redis.Publish(ctx, redisRoomChannelPrefix+roomID, data).Err(); err != nil {
		return fmt.Errorf("redis publish: %w", err)
	}
	return nil
}

func (h *Hub) PublishToUser(ctx context.Context, userID string, event ws_domain.OutgoingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := h.redis.Publish(ctx, redisUserChannelPrefix+userID, data).Err(); err != nil {
		return fmt.Errorf("redis publish user: %w", err)
	}
	return nil
}

func (h *Hub) broadcastRoom(roomID string, event ws_domain.OutgoingEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}
	for _, client := range clients {
		client.SendEvent(event)
	}
}

func (h *Hub) broadcastUser(userID string, event ws_domain.OutgoingEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.users[userID]
	if !ok {
		return
	}
	for _, client := range clients {
		client.SendEvent(event)
	}
}
