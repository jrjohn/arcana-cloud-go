package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testHubLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// TestNewHub creates a hub and checks it's not nil
func TestNewHub(t *testing.T) {
	hub := NewHub(testHubLogger())
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.userClients)
	assert.NotNil(t, hub.roomClients)
	assert.NotNil(t, hub.metrics)
}

// TestHub_GetClientCount returns 0 when empty
func TestHub_GetClientCount(t *testing.T) {
	hub := NewHub(testHubLogger())
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestHub_GetRoomCount returns 0 when empty
func TestHub_GetRoomCount(t *testing.T) {
	hub := NewHub(testHubLogger())
	assert.Equal(t, 0, hub.GetRoomCount())
}

// TestHub_GetRoomClientCount returns 0 for non-existent room
func TestHub_GetRoomClientCount(t *testing.T) {
	hub := NewHub(testHubLogger())
	count := hub.GetRoomClientCount("non-existent-room")
	assert.Equal(t, 0, count)
}

// TestHub_IsUserOnline returns false for unknown user
func TestHub_IsUserOnline(t *testing.T) {
	hub := NewHub(testHubLogger())
	assert.False(t, hub.IsUserOnline(999))
}

// TestHub_GetOnlineUsers returns empty slice when no users
func TestHub_GetOnlineUsers(t *testing.T) {
	hub := NewHub(testHubLogger())
	users := hub.GetOnlineUsers()
	assert.NotNil(t, users)
	assert.Empty(t, users)
}

// TestHub_GetMetrics returns zero metrics initially
func TestHub_GetMetrics(t *testing.T) {
	hub := NewHub(testHubLogger())
	metrics := hub.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalConnections)
	assert.Equal(t, int64(0), metrics.ActiveConnections)
	assert.Equal(t, int64(0), metrics.TotalMessages)
	assert.Equal(t, int64(0), metrics.TotalBroadcasts)
	assert.Equal(t, 0, metrics.TotalRooms)
}

// TestHub_registerClient_andUnregister tests client registration and unregistration
func TestHub_registerClient_andUnregister(t *testing.T) {
	hub := NewHub(testHubLogger())

	// Create a client without a real connection (we'll set fields manually)
	client := &Client{
		ID:     "test-client-1",
		UserID: 1,
		Rooms:  make(map[string]bool),
		send:   make(chan *Message, sendBufferSize),
	}

	// Register manually
	hub.registerClient(client)

	assert.Equal(t, 1, hub.GetClientCount())
	assert.Equal(t, int64(1), hub.GetMetrics().TotalConnections)
	assert.Equal(t, int64(1), hub.GetMetrics().ActiveConnections)
	assert.True(t, hub.IsUserOnline(1))

	// Unregister manually
	hub.unregisterClient(client)

	assert.Equal(t, 0, hub.GetClientCount())
	assert.Equal(t, int64(0), hub.GetMetrics().ActiveConnections)
	assert.False(t, hub.IsUserOnline(1))
}

// TestHub_registerClient_NoUserID registers a client without user ID
func TestHub_registerClient_NoUserID(t *testing.T) {
	hub := NewHub(testHubLogger())

	client := &Client{
		ID:     "anon-client",
		UserID: 0,
		Rooms:  make(map[string]bool),
		send:   make(chan *Message, sendBufferSize),
	}

	hub.registerClient(client)
	assert.Equal(t, 1, hub.GetClientCount())
	assert.False(t, hub.IsUserOnline(0))

	hub.unregisterClient(client)
}

// TestHub_unregisterClient_NotExisting does not panic
func TestHub_unregisterClient_NotExisting(t *testing.T) {
	hub := NewHub(testHubLogger())

	client := &Client{
		ID:    "ghost-client",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}

	// Should not panic even though not registered
	assert.NotPanics(t, func() {
		hub.unregisterClient(client)
	})
}

