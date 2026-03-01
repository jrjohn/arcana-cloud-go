package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMessage creates a valid message
func TestNewMessage(t *testing.T) {
	msg := NewMessage(MessageTypeMessage, "hello")
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, MessageTypeMessage, msg.Type)
	assert.Equal(t, "hello", msg.Data)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

// TestNewMessage_WithComplexData handles complex data types
func TestNewMessage_WithComplexData(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
		"num": 42,
	}
	msg := NewMessage(MessageTypeEvent, data)
	assert.NotNil(t, msg)
	assert.Equal(t, data, msg.Data)
}

// TestNewMessage_NilData handles nil data
func TestNewMessage_NilData(t *testing.T) {
	msg := NewMessage(MessageTypePing, nil)
	assert.NotNil(t, msg)
	assert.Nil(t, msg.Data)
}

// TestNewEventMessage creates an event message
func TestNewEventMessage(t *testing.T) {
	msg := NewEventMessage("user.created", map[string]string{"id": "123"})
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, MessageTypeEvent, msg.Type)
	assert.Equal(t, "user.created", msg.Event)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

// TestNewNotification creates a notification message
func TestNewNotification(t *testing.T) {
	msg := NewNotification("Test Title", "Test Body")
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, MessageTypeNotification, msg.Type)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)

	data, ok := msg.Data.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "Test Title", data["title"])
	assert.Equal(t, "Test Body", data["body"])
}

// TestMessageType_Constants checks message type constants
func TestMessageType_Constants(t *testing.T) {
	assert.Equal(t, MessageType("message"), MessageTypeMessage)
	assert.Equal(t, MessageType("notification"), MessageTypeNotification)
	assert.Equal(t, MessageType("event"), MessageTypeEvent)
	assert.Equal(t, MessageType("ping"), MessageTypePing)
	assert.Equal(t, MessageType("pong"), MessageTypePong)
	assert.Equal(t, MessageType("error"), MessageTypeError)
	assert.Equal(t, MessageType("subscribe"), MessageTypeSubscribe)
	assert.Equal(t, MessageType("unsubscribe"), MessageTypeUnsubscribe)
	assert.Equal(t, MessageType("ack"), MessageTypeAck)
}

// TestNewMessage_UniqueIDs each message gets a unique ID
func TestNewMessage_UniqueIDs(t *testing.T) {
	msg1 := NewMessage(MessageTypeMessage, nil)
	msg2 := NewMessage(MessageTypeMessage, nil)
	assert.NotEqual(t, msg1.ID, msg2.ID)
}

// TestNewEventMessage_UniqueIDs each event message gets unique ID
func TestNewEventMessage_UniqueIDs(t *testing.T) {
	msg1 := NewEventMessage("test.event", nil)
	msg2 := NewEventMessage("test.event", nil)
	assert.NotEqual(t, msg1.ID, msg2.ID)
}

// TestNewNotification_UniqueIDs each notification gets unique ID
func TestNewNotification_UniqueIDs(t *testing.T) {
	msg1 := NewNotification("Title", "Body")
	msg2 := NewNotification("Title", "Body")
	assert.NotEqual(t, msg1.ID, msg2.ID)
}

