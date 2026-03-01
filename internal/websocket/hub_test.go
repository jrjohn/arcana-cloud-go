package websocket

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	return NewHub(zap.NewNop())
}

func newTestClient(hub *Hub, userID uint) *Client {
	return &Client{
		ID:     "client-" + string(rune('A'+userID)),
		UserID: userID,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub(zap.NewNop())

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}
	if hub.userClients == nil {
		t.Error("userClients map should be initialized")
	}
	if hub.roomClients == nil {
		t.Error("roomClients map should be initialized")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}
	if hub.register == nil {
		t.Error("register channel should be initialized")
	}
	if hub.unregister == nil {
		t.Error("unregister channel should be initialized")
	}
	if hub.metrics == nil {
		t.Error("metrics should be initialized")
	}
}

func TestHub_RegisterClient(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	hub.registerClient(client)

	if !hub.clients[client] {
		t.Error("client should be in clients map")
	}
	if _, ok := hub.userClients[1]; !ok {
		t.Error("user clients should have entry for userID 1")
	}
	if hub.metrics.TotalConnections != 1 {
		t.Errorf("TotalConnections = %v, want 1", hub.metrics.TotalConnections)
	}
	if hub.metrics.ActiveConnections != 1 {
		t.Errorf("ActiveConnections = %v, want 1", hub.metrics.ActiveConnections)
	}
}

func TestHub_RegisterClient_NoUserID(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 0) // No user ID

	hub.registerClient(client)

	if !hub.clients[client] {
		t.Error("client should be in clients map")
	}
	// userClients should not have entry for userID 0
	if _, ok := hub.userClients[0]; ok {
		t.Error("userClients should not have entry for userID 0")
	}
}

func TestHub_RegisterMultipleClients(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c3 := newTestClient(hub, 1) // same user as c1
	c3.ID = "c3"

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.registerClient(c3)

	if hub.GetClientCount() != 3 {
		t.Errorf("GetClientCount() = %v, want 3", hub.GetClientCount())
	}
	if len(hub.userClients[1]) != 2 {
		t.Errorf("user 1 should have 2 clients, got %d", len(hub.userClients[1]))
	}
	if hub.metrics.TotalConnections != 3 {
		t.Errorf("TotalConnections = %v, want 3", hub.metrics.TotalConnections)
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	hub.registerClient(client)
	hub.unregisterClient(client)

	if hub.clients[client] {
		t.Error("client should be removed from clients map")
	}
	if _, ok := hub.userClients[1]; ok {
		t.Error("user clients entry should be removed when no clients remain")
	}
	if hub.metrics.ActiveConnections != 0 {
		t.Errorf("ActiveConnections = %v, want 0", hub.metrics.ActiveConnections)
	}
}

func TestHub_UnregisterClient_NotRegistered(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	// Should not panic when unregistering a client that wasn't registered
	hub.unregisterClient(client)
}

func TestHub_UnregisterClient_RemovesFromRooms(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	hub.registerClient(client)
	// Manually add to room
	hub.roomClients["test-room"] = map[*Client]bool{client: true}

	hub.unregisterClient(client)

	if _, ok := hub.roomClients["test-room"]; ok {
		t.Error("room should be removed when last client leaves")
	}
}

func TestHub_HandleJoinRoom(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	op := &RoomOperation{Client: client, Room: "test-room"}
	hub.handleJoinRoom(op)

	if !hub.roomClients["test-room"][client] {
		t.Error("client should be in room")
	}
	if !client.Rooms["test-room"] {
		t.Error("client.Rooms should contain test-room")
	}
	if hub.metrics.TotalRooms != 1 {
		t.Errorf("TotalRooms = %v, want 1", hub.metrics.TotalRooms)
	}
}

func TestHub_HandleJoinRoom_MultipleTimes(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.handleJoinRoom(&RoomOperation{Client: c1, Room: "room1"})
	hub.handleJoinRoom(&RoomOperation{Client: c2, Room: "room1"})
	hub.handleJoinRoom(&RoomOperation{Client: c1, Room: "room2"})

	if hub.GetRoomCount() != 2 {
		t.Errorf("GetRoomCount() = %v, want 2", hub.GetRoomCount())
	}
	if hub.GetRoomClientCount("room1") != 2 {
		t.Errorf("GetRoomClientCount(room1) = %v, want 2", hub.GetRoomClientCount("room1"))
	}
}

func TestHub_HandleLeaveRoom(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 1)

	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "test-room"})
	hub.handleLeaveRoom(&RoomOperation{Client: client, Room: "test-room"})

	if _, ok := hub.roomClients["test-room"]; ok {
		t.Error("room should be removed when last client leaves")
	}
	if client.Rooms["test-room"] {
		t.Error("client.Rooms should not contain test-room")
	}
}