// TestHub_handleJoinRoom and handleLeaveRoom
func TestHub_handleJoinLeaveRoom(t *testing.T) {
	hub := NewHub(testHubLogger())

	client := &Client{
		ID:    "room-client",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}

	// Join room
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "room-1"})
	assert.Equal(t, 1, hub.GetRoomCount())
	assert.Equal(t, 1, hub.GetRoomClientCount("room-1"))
	assert.True(t, client.Rooms["room-1"])
	assert.Equal(t, 1, hub.GetMetrics().TotalRooms)

	// Join another room
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "room-2"})
	assert.Equal(t, 2, hub.GetRoomCount())

	// Leave room-1
	hub.handleLeaveRoom(&RoomOperation{Client: client, Room: "room-1"})
	assert.Equal(t, 1, hub.GetRoomCount())
	assert.Equal(t, 0, hub.GetRoomClientCount("room-1"))
	assert.False(t, client.Rooms["room-1"])

	// Leave room-2
	hub.handleLeaveRoom(&RoomOperation{Client: client, Room: "room-2"})
	assert.Equal(t, 0, hub.GetRoomCount())
}

// TestHub_handleLeaveRoom_NotInRoom does not panic
func TestHub_handleLeaveRoom_NotInRoom(t *testing.T) {
	hub := NewHub(testHubLogger())
	client := &Client{
		ID:    "no-room-client",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}

	assert.NotPanics(t, func() {
		hub.handleLeaveRoom(&RoomOperation{Client: client, Room: "non-existent"})
	})
}

