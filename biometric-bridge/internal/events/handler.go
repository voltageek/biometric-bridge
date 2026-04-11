package events

import (
	"log/slog"
	"net/http"
	"time"

	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/driver"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation is handled by CORS middleware on the HTTP layer.
		// WebSocket upgrade requests that pass CORS are allowed.
		return true
	},
}

// NewHandler returns an HTTP handler for GET /events that upgrades to WebSocket,
// validates the JWT from the "token" query param, enforces the subscriber limit,
// and schedules a close frame when the JWT expires (close code 4001, "token expired").
func NewHandler(broker *Broker, validator *auth.TokenValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate JWT from query param
		claims, err := auth.ExtractWSToken(validator, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		// Subscribe to broker (checks limit)
		ch, ok := broker.Subscribe()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"max subscribers reached"}`))
			return
		}

		// Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "error", err)
			broker.Unsubscribe(ch)
			return
		}

		slog.Info("websocket subscriber connected",
			"remote", r.RemoteAddr,
			"subscribers", broker.SubscriberCount(),
		)

		// Schedule token expiry timer
		var expiryTimer *time.Timer
		if claims.ExpiresAt != nil {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl <= 0 {
				// Token already expired (shouldn't happen since Validate passed, but be safe)
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(4001, "token expired"))
				conn.Close()
				broker.Unsubscribe(ch)
				return
			}
			expiryTimer = time.NewTimer(ttl)
		}

		// Run event loop in a goroutine
		go serveWS(conn, ch, broker, expiryTimer)
	}
}

func serveWS(conn *websocket.Conn, ch chan driver.Event, broker *Broker, expiryTimer *time.Timer) {
	defer func() {
		broker.Unsubscribe(ch)
		conn.Close()
		slog.Info("websocket subscriber disconnected",
			"subscribers", broker.SubscriberCount(),
		)
	}()

	// Start a goroutine to detect client disconnect (reads are discarded)
	clientGone := make(chan struct{})
	go func() {
		defer close(clientGone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	var expiryCh <-chan time.Time
	if expiryTimer != nil {
		expiryCh = expiryTimer.C
		defer expiryTimer.Stop()
	}

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				// Broker closed the channel (shutdown)
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal closure"))
				return
			}
			if err := conn.WriteJSON(evt); err != nil {
				slog.Debug("websocket write error", "error", err)
				return
			}

		case <-expiryCh:
			slog.Info("websocket token expired, closing connection")
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "token expired"))
			return

		case <-clientGone:
			return
		}
	}
}