func TestHub_HandleLeaveRoom_MultipleClients(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.handleJoinRoom(&RoomOperation{Client: c1, Room: "room1"})
	hub.handleJoinRoom(&RoomOperation{Client: c2, Room: "room1"})
	hub.handleLeaveRoom(&RoomOperation{Client: c1, Room: "room1"})

	// Room should still exist with c2
	if _, ok := hub.roomClients["room1"]; !ok {
		t.Error("room should still exist")
	}
	if hub.GetRoomClientCount("room1") != 1 {
		t.Errorf("GetRoomClientCount(room1) = %v, want 1", hub.GetRoomClientCount("room1"))
	}
}

func TestHub_HandleBroadcast_ToAll(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.registerClient(c1)
	hub.registerClient(c2)

	msg := &Message{Type: MessageTypeMessage, Data: "hello all"}
	hub.handleBroadcast(msg)

	if len(c1.send) != 1 {
		t.Errorf("c1 should have received 1 message, got %d", len(c1.send))
	}
	if len(c2.send) != 1 {
		t.Errorf("c2 should have received 1 message, got %d", len(c2.send))
	}
	if hub.metrics.TotalBroadcasts != 1 {
		t.Errorf("TotalBroadcasts = %v, want 1", hub.metrics.TotalBroadcasts)
	}
}

func TestHub_HandleBroadcast_ToRoom(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.handleJoinRoom(&RoomOperation{Client: c1, Room: "room1"})

	msg := &Message{Type: MessageTypeMessage, Room: "room1"}
	hub.handleBroadcast(msg)

	if len(c1.send) != 1 {
		t.Errorf("c1 should have received 1 message, got %d", len(c1.send))
	}
	if len(c2.send) != 0 {
		t.Errorf("c2 should not have received message, got %d messages", len(c2.send))
	}
}

func TestHub_HandleBroadcast_ToUser(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.registerClient(c1)
	hub.registerClient(c2)

	msg := &Message{Type: MessageTypeMessage, UserID: 1}
	hub.handleBroadcast(msg)

	if len(c1.send) != 1 {
		t.Errorf("c1 should have received 1 message, got %d", len(c1.send))
	}
	if len(c2.send) != 0 {
		t.Errorf("c2 should not have received message, got %d messages", len(c2.send))
	}
}

func TestHub_HandleBroadcast_FullBuffer(t *testing.T) {
	hub := newTestHub(t)
	// Create a client with a 0-buffer channel (actually must be at least 1, so use 1)
	client := &Client{
		ID:     "buffer-test",
		UserID: 99,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, 1),
		logger: zap.NewNop(),
	}
	// Fill the buffer first
	client.send <- &Message{Type: MessageTypeMessage}

	hub.registerClient(client)

	// This should not panic (full buffer is handled gracefully)
	msg := &Message{Type: MessageTypeMessage, Data: "overflow"}
	hub.handleBroadcast(msg)
}

