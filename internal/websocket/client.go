package websocket

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 65536

	// Send buffer size
	sendBufferSize = 256
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeMessage      MessageType = "message"
	MessageTypeNotification MessageType = "notification"
	MessageTypeEvent        MessageType = "event"
	MessageTypePing         MessageType = "ping"
	MessageTypePong         MessageType = "pong"
	MessageTypeError        MessageType = "error"
	MessageTypeSubscribe    MessageType = "subscribe"
	MessageTypeUnsubscribe  MessageType = "unsubscribe"
	MessageTypeAck          MessageType = "ack"
)

// Message represents a WebSocket message
type Message struct {
	ID        string                 `json:"id,omitempty"`
	Type      MessageType            `json:"type"`
	Event     string                 `json:"event,omitempty"`
	Room      string                 `json:"room,omitempty"`
	UserID    uint                   `json:"userId,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewEventMessage creates a new event message
func NewEventMessage(event string, data interface{}) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeEvent,
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewNotification creates a new notification message
func NewNotification(title, body string) *Message {
	return &Message{
		ID:   uuid.New().String(),
		Type: MessageTypeNotification,
		Data: map[string]string{
			"title": title,
			"body":  body,
		},
		Timestamp: time.Now(),
	}
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	UserID   uint
	Username string
	Rooms    map[string]bool
	hub      *Hub
	conn     *websocket.Conn
	send     chan *Message
	logger   *zap.Logger
	metadata map[string]interface{}
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uint, username string, logger *zap.Logger) *Client {
	return &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		Username: username,
		Rooms:    make(map[string]bool),
		hub:      hub,
		conn:     conn,
		send:     make(chan *Message, sendBufferSize),
		logger:   logger,
		metadata: make(map[string]interface{}),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("WebSocket read error",
					zap.String("client_id", c.ID),
					zap.Error(err),
				)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			c.logger.Warn("Failed to parse message",
				zap.String("client_id", c.ID),
				zap.Error(err),
			)
			continue
		}

		// Handle message based on type
		c.handleMessage(&message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				c.logger.Warn("Failed to write message",
					zap.String("client_id", c.ID),
					zap.Error(err),
				)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(message *Message) {
	switch message.Type {
	case MessageTypePing:
		// Respond with pong
		c.send <- &Message{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		}

	case MessageTypeSubscribe:
		// Subscribe to a room
		if room, ok := message.Data.(string); ok {
			c.hub.JoinRoom(c, room)
			c.send <- &Message{
				Type:      MessageTypeAck,
				Data:      map[string]string{"action": "subscribed", "room": room},
				Timestamp: time.Now(),
			}
		}

	case MessageTypeUnsubscribe:
		// Unsubscribe from a room
		if room, ok := message.Data.(string); ok {
			c.hub.LeaveRoom(c, room)
			c.send <- &Message{
				Type:      MessageTypeAck,
				Data:      map[string]string{"action": "unsubscribed", "room": room},
				Timestamp: time.Now(),
			}
		}

	case MessageTypeMessage:
		// Broadcast message to room if specified, otherwise to all
		message.UserID = c.UserID
		message.Timestamp = time.Now()
		c.hub.Broadcast(message)

	default:
		c.logger.Debug("Unknown message type",
			zap.String("client_id", c.ID),
			zap.String("type", string(message.Type)),
		)
	}
}

// Send sends a message to the client
func (c *Client) Send(message *Message) {
	select {
	case c.send <- message:
	default:
		c.logger.Warn("Client send buffer full",
			zap.String("client_id", c.ID),
		)
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.hub.unregister <- c
}

// SetMetadata sets client metadata
func (c *Client) SetMetadata(key string, value interface{}) {
	c.metadata[key] = value
}

// GetMetadata gets client metadata
func (c *Client) GetMetadata(key string) (interface{}, bool) {
	value, ok := c.metadata[key]
	return value, ok
}
