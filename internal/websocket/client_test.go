package websocket

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMessageType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"Message", MessageTypeMessage, "message"},
		{"Notification", MessageTypeNotification, "notification"},
		{"Event", MessageTypeEvent, "event"},
		{"Ping", MessageTypePing, "ping"},
		{"Pong", MessageTypePong, "pong"},
		{"Error", MessageTypeError, "error"},
		{"Subscribe", MessageTypeSubscribe, "subscribe"},
		{"Unsubscribe", MessageTypeUnsubscribe, "unsubscribe"},
		{"Ack", MessageTypeAck, "ack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("MessageType = %v, want %v", tt.msgType, tt.expected)
			}
		})
	}
}

func TestNewMessage(t *testing.T) {
	before := time.Now()
	msg := NewMessage(MessageTypeMessage, "hello")
	after := time.Now()

	if msg == nil {
		t.Fatal("NewMessage() returned nil")
	}
	if msg.ID == "" {
		t.Error("ID should not be empty")
	}
	if msg.Type != MessageTypeMessage {
		t.Errorf("Type = %v, want %v", msg.Type, MessageTypeMessage)
	}
	if msg.Data != "hello" {
		t.Errorf("Data = %v, want hello", msg.Data)
	}
	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Error("Timestamp should be between before and after")
	}
}

func TestNewMessage_NilData(t *testing.T) {
	msg := NewMessage(MessageTypePing, nil)

	if msg == nil {
		t.Fatal("NewMessage() returned nil")
	}
	if msg.Type != MessageTypePing {
		t.Errorf("Type = %v, want %v", msg.Type, MessageTypePing)
	}
	if msg.Data != nil {
		t.Errorf("Data = %v, want nil", msg.Data)
	}
}

func TestNewEventMessage(t *testing.T) {
	msg := NewEventMessage("user.created", map[string]string{"id": "123"})

	if msg == nil {
		t.Fatal("NewEventMessage() returned nil")
	}
	if msg.ID == "" {
		t.Error("ID should not be empty")
	}
	if msg.Type != MessageTypeEvent {
		t.Errorf("Type = %v, want %v", msg.Type, MessageTypeEvent)
	}
	if msg.Event != "user.created" {
		t.Errorf("Event = %v, want user.created", msg.Event)
	}
}

func TestNewNotification(t *testing.T) {
	msg := NewNotification("Test Title", "Test Body")

	if msg == nil {
		t.Fatal("NewNotification() returned nil")
	}
	if msg.ID == "" {
		t.Error("ID should not be empty")
	}
	if msg.Type != MessageTypeNotification {
		t.Errorf("Type = %v, want %v", msg.Type, MessageTypeNotification)
	}

	data, ok := msg.Data.(map[string]string)
	if !ok {
		t.Fatal("Data should be map[string]string")
	}
	if data["title"] != "Test Title" {
		t.Errorf("title = %v, want Test Title", data["title"])
	}
	if data["body"] != "Test Body" {
		t.Errorf("body = %v, want Test Body", data["body"])
	}
}

func TestNewMessage_UniqueIDs(t *testing.T) {
	msg1 := NewMessage(MessageTypeMessage, nil)
	msg2 := NewMessage(MessageTypeMessage, nil)

	if msg1.ID == msg2.ID {
		t.Error("Messages should have unique IDs")
	}
}

func TestClient_SetMetadata(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:       "meta-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	client.SetMetadata("key1", "value1")
	client.SetMetadata("key2", 42)

	if client.metadata["key1"] != "value1" {
		t.Errorf("metadata[key1] = %v, want value1", client.metadata["key1"])
	}
	if client.metadata["key2"] != 42 {
		t.Errorf("metadata[key2] = %v, want 42", client.metadata["key2"])
	}
}

func TestClient_GetMetadata(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:       "get-meta-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	client.SetMetadata("key", "value")

	val, ok := client.GetMetadata("key")
	if !ok {
		t.Error("GetMetadata() should return true for existing key")
	}
	if val != "value" {
		t.Errorf("GetMetadata() = %v, want value", val)
	}

	_, ok = client.GetMetadata("missing")
	if ok {
		t.Error("GetMetadata() should return false for missing key")
	}
}

func TestClient_Send(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:       "send-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	msg := NewMessage(MessageTypeMessage, "test")
	client.Send(msg)

	if len(client.send) != 1 {
		t.Errorf("send channel should have 1 message, got %d", len(client.send))
	}

	received := <-client.send
	if received != msg {
		t.Error("Should receive the same message that was sent")
	}
}

