package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func newTestHub() *Hub {
	return NewHub(zap.NewNop())
}

func TestNewHub(t *testing.T) {
	h := newTestHub()
	assert.NotNil(t, h)
	assert.Equal(t, 0, h.GetClientCount())
	assert.Equal(t, 0, h.GetRoomCount())
}

func TestHub_GetMetrics_Empty(t *testing.T) {
	h := newTestHub()
	m := h.GetMetrics()
	assert.Equal(t, int64(0), m.TotalConnections)
	assert.Equal(t, int64(0), m.ActiveConnections)
	assert.Equal(t, 0, m.TotalRooms)
}

func TestHub_GetClientCount_Empty(t *testing.T) {
	h := newTestHub()
	assert.Equal(t, 0, h.GetClientCount())
}

func TestHub_GetRoomCount_Empty(t *testing.T) {
	h := newTestHub()
	assert.Equal(t, 0, h.GetRoomCount())
}

func TestHub_GetRoomClientCount_Empty(t *testing.T) {
	h := newTestHub()
	assert.Equal(t, 0, h.GetRoomClientCount("nonexistent-room"))
}

func TestHub_IsUserOnline_Empty(t *testing.T) {
	h := newTestHub()
	assert.False(t, h.IsUserOnline(1))
}

func TestHub_GetOnlineUsers_Empty(t *testing.T) {
	h := newTestHub()
	users := h.GetOnlineUsers()
	assert.Empty(t, users)
}

func TestHub_Broadcast_NoClients(t *testing.T) {
	h := newTestHub()
	// Should not block or panic when no clients
	msg := NewMessage(MessageTypeEvent, "test")
	assert.NotPanics(t, func() {
		h.Broadcast(msg)
	})
}

func TestHub_BroadcastToRoom_NonExistentRoom(t *testing.T) {
	h := newTestHub()
	msg := NewMessage(MessageTypeEvent, "test")
	assert.NotPanics(t, func() {
		h.BroadcastToRoom("no-such-room", msg)
	})
}

func TestHub_BroadcastToUser_NotOnline(t *testing.T) {
	h := newTestHub()
	msg := NewMessage(MessageTypeEvent, "test")
	assert.NotPanics(t, func() {
		h.BroadcastToUser(999, msg)
	})
}

func TestHub_SendHeartbeat_NoClients(t *testing.T) {
	h := newTestHub()
	assert.NotPanics(t, func() {
		h.SendHeartbeat()
	})
}

func TestHub_RunIsNonBlocking(t *testing.T) {
	h := newTestHub()
	started := make(chan struct{})
	go func() {
		close(started)
		h.Run()
	}()
	select {
	case <-started:
		// Hub started successfully
	case <-time.After(time.Second):
		t.Error("Hub.Run() did not start in time")
	}
}

// ── Message helpers ──────────────────────────────────────────────────────────

func TestNewMessage(t *testing.T) {
	msg := NewMessage(MessageTypeEvent, map[string]string{"key": "value"})
	assert.NotNil(t, msg)
	assert.Equal(t, MessageTypeEvent, msg.Type)
}

func TestNewEventMessage(t *testing.T) {
	msg := NewEventMessage("user.created", map[string]interface{}{"id": 1})
	assert.NotNil(t, msg)
}

func TestNewNotification(t *testing.T) {
	msg := NewNotification("Hello", "World")
	assert.NotNil(t, msg)
}
