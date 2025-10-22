package e2e_test

import (
	"time"

	"github.com/gorilla/websocket"
)

// Helper function to create a WebSocket dialer
func newDialer() *websocket.Dialer {
	return &websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
}