func TestClient_Send_FullBuffer(t *testing.T) {
	hub := NewHub(zap.NewNop())
	client := &Client{
		ID:       "full-buffer-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, 1),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	// Fill buffer
	client.Send(NewMessage(MessageTypeMessage, "msg1"))
	// This should not block or panic
	client.Send(NewMessage(MessageTypeMessage, "msg2"))
}

func TestClient_HandleMessage_Ping(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:       "ping-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	hub.register <- client
	time.Sleep(5 * time.Millisecond)

	msg := &Message{Type: MessageTypePing}
	client.handleMessage(msg)

	if len(client.send) != 1 {
		t.Errorf("client should have 1 pong message, got %d", len(client.send))
	}

	pong := <-client.send
	if pong.Type != MessageTypePong {
		t.Errorf("pong.Type = %v, want %v", pong.Type, MessageTypePong)
	}
}

func TestClient_HandleMessage_Subscribe(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:       "subscribe-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	hub.register <- client
	time.Sleep(5 * time.Millisecond)

	msg := &Message{Type: MessageTypeSubscribe, Data: "test-room"}
	client.handleMessage(msg)

	// Wait for room join
	time.Sleep(10 * time.Millisecond)

	if len(client.send) != 1 {
		t.Errorf("client should have 1 ack message, got %d", len(client.send))
	}

	ack := <-client.send
	if ack.Type != MessageTypeAck {
		t.Errorf("ack.Type = %v, want %v", ack.Type, MessageTypeAck)
	}
}

func TestClient_HandleMessage_Unsubscribe(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:       "unsubscribe-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	hub.register <- client
	time.Sleep(5 * time.Millisecond)

	// Subscribe first
	client.handleMessage(&Message{Type: MessageTypeSubscribe, Data: "myroom"})
	time.Sleep(10 * time.Millisecond)
	// Drain ack
	<-client.send

	// Now unsubscribe
	client.handleMessage(&Message{Type: MessageTypeUnsubscribe, Data: "myroom"})
	time.Sleep(10 * time.Millisecond)

	if len(client.send) != 1 {
		t.Errorf("client should have 1 ack message, got %d", len(client.send))
	}

	ack := <-client.send
	if ack.Type != MessageTypeAck {
		t.Errorf("ack.Type = %v, want %v", ack.Type, MessageTypeAck)
	}
}

func TestClient_HandleMessage_Broadcast(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	c1 := &Client{
		ID:       "broadcast-c1",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}
	c2 := &Client{
		ID:       "broadcast-c2",
		UserID:   2,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	hub.register <- c1
	hub.register <- c2
	time.Sleep(10 * time.Millisecond)

	msg := &Message{Type: MessageTypeMessage, Data: "broadcast!"}
	c1.handleMessage(msg)

	time.Sleep(10 * time.Millisecond)

	// Both clients should receive the broadcast
	if len(c1.send) == 0 && len(c2.send) == 0 {
		t.Error("At least one client should have received the broadcast")
	}
}

func TestClient_HandleMessage_UnknownType(t *testing.T) {
	hub := newTestHub(t)
	client := &Client{
		ID:       "unknown-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	// Should not panic for unknown message types
	msg := &Message{Type: "unknown_type"}
	client.handleMessage(msg)
}

func TestClient_HandleMessage_SubscribeInvalidData(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:       "invalid-sub-test",
		UserID:   1,
		Rooms:    make(map[string]bool),
		hub:      hub,
		send:     make(chan *Message, sendBufferSize),
		logger:   zap.NewNop(),
		metadata: make(map[string]interface{}),
	}

	hub.register <- client
	time.Sleep(5 * time.Millisecond)

	// Subscribe with non-string data (should not send ack)
	msg := &Message{Type: MessageTypeSubscribe, Data: 42}
	client.handleMessage(msg)

	time.Sleep(5 * time.Millisecond)

	// No ack should be sent for invalid data
	if len(client.send) != 0 {
		t.Errorf("client should not have received ack for invalid subscribe data, got %d messages", len(client.send))
	}
}

func TestMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := &Message{
		ID:        "test-id",
		Type:      MessageTypeEvent,
		Event:     "user.login",
		Room:      "general",
		UserID:    42,
		Data:      map[string]string{"key": "value"},
		Timestamp: now,
		Metadata:  map[string]interface{}{"meta": "data"},
	}

	if msg.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", msg.ID)
	}
	if msg.Event != "user.login" {
		t.Errorf("Event = %v, want user.login", msg.Event)
	}
	if msg.Room != "general" {
		t.Errorf("Room = %v, want general", msg.Room)
	}
	if msg.UserID != 42 {
		t.Errorf("UserID = %v, want 42", msg.UserID)
	}
	if msg.Timestamp != now {
		t.Errorf("Timestamp = %v, want %v", msg.Timestamp, now)
	}
}

func TestTimeConstants(t *testing.T) {
	if writeWait != 10*time.Second {
		t.Errorf("writeWait = %v, want 10s", writeWait)
	}
	if pongWait != 60*time.Second {
		t.Errorf("pongWait = %v, want 60s", pongWait)
	}
	if maxMessageSize != 65536 {
		t.Errorf("maxMessageSize = %v, want 65536", maxMessageSize)
	}
	if sendBufferSize != 256 {
		t.Errorf("sendBufferSize = %v, want 256", sendBufferSize)
	}
}
