package ws

import (
	"net/http"

	"github.com/luciancaetano/kephasnet"
	"github.com/luciancaetano/kephasnet/internal/websocket"
)

type RateLimitConfig = websocket.RateLimitConfig
type CheckOriginFn = websocket.CheckOriginFn

// NewServer creates a new WebSocket server
func New(addr string, rateLimnit *RateLimitConfig, onCheckOrigin CheckOriginFn) kephasnet.WebsocketServer {
	return websocket.New(addr, rateLimnit, onCheckOrigin)
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
