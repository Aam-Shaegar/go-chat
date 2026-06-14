package ws_client

import (
	"encoding/json"
	"sync"
	"time"

	ws_domain "go-chat/internal/features/ws/domain"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeTimeout   = 10 * time.Second
	pongTimeout    = 60 * time.Second
	pingInterval   = 54 * time.Second
	maxMessageSize = 4096
)

// Client — одно активное WebSocket соединение.
// Потокобезопасен: SendEvent/Close можно вызывать из любой горутины.
type Client struct {
	ID       string
	Username string
	RoomID   string
	conn     *websocket.Conn
	send     chan ws_domain.OutgoingEvent
	once     sync.Once      // гарантирует закрытие канала ровно один раз
	wg       sync.WaitGroup // ждём завершения обеих pump горутин
	log      Logger
}

type Logger interface {
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

func NewClient(userID, username, roomID string, conn *websocket.Conn, log Logger) *Client {
	return &Client{
		ID:       userID,
		Username: username,
		RoomID:   roomID,
		conn:     conn,
		send:     make(chan ws_domain.OutgoingEvent, 256),
		log:      log,
	}
}

// Close закрывает канал send ровно один раз
func (c *Client) Close() {
	c.once.Do(func() {
		close(c.send)
	})
}

// Wait блокирует до завершения обеих pump горутин — для graceful shutdown
func (c *Client) Wait() {
	c.wg.Wait()
}

// SendEvent отправляет событие клиенту — не блокирует, не паникует
func (c *Client) SendEvent(event ws_domain.OutgoingEvent) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Warn("ws: send on closed channel", zap.Any("recover", r))
		}
	}()
	select {
	case c.send <- event:
	default:
		c.log.Warn("ws: client send buffer full, closing",
			zap.String("user_id", c.ID),
			zap.String("room_id", c.RoomID),
		)
		c.Close()
	}
}

// ReadPump читает сообщения от клиента.
// Завершение → закрывает канал → WritePump завершается.
func (c *Client) ReadPump(
	handle func(client *Client, event ws_domain.IncomingEvent),
	onClose func(client *Client),
) {
	c.wg.Add(1)
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("ws: panic in ReadPump",
				zap.Any("recover", r),
				zap.String("user_id", c.ID),
			)
		}
		onClose(c)
		c.Close()
		c.conn.Close()
		c.wg.Done()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseNoStatusReceived,
			) {
				c.log.Warn("ws: unexpected close",
					zap.String("user_id", c.ID),
					zap.Error(err),
				)
			}
			return
		}
		var event ws_domain.IncomingEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			c.SendEvent(ws_domain.OutgoingEvent{
				Type:    ws_domain.EventTypeError,
				Payload: ws_domain.ErrorPayload{Message: "invalid json"},
			})
			continue
		}
		handle(c, event)
	}
}

// WritePump пишет сообщения клиенту.
// Запускать горутиной ДО ReadPump.
func (c *Client) WritePump() {
	c.wg.Add(1)
	ticker := time.NewTicker(pingInterval)
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("ws: panic in WritePump",
				zap.Any("recover", r),
				zap.String("user_id", c.ID),
			)
		}
		ticker.Stop()
		c.conn.Close()
		c.wg.Done()
	}()

	for {
		select {
		case event, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(event); err != nil {
				c.log.Warn("ws: write error",
					zap.String("user_id", c.ID),
					zap.Error(err),
				)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