func TestHub_GetClientCount(t *testing.T) {
	hub := newTestHub(t)

	if hub.GetClientCount() != 0 {
		t.Errorf("GetClientCount() = %v, want 0", hub.GetClientCount())
	}

	c1 := newTestClient(hub, 1)
	hub.registerClient(c1)

	if hub.GetClientCount() != 1 {
		t.Errorf("GetClientCount() = %v, want 1", hub.GetClientCount())
	}
}

func TestHub_GetRoomCount(t *testing.T) {
	hub := newTestHub(t)

	if hub.GetRoomCount() != 0 {
		t.Errorf("GetRoomCount() = %v, want 0", hub.GetRoomCount())
	}

	client := newTestClient(hub, 1)
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "room1"})
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "room2"})

	if hub.GetRoomCount() != 2 {
		t.Errorf("GetRoomCount() = %v, want 2", hub.GetRoomCount())
	}
}

func TestHub_GetRoomClientCount(t *testing.T) {
	hub := newTestHub(t)

	// Non-existent room
	if hub.GetRoomClientCount("missing") != 0 {
		t.Errorf("GetRoomClientCount(missing) = %v, want 0", hub.GetRoomClientCount("missing"))
	}

	client := newTestClient(hub, 1)
	hub.handleJoinRoom(&RoomOperation{Client: client, Room: "room1"})

	if hub.GetRoomClientCount("room1") != 1 {
		t.Errorf("GetRoomClientCount(room1) = %v, want 1", hub.GetRoomClientCount("room1"))
	}
}

func TestHub_GetMetrics(t *testing.T) {
	hub := newTestHub(t)
	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.registerClient(c1)
	hub.registerClient(c2)
	hub.handleBroadcast(&Message{Type: MessageTypeMessage})

	metrics := hub.GetMetrics()

	if metrics.TotalConnections != 2 {
		t.Errorf("TotalConnections = %v, want 2", metrics.TotalConnections)
	}
	if metrics.ActiveConnections != 2 {
		t.Errorf("ActiveConnections = %v, want 2", metrics.ActiveConnections)
	}
	if metrics.TotalBroadcasts != 1 {
		t.Errorf("TotalBroadcasts = %v, want 1", metrics.TotalBroadcasts)
	}
}

func TestHub_IsUserOnline(t *testing.T) {
	hub := newTestHub(t)
	client := newTestClient(hub, 42)

	if hub.IsUserOnline(42) {
		t.Error("User should not be online before registration")
	}

	hub.registerClient(client)

	if !hub.IsUserOnline(42) {
		t.Error("User should be online after registration")
	}

	hub.unregisterClient(client)

	if hub.IsUserOnline(42) {
		t.Error("User should not be online after unregistration")
	}
}

func TestHub_GetOnlineUsers(t *testing.T) {
	hub := newTestHub(t)

	users := hub.GetOnlineUsers()
	if len(users) != 0 {
		t.Errorf("GetOnlineUsers() = %v, want empty", users)
	}

	c1 := newTestClient(hub, 1)
	c2 := newTestClient(hub, 2)
	c2.ID = "c2"

	hub.registerClient(c1)
	hub.registerClient(c2)

	users = hub.GetOnlineUsers()
	if len(users) != 2 {
		t.Errorf("GetOnlineUsers() len = %v, want 2", len(users))
	}
}

