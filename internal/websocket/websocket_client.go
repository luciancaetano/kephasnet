package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/luciancaetano/kephasnet"
	"github.com/luciancaetano/kephasnet/internal/protocol"
)

// Client implements the WSClient interface
type Client struct {
	id          string
	conn        *websocket.Conn
	remoteAddr  string
	ctx         context.Context
	cancel      context.CancelFunc
	sendCh      chan []byte
	mu          sync.RWMutex
	closed      bool
	rateLimiter *rate.Limiter // Rate limiter for incoming messages
}

// NewClient creates a new WebSocket client with rate limiting
func NewClient(conn *websocket.Conn, remoteAddr string, rateLimitConfig *RateLimitConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	var limiter *rate.Limiter
	if rateLimitConfig != nil && rateLimitConfig.Enabled {
		limiter = rate.NewLimiter(rateLimitConfig.MessagesPerSecond, rateLimitConfig.Burst)
	}

	client := &Client{
		id:          uuid.New().String(),
		conn:        conn,
		remoteAddr:  remoteAddr,
		ctx:         ctx,
		cancel:      cancel,
		sendCh:      make(chan []byte, 256),
		closed:      false,
		rateLimiter: limiter,
	}

	// Start the write pump
	go client.writePump()

	return client
}

// ID returns a unique identifier for the connected client
func (c *Client) ID() string {
	return c.id
}

// RemoteAddr returns the client's remote network address
func (c *Client) RemoteAddr() string {
	return c.remoteAddr
}

// Context returns the client's lifecycle context
func (c *Client) Context() context.Context {
	return c.ctx
}

// Send encodes and sends a message with the given command ID and payload
func (c *Client) Send(ctx context.Context, command uint32, payload []byte) error {
	// Encode the message using protocol first (before acquiring lock)
	data, err := protocol.Encode(command, payload)
	if err != nil {
		return fmt.Errorf("%s: %w", kephasnet.ErrFailedToEncode, err)
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf(kephasnet.ErrConnectionClosed)
	}

	// Keep the lock while sending to prevent race with Close()
	select {
	case c.sendCh <- data:
		c.mu.RUnlock()
		return nil
	case <-ctx.Done():
		c.mu.RUnlock()
		return ctx.Err()
	case <-c.ctx.Done():
		c.mu.RUnlock()
		return fmt.Errorf(kephasnet.ErrContextCancelled)
	}
}

// Close closes the client connection
func (c *Client) Close(ctx context.Context) error {
	return c.CloseWithCode(ctx, websocket.CloseNormalClosure, "")
}

// CloseWithCode closes the connection with a close code and optional reason
func (c *Client) CloseWithCode(ctx context.Context, code int, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel()

	// Send close message
	message := websocket.FormatCloseMessage(code, reason)
	deadline := time.Now().Add(time.Second)
	c.conn.WriteControl(websocket.CloseMessage, message, deadline)

	close(c.sendCh)
	return c.conn.Close()
}

// IsAlive returns true if the connection is still active
func (c *Client) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.closed
}

// CheckRateLimit checks if the client has exceeded the rate limit
// Returns true if the message is allowed, false if rate limited
func (c *Client) CheckRateLimit(ctx context.Context) bool {
	if c.rateLimiter == nil {
		// Rate limiting disabled
		return true
	}
	return c.rateLimiter.Allow()
}

// writePump pumps messages from the send channel to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// SetPongHandler sets the handler for pong messages
func (c *Client) SetPongHandler(handler func(appData string) error) {
	c.conn.SetPongHandler(handler)
}

// SetCloseHandler sets the handler for close messages
func (c *Client) SetCloseHandler(handler func(code int, text string) error) {
	c.conn.SetCloseHandler(handler)
}
