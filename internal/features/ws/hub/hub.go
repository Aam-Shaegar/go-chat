package ws_hub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	ws_client "go-chat/internal/features/ws/client"
	ws_domain "go-chat/internal/features/ws/domain"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const redisChannelPrefix = "gochat:room:"

type Hub struct {
	rooms map[string]map[string]*ws_client.Client
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
		redis: redisClient,
		log:   log,
	}
}

// Run подписывается на Redis и раздаёт события клиентам.
// Блокирует до отмены ctx.
func (h *Hub) Run(ctx context.Context) {
	pubsub := h.redis.PSubscribe(ctx, redisChannelPrefix+"*")
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
			roomID := msg.Channel[len(redisChannelPrefix):]
			h.broadcast(roomID, event)
		}
	}
}

// Shutdown закрывает всех клиентов и ждёт завершения их горутин.
// Вызывать после отмены ctx переданного в Run.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	clients := make([]*ws_client.Client, 0)
	for _, room := range h.rooms {
		for _, c := range room {
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

	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[string]*ws_client.Client)
	}
	h.rooms[client.RoomID][client.ID] = client
	h.log.Debug("hub: client registered",
		zap.String("user_id", client.ID),
		zap.String("room_id", client.RoomID),
	)
}

func (h *Hub) Unregister(client *ws_client.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[client.RoomID]; ok {
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}
	h.log.Debug("hub: client unregistered",
		zap.String("user_id", client.ID),
		zap.String("room_id", client.RoomID),
	)
}

func (h *Hub) Publish(ctx context.Context, roomID string, event ws_domain.OutgoingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := h.redis.Publish(ctx, redisChannelPrefix+roomID, data).Err(); err != nil {
		return fmt.Errorf("redis publish: %w", err)
	}
	return nil
}

func (h *Hub) broadcast(roomID string, event ws_domain.OutgoingEvent) {
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