func TestHub_Run_RegisterUnregister(t *testing.T) {
	hub := newTestHub(t)

	go hub.Run()

	client := &Client{
		ID:     "run-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	// Register via channel
	hub.register <- client

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	if hub.GetClientCount() != 1 {
		t.Errorf("GetClientCount() = %v, want 1", hub.GetClientCount())
	}

	// Unregister via channel
	hub.unregister <- client

	time.Sleep(10 * time.Millisecond)

	if hub.GetClientCount() != 0 {
		t.Errorf("GetClientCount() = %v, want 0", hub.GetClientCount())
	}
}

func TestHub_Run_JoinLeaveRoom(t *testing.T) {
	hub := newTestHub(t)

	go hub.Run()

	client := &Client{
		ID:     "room-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.joinRoom <- &RoomOperation{Client: client, Room: "test-room"}
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomClientCount("test-room") != 1 {
		t.Errorf("GetRoomClientCount() = %v, want 1", hub.GetRoomClientCount("test-room"))
	}

	hub.leaveRoom <- &RoomOperation{Client: client, Room: "test-room"}
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomCount() != 0 {
		t.Errorf("GetRoomCount() = %v, want 0", hub.GetRoomCount())
	}
}

func TestHub_Run_Broadcast(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "broadcast-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(&Message{Type: MessageTypeMessage, Data: "hello"})
	time.Sleep(10 * time.Millisecond)

	if len(client.send) != 1 {
		t.Errorf("client should have received 1 message, got %d", len(client.send))
	}
}

func TestHub_BroadcastToRoom(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "room-broadcast-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)
	hub.joinRoom <- &RoomOperation{Client: client, Room: "chat"}
	time.Sleep(10 * time.Millisecond)

	msg := &Message{Type: MessageTypeMessage}
	hub.BroadcastToRoom("chat", msg)
	time.Sleep(10 * time.Millisecond)

	if msg.Room != "chat" {
		t.Errorf("msg.Room = %v, want chat", msg.Room)
	}
	if len(client.send) != 1 {
		t.Errorf("client should have 1 message, got %d", len(client.send))
	}
}

func TestHub_BroadcastToUser(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "user-broadcast-test",
		UserID: 7,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	msg := &Message{Type: MessageTypeMessage}
	hub.BroadcastToUser(7, msg)
	time.Sleep(10 * time.Millisecond)

	if msg.UserID != 7 {
		t.Errorf("msg.UserID = %v, want 7", msg.UserID)
	}
	if len(client.send) != 1 {
		t.Errorf("client should have 1 message, got %d", len(client.send))
	}
}

func TestHub_JoinRoom_ViaChannel(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "join-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.JoinRoom(client, "testroom")
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomClientCount("testroom") != 1 {
		t.Errorf("GetRoomClientCount(testroom) = %v, want 1", hub.GetRoomClientCount("testroom"))
	}
}

func TestHub_LeaveRoom_ViaChannel(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "leave-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.JoinRoom(client, "leaveroom")
	time.Sleep(10 * time.Millisecond)

	hub.LeaveRoom(client, "leaveroom")
	time.Sleep(10 * time.Millisecond)

	if hub.GetRoomCount() != 0 {
		t.Errorf("GetRoomCount() = %v, want 0", hub.GetRoomCount())
	}
}

func TestHub_SendHeartbeat(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	client := &Client{
		ID:     "heartbeat-test",
		UserID: 1,
		Rooms:  make(map[string]bool),
		hub:    hub,
		send:   make(chan *Message, sendBufferSize),
		logger: zap.NewNop(),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.SendHeartbeat()
	time.Sleep(10 * time.Millisecond)

	if len(client.send) != 1 {
		t.Errorf("client should have 1 heartbeat message, got %d", len(client.send))
	}

	msg := <-client.send
	if msg.Type != MessageTypePing {
		t.Errorf("msg.Type = %v, want %v", msg.Type, MessageTypePing)
	}
}

func TestHub_Concurrency(t *testing.T) {
	hub := newTestHub(t)
	go hub.Run()

	var wg sync.WaitGroup

	// Concurrent registrations and reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &Client{
				ID:     "concurrent-" + string(rune('0'+id)),
				UserID: uint(id + 1),
				Rooms:  make(map[string]bool),
				hub:    hub,
				send:   make(chan *Message, sendBufferSize),
				logger: zap.NewNop(),
			}
			hub.register <- client
			time.Sleep(5 * time.Millisecond)
			hub.GetClientCount()
			hub.IsUserOnline(uint(id + 1))
			hub.GetOnlineUsers()
		}(i)
	}

	wg.Wait()
	time.Sleep(20 * time.Millisecond)
}