// TestHub_handleBroadcast_AllClients broadcasts to all
func TestHub_handleBroadcast_AllClients(t *testing.T) {
	hub := NewHub(testHubLogger())

	client1 := &Client{
		ID:    "client-1",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}
	client2 := &Client{
		ID:    "client-2",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	msg := &Message{
		Type:      MessageTypeMessage,
		Data:      "hello all",
		Timestamp: time.Now(),
	}
	hub.handleBroadcast(msg)

	// Both should receive the message
	select {
	case received := <-client1.send:
		assert.Equal(t, MessageTypeMessage, received.Type)
	default:
		t.Error("client1 did not receive message")
	}
	select {
	case received := <-client2.send:
		assert.Equal(t, MessageTypeMessage, received.Type)
	default:
		t.Error("client2 did not receive message")
	}

	assert.Equal(t, int64(1), hub.GetMetrics().TotalBroadcasts)
	assert.Equal(t, int64(2), hub.GetMetrics().TotalMessages)

	// Cleanup
	hub.clients[client1] = false
	hub.clients[client2] = false
}

// TestHub_handleBroadcast_ToUser sends to specific user
func TestHub_handleBroadcast_ToUser(t *testing.T) {
	hub := NewHub(testHubLogger())

	client1 := &Client{
		ID:     "c1",
		UserID: 10,
		Rooms:  make(map[string]bool),
		send:   make(chan *Message, sendBufferSize),
	}
	client2 := &Client{
		ID:     "c2",
		UserID: 20,
		Rooms:  make(map[string]bool),
		send:   make(chan *Message, sendBufferSize),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	msg := &Message{
		Type:      MessageTypeNotification,
		UserID:    10,
		Timestamp: time.Now(),
	}
	hub.handleBroadcast(msg)

	// Only client1 should receive
	select {
	case received := <-client1.send:
		assert.Equal(t, MessageTypeNotification, received.Type)
	default:
		t.Error("client1 did not receive message")
	}

	// client2 should NOT receive
	select {
	case <-client2.send:
		t.Error("client2 should not have received message")
	default:
		// Good
	}

	// Cleanup (avoid closing send channel since we manually unregistered)
	hub.mutex.Lock()
	delete(hub.clients, client1)
	delete(hub.clients, client2)
	hub.mutex.Unlock()
}

// TestHub_handleBroadcast_ToRoom sends to room members only
func TestHub_handleBroadcast_ToRoom(t *testing.T) {
	hub := NewHub(testHubLogger())

	client1 := &Client{
		ID:    "room-c1",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}
	client2 := &Client{
		ID:    "room-c2",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}

	hub.registerClient(client1)
	hub.registerClient(client2)

	hub.handleJoinRoom(&RoomOperation{Client: client1, Room: "test-room"})

	msg := &Message{
		Type:      MessageTypeEvent,
		Room:      "test-room",
		Timestamp: time.Now(),
	}
	hub.handleBroadcast(msg)

	// client1 (in room) should receive
	select {
	case received := <-client1.send:
		assert.Equal(t, MessageTypeEvent, received.Type)
	default:
		t.Error("client1 did not receive message")
	}

	// client2 (not in room) should NOT receive
	select {
	case <-client2.send:
		t.Error("client2 should not have received message")
	default:
		// Good
	}

	// Cleanup
	hub.mutex.Lock()
	delete(hub.clients, client1)
	delete(hub.clients, client2)
	hub.mutex.Unlock()
}

// TestHub_handleBroadcast_FullBuffer logs warning for full send buffer
func TestHub_handleBroadcast_FullBuffer(t *testing.T) {
	hub := NewHub(testHubLogger())

	// Create client with tiny buffer (size 1)
	client := &Client{
		ID:    "full-buffer-client",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, 1),
	}
	// Fill the buffer
	client.send <- &Message{Type: MessageTypeMessage}

	hub.registerClient(client)

	// This should not panic but log a warning
	assert.NotPanics(t, func() {
		hub.handleBroadcast(&Message{Type: MessageTypeEvent, Timestamp: time.Now()})
	})

	// Cleanup
	hub.mutex.Lock()
	delete(hub.clients, client)
	hub.mutex.Unlock()
}

// TestHub_SendHeartbeat sends a ping to all clients
func TestHub_SendHeartbeat(t *testing.T) {
	hub := NewHub(testHubLogger())

	// Run the hub's broadcast loop briefly
	go func() {
		for msg := range hub.broadcast {
			hub.handleBroadcast(msg)
		}
	}()

	client := &Client{
		ID:    "heartbeat-client",
		Rooms: make(map[string]bool),
		send:  make(chan *Message, sendBufferSize),
	}
	hub.registerClient(client)

	hub.SendHeartbeat()

	// Give time for goroutine to process
	time.Sleep(10 * time.Millisecond)

	select {
	case msg := <-client.send:
		assert.Equal(t, MessageTypePing, msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("heartbeat not received")
	}

	// Cleanup
	hub.mutex.Lock()
	delete(hub.clients, client)
	hub.mutex.Unlock()
}

// TestHub_unregisterClient_WithRooms removes client from rooms
func TestHub_unregisterClient_WithRooms(t *testing.T) {
	hub := NewHub(testHubLogger())

	client := &Client{
		ID:     "room-unreg-client",
		UserID: 5,
		Rooms:  make(map[string]bool),
		send:   make(chan *Message, sendBufferSize),
	}

	hub.registerClient(client)
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "my-room"})

	assert.Equal(t, 1, hub.GetRoomCount())

	hub.unregisterClient(client)

	assert.Equal(t, 0, hub.GetRoomCount())
	assert.Equal(t, 0, hub.GetClientCount())
	assert.False(t, hub.IsUserOnline(5))
}

// TestHub_MultipleClientsPerUser tracks multiple connections per user
func TestHub_MultipleClientsPerUser(t *testing.T) {
	hub := NewHub(testHubLogger())

	c1 := &Client{ID: "uc1", UserID: 42, Rooms: make(map[string]bool), send: make(chan *Message, sendBufferSize)}
	c2 := &Client{ID: "uc2", UserID: 42, Rooms: make(map[string]bool), send: make(chan *Message, sendBufferSize)}

	hub.registerClient(c1)
	hub.registerClient(c2)

	assert.True(t, hub.IsUserOnline(42))
	assert.Equal(t, 2, hub.GetClientCount())

	hub.unregisterClient(c1)
	// User still online with one connection
	assert.True(t, hub.IsUserOnline(42))

	hub.unregisterClient(c2)
	// User offline now
	assert.False(t, hub.IsUserOnline(42))
}
