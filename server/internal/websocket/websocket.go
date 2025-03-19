package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/coder/websocket"
)

// Hub manages all active WebSocket connections
type Hub struct {
	bufferSize int

	logger *slog.Logger
	models data.Models

	// Map user IDs to their active connections
	subscribers  map[string]*subscriber
	subscriberMu sync.RWMutex
}

type subscriber struct {
	UserID int64
	PeerID int64
	Conn   *websocket.Conn
	// Buffered channel for sending messages to the subscriber
	SendBuffer chan []byte
	// CloseSlow is called when the subscriber is slow to read messages
	CloseSlow func()
}

func NewHub(logger *slog.Logger, models data.Models) *Hub {
	return &Hub{
		subscribers: make(map[string]*subscriber),
		logger:      logger,
		models:      models,
		bufferSize:  16,
	}
}

func (h *Hub) HandleConnection(ctx context.Context, userID int64, peerID int64, conn *websocket.Conn) {
	var mu sync.Mutex

	s := &subscriber{
		UserID:     userID,
		PeerID:     peerID,
		Conn:       conn,
		SendBuffer: make(chan []byte, h.bufferSize),
		CloseSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			conn.Close(websocket.StatusPolicyViolation, "connection too slow to receive messages")
		},
	}

	defer conn.CloseNow()

	h.registerSubscriber(s)
	defer h.unregisterSubscriber(userID, peerID)
	defer s.Conn.CloseNow()

	ctx = s.Conn.CloseRead(context.Background())

	for {
		select {
		case msg := <-s.SendBuffer:
			err := writeTimeout(ctx, time.Second*5, s.Conn, msg)
			if err != nil {
				h.logger.Error("failed to write message", "error", err)
				return
			}
		case <-ctx.Done():
			h.logger.Info("connection closed", "user_id", userID)
			return
		}
	}
}

func (h *Hub) registerSubscriber(s *subscriber) {
	h.subscriberMu.Lock()
	defer h.subscriberMu.Unlock()

	// Close existing connection if any
	if existing, ok := h.subscribers[fmt.Sprintf("%d-%d", s.UserID, s.PeerID)]; ok {
		existing.Conn.Close(websocket.StatusGoingAway, "new subscription established")
	}

	h.subscribers[fmt.Sprintf("%d-%d", s.UserID, s.PeerID)] = s
	h.logger.Info("user subscribed", "user_id", s.UserID)
}

func (h *Hub) unregisterSubscriber(userID int64, peerID int64) {
	h.subscriberMu.Lock()
	defer h.subscriberMu.Unlock()

	if s, ok := h.subscribers[fmt.Sprintf("%d-%d", userID, peerID)]; ok {
		delete(h.subscribers, fmt.Sprintf("%d-%d", userID, peerID))
		close(s.SendBuffer)
		h.logger.Info("user disconnected", "user_id", userID)
	}
}

// writeTimeout writes a message to the WebSocket with a timeout
func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}

func (h *Hub) PublishToUser(peerID int64, userID int64, msg []byte) {
	h.logger.Info("publishing message to user", "user_id", userID)
	h.subscriberMu.RLock()
	s, ok := h.subscribers[fmt.Sprintf("%d-%d", userID, peerID)]
	defer h.subscriberMu.RUnlock()
	h.logger.Info("publishing message to user next step", "user_id", userID)
	if ok {
		// select sends message to SendBuffer channel if its ready
		// if the channel is full, default executes and s.CloseSlow() is called
		select {
		case s.SendBuffer <- msg:
			h.logger.Info("message published to user", "user_id", userID)
		default:
			h.logger.Error("failed to publish message to user", "user_id", userID)
			go s.CloseSlow()
		}
	}
}
