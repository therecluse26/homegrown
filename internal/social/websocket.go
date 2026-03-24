package social

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

const (
	wsPingInterval = 30 * time.Second
	wsPongTimeout  = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		// Origin check delegated to CORS middleware. [05-social §12]
		return true
	},
}

// WebSocketMessage is the JSON envelope for WebSocket messages. [05-social §12]
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// handleWebSocket upgrades an HTTP connection to WebSocket and subscribes to
// Redis pub/sub for real-time delivery. [05-social §12]
func handleWebSocket(pubsub shared.PubSub) echo.HandlerFunc {
	return func(c echo.Context) error {
		auth, err := shared.GetAuthContext(c)
		if err != nil {
			return err
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			return nil
		}

		parentID := auth.ParentID
		channel := "ws:parent:" + parentID.String()

		sub, err := pubsub.Subscribe(c.Request().Context(), channel)
		if err != nil {
			slog.Error("pubsub subscribe failed", "parent_id", parentID, "error", err)
			if closeErr := conn.Close(); closeErr != nil {
				slog.Debug("websocket close failed", "error", closeErr)
			}
			return nil
		}

		// Thread-safe WebSocket writer.
		var writeMu sync.Mutex
		writeJSON := func(msg WebSocketMessage) {
			writeMu.Lock()
			defer writeMu.Unlock()
			if writeErr := conn.WriteJSON(msg); writeErr != nil {
				slog.Debug("websocket write failed", "parent_id", parentID, "error", writeErr)
			}
		}

		// Ping/pong keepalive — prevents proxy timeouts. [RFC 6455 §5.5.2]
		conn.SetPongHandler(func(_ string) error {
			return conn.SetReadDeadline(time.Now().Add(wsPingInterval + wsPongTimeout))
		})
		_ = conn.SetReadDeadline(time.Now().Add(wsPingInterval + wsPongTimeout))

		// Write goroutine: forwards Redis pub/sub messages to WebSocket.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for data := range sub.Channel() {
				var msg WebSocketMessage
				if unmarshalErr := json.Unmarshal(data, &msg); unmarshalErr != nil {
					slog.Debug("websocket unmarshal failed", "error", unmarshalErr)
					continue
				}
				writeJSON(msg)
			}
		}()

		// Ping goroutine: sends periodic pings to the client.
		pingDone := make(chan struct{})
		go func() {
			defer close(pingDone)
			ticker := time.NewTicker(wsPingInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					writeMu.Lock()
					err := conn.WriteMessage(websocket.PingMessage, nil)
					writeMu.Unlock()
					if err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}()

		// Read loop: handles incoming frames (typing indicators, pings).
		for {
			_, _, readErr := conn.ReadMessage()
			if readErr != nil {
				break
			}
		}

		// Cleanup on disconnect.
		if closeErr := sub.Close(); closeErr != nil {
			slog.Debug("pubsub unsubscribe failed", "error", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil {
			slog.Debug("websocket close failed", "error", closeErr)
		}
		<-done
		<-pingDone
		return nil
	}
}

// publishToParent sends a WebSocket message to a specific parent via Redis pub/sub.
func publishToParent(pubsub shared.PubSub, parentID uuid.UUID, msgType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("websocket payload marshal failed", "error", err)
		return
	}
	msg := WebSocketMessage{
		Type:    msgType,
		Payload: data,
	}
	msgData, err := json.Marshal(msg)
	if err != nil {
		slog.Error("websocket message marshal failed", "error", err)
		return
	}
	channel := "ws:parent:" + parentID.String()
	if pubErr := pubsub.Publish(context.Background(), channel, msgData); pubErr != nil {
		slog.Debug("websocket publish failed", "parent_id", parentID, "error", pubErr)
	}
}