// TestClient_SetGetMetadata tests metadata operations
func TestClient_SetGetMetadata(t *testing.T) {
	hub := NewHub(testHubLogger())

	client := &Client{
		ID:       "meta-client",
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		metadata: make(map[string]interface{}),
	}

	client.SetMetadata("user_agent", "Mozilla/5.0")
	client.SetMetadata("ip", "192.168.1.1")
	client.SetMetadata("count", 42)

	val, ok := client.GetMetadata("user_agent")
	assert.True(t, ok)
	assert.Equal(t, "Mozilla/5.0", val)

	val, ok = client.GetMetadata("ip")
	assert.True(t, ok)
	assert.Equal(t, "192.168.1.1", val)

	val, ok = client.GetMetadata("count")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

// TestClient_GetMetadata_Missing returns false for missing key
func TestClient_GetMetadata_Missing(t *testing.T) {
	client := &Client{
		metadata: make(map[string]interface{}),
	}

	val, ok := client.GetMetadata("non-existent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

// TestClient_Send_Success sends message to client's channel
func TestClient_Send_Success(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "send-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	msg := NewMessage(MessageTypeMessage, "test")
	client.Send(msg)

	select {
	case received := <-client.send:
		assert.Equal(t, msg, received)
	default:
		t.Error("message was not sent")
	}
}

// TestClient_Send_FullBuffer logs warning when buffer is full
func TestClient_Send_FullBuffer(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "full-buf-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, 1),
		logger: testHubLogger(),
	}

	// Fill the buffer
	client.send <- NewMessage(MessageTypeMessage, "first")

	// This should not block or panic
	assert.NotPanics(t, func() {
		client.Send(NewMessage(MessageTypeMessage, "overflow"))
	})
}

// TestClient_handleMessage_Ping responds with pong
func TestClient_handleMessage_Ping(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "ping-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	client.handleMessage(&Message{Type: MessageTypePing})

	select {
	case resp := <-client.send:
		assert.Equal(t, MessageTypePong, resp.Type)
	default:
		t.Error("no pong response")
	}
}

// TestClient_handleMessage_Subscribe subscribes to a room
func TestClient_handleMessage_Subscribe(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "sub-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	// Need a room operation reader to avoid blocking
	go func() {
		for op := range hub.joinRoom {
			hub.handleJoinRoom(op)
		}
	}()

	client.handleMessage(&Message{
		Type: MessageTypeSubscribe,
		Data: "my-room",
	})

	// Should get ACK
	select {
	case resp := <-client.send:
		assert.Equal(t, MessageTypeAck, resp.Type)
		data, ok := resp.Data.(map[string]string)
		assert.True(t, ok)
		assert.Equal(t, "subscribed", data["action"])
		assert.Equal(t, "my-room", data["room"])
	case <-time.After(100 * time.Millisecond):
		t.Error("no ack received")
	}
}

// TestClient_handleMessage_Subscribe_InvalidData ignores non-string data
func TestClient_handleMessage_Subscribe_InvalidData(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "sub-invalid-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	// Data is not a string, so should not send anything
	client.handleMessage(&Message{
		Type: MessageTypeSubscribe,
		Data: 12345,
	})

	select {
	case <-client.send:
		t.Error("should not have sent anything")
	default:
		// Good
	}
}

// TestClient_handleMessage_Unsubscribe unsubscribes from a room
func TestClient_handleMessage_Unsubscribe(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "unsub-client",
		Rooms:  map[string]bool{"leave-room": true},
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	// Set up room first
	hub.roomClients["leave-room"] = map[*Client]bool{client: true}

	go func() {
		for op := range hub.leaveRoom {
			hub.handleLeaveRoom(op)
		}
	}()

	client.handleMessage(&Message{
		Type: MessageTypeUnsubscribe,
		Data: "leave-room",
	})

	select {
	case resp := <-client.send:
		assert.Equal(t, MessageTypeAck, resp.Type)
		data, ok := resp.Data.(map[string]string)
		assert.True(t, ok)
		assert.Equal(t, "unsubscribed", data["action"])
	case <-time.After(100 * time.Millisecond):
		t.Error("no ack received")
	}
}

// TestClient_handleMessage_Broadcast_Message broadcasts message to hub
func TestClient_handleMessage_Broadcast_Message(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "bcast-client",
		UserID: 77,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	go func() {
		for msg := range hub.broadcast {
			hub.handleBroadcast(msg)
		}
	}()

	// Register so it can receive its own broadcast
	hub.registerClient(client)

	msg := &Message{Type: MessageTypeMessage, Data: "hello all"}
	client.handleMessage(msg)

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, uint(77), msg.UserID)
}

// TestClient_handleMessage_Unknown logs unknown type
func TestClient_handleMessage_Unknown(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:     "unknown-client",
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: testHubLogger(),
	}

	assert.NotPanics(t, func() {
		client.handleMessage(&Message{Type: "unknown-type"})
	})
}

// TestWebSocketConfig_Default tests default config
func TestWebSocketConfig_Default(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	assert.NotNil(t, cfg)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "/ws", cfg.Path)
	assert.Equal(t, []string{"*"}, cfg.AllowedOrigins)
	assert.Equal(t, 1024, cfg.ReadBufferSize)
	assert.Equal(t, 1024, cfg.WriteBufferSize)
	assert.Equal(t, 10*time.Second, cfg.HandshakeTimeout)
	assert.True(t, cfg.EnableCompression)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatInterval)
}
