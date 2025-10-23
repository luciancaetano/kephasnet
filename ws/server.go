package ws

import (
	"net/http"

	"github.com/luciancaetano/kephasnet"
	"github.com/luciancaetano/kephasnet/internal/websocket"
)

type RateLimitConfig = websocket.RateLimitConfig
type CheckOriginFn = websocket.CheckOriginFn
type OnConnectFn = websocket.OnConnectFn
type OnDisconnectFn = websocket.OnClientDisconnectFn
type ServerConfig = *websocket.ServerConfig

// New creates a new WebSocket server with rate limiting and connection callbacks.
//
// Parameters:
//   - addr: The server address (e.g., ":8080" or "localhost:8080")
//   - rateLimitConfig: Rate limiting configuration. Use DefaultRateLimitConfig() or NoRateLimit()
//   - onCheckOrigin: Function to validate WebSocket origins. Use AllOrigins() to allow all (dev only)
//   - onConnect: Optional callback function called when a client connects. Can be nil.
//     This is called after the WebSocket handshake completes but before
//     the message reading loop starts. Use this to track connections,
//     send welcome messages, or perform authentication.
//
// Example:
//
//	server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(), func(client kephasnet.Client) {
//	    log.Printf("Client connected: %s", client.ID())
//	})
func New(cfg ServerConfig) kephasnet.WebsocketServer {
	return websocket.New(cfg)
}

func NewConfig(addr string, rateLimitConfig *RateLimitConfig, checkOrigin CheckOriginFn, onConnect OnConnectFn, onDisconnect OnDisconnectFn) ServerConfig {
	return &websocket.ServerConfig{
		Addr:               addr,
		RateLimitConfig:    rateLimitConfig,
		CheckOrigin:        checkOrigin,
		OnConnect:          onConnect,
		OnClientDisconnect: onDisconnect,
	}
}

// AllOrigins returns the default checkOrigin function that allows all origins
func AllOrigins() CheckOriginFn {
	return func(r *http.Request) bool {
		return true
	}
}

// DefaultRateLimitConfig returns the default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return websocket.DefaultRateLimitConfig()
}

// NoRateLimit returns a configuration with rate limiting disabled
func NoRateLimit() *RateLimitConfig {
	return websocket.NoRateLimit()
}
