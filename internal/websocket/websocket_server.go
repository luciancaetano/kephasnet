package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/luciancaetano/knet"
	"github.com/luciancaetano/knet/internal/protocol"
)

// CheckOriginFn is a function that validates the origin of a WebSocket connection request.
// It receives the HTTP request and returns true if the origin is allowed, false otherwise.
// Use this to implement CORS policies for your WebSocket server.
type CheckOriginFn = func(r *http.Request) bool

// OnConnectFn is a callback function that is called when a new client connects.
// It is called after the WebSocket handshake completes and before the message
// reading loop starts. This is the ideal place to:
//   - Track connected clients
//   - Send welcome messages
//   - Perform authentication or authorization
//   - Initialize client-specific state
//
// The callback receives the client instance which can be used to send messages
// or access client information (ID, remote address, context).
//
// Note: This function is called synchronously during connection setup.
// Avoid long-running operations that could block new connections.
type OnConnectFn = func(client knet.Client)

// OnClientDisconnectFn is a callback type invoked when a connected client disconnects from the server.
// The function receives the disconnected client and a boolean that is true when the disconnect was
// initiated by the client (voluntary), and false for unexpected or server-initiated disconnects.
// Implementations can use this hook to perform cleanup, logging, resource reclamation, or
// application-specific notification when a client connection ends.
type OnClientDisconnectFn = func(client knet.Client, voluntary bool)

type ServerConfig struct {
	Addr               string
	RateLimitConfig    *RateLimitConfig
	CheckOrigin        CheckOriginFn
	OnConnect          OnConnectFn
	OnClientDisconnect OnClientDisconnectFn
}

// RateLimitConfig defines rate limiting configuration for clients
type RateLimitConfig struct {
	// MessagesPerSecond defines how many messages a client can send per second
	MessagesPerSecond rate.Limit
	// Burst defines the maximum burst size (token bucket capacity)
	Burst int
	// Enabled determines if rate limiting is active
	Enabled bool
}

// DefaultRateLimitConfig returns the default rate limit configuration
// Allows 100 messages per second with burst of 200
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		MessagesPerSecond: 100,
		Burst:             200,
		Enabled:           true,
	}
}

// NoRateLimit returns a configuration with rate limiting disabled
func NoRateLimit() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: false,
	}
}

// Server implements the WebsocketServer interface
type Server struct {
	addr     string
	server   *http.Server
	clients  sync.Map // map[string]*Client
	handlers sync.Map // map[uint32]func(client knet.Client, payload []byte)

	// JSON-RPC handlers (converted to protocol messages internally)
	jsonRPCHandlers sync.Map // map[string]func(params map[string]interface{}) (interface{}, error)

	// Rate limiting configuration
	rateLimitConfig *RateLimitConfig

	mu           sync.RWMutex
	running      bool
	upgrader     websocket.Upgrader
	onConnect    OnConnectFn
	onDisconnect OnClientDisconnectFn
}

// New creates a new WebSocket server instance with the specified configuration.
//
// Parameters:
//   - addr: The network address to listen on (e.g., ":8080" or "localhost:8080")
//   - rateLimitConfig: Rate limiting configuration. If nil, DefaultRateLimitConfig() is used.
//   - checkOrigin: Function to validate WebSocket connection origins (CORS).
//     Return true to allow the connection, false to reject it.
//   - onConnect: Optional callback called when a client connects. Can be nil.
//     Called after handshake but before message reading starts.
//
// The server uses the Gorilla WebSocket library with read/write buffer sizes of 1024 bytes.
// Rate limiting is applied per-client using a token bucket algorithm.
//
// Example:
//
//	server := New(":8080", DefaultRateLimitConfig(),
//	    func(r *http.Request) bool { return true },
//	    func(client knet.Client) {
//	        log.Printf("Client connected: %s", client.ID())
//	    })
func New(cfg *ServerConfig) *Server {
	if cfg.RateLimitConfig == nil {
		cfg.RateLimitConfig = DefaultRateLimitConfig()
	}
	return &Server{
		addr:            cfg.Addr,
		rateLimitConfig: cfg.RateLimitConfig,
		onConnect:       cfg.OnConnect,
		onDisconnect:    cfg.OnClientDisconnect,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     cfg.CheckOrigin,
		},
	}
}

// Start starts the WebSocket server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf(knet.ErrServerAlreadyRunning)
	}
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Check for immediate startup errors with a small timeout
	select {
	case err := <-errChan:
		// Reset running state without calling Stop to avoid deadlock
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	case <-ctx.Done():
		// Context cancelled, stop the server
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Stop(stopCtx)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully, no immediate errors
		return nil
	}
}

// Stop stops the WebSocket server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	// Close all client connections
	s.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			client.Close(ctx)
		}
		return true
	})

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// RegisterHandler registers a handler for a specific command ID
// The handler is executed asynchronously and receives the client and payload
func (s *Server) RegisterHandler(ctx context.Context, commandID uint32, handler func(client knet.Client, payload []byte)) error {
	s.handlers.Store(commandID, handler)
	return nil
}

// RegisterJSONRPCHandler registers a JSON-RPC handler for a specific method
// Internally, JSON-RPC requests are converted to protocol messages
// This uses the reserved command ID net.CmdJSONRPC
func (s *Server) RegisterJSONRPCHandler(ctx context.Context, method string, handler func(params map[string]interface{}) (interface{}, error)) error {
	s.jsonRPCHandlers.Store(method, handler)
	return nil
}

// handleWebSocket handles incoming WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
		return
	}

	client := NewClient(conn, r.RemoteAddr, s.rateLimitConfig)
	s.clients.Store(client.ID(), client)

	// Start reading messages from client
	go s.handleClient(client)
}

// handleClient handles messages from a connected client
func (s *Server) handleClient(client *Client) {
	defer func() {
		voluntary := client.Context().Err() == context.Canceled

		if s.onDisconnect != nil {
			s.onDisconnect(client, voluntary)
		}
		s.clients.Delete(client.ID())
		client.Close(context.Background())
	}()

	// Set read deadline to prevent indefinite blocking
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Set pong handler to reset read deadline on pong
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Call onConnect callback if provided
	// This is the ideal place to send welcome messages, track connections,
	// or perform initial authentication
	if s.onConnect != nil {
		s.onConnect(client)
	}

	for {
		select {
		case <-client.Context().Done():
			return
		default:
			_, data, err := client.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("Unexpected WebSocket close error: %v\n", err)
				}
				return
			}

			// Reset read deadline after successful read
			client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Check rate limit before processing message
			if !client.CheckRateLimit(context.Background()) {
				// Rate limit exceeded, send error and close connection
				fmt.Printf("Warn: Rate limit exceeded for client client_id=%s remote_addr=%s\n", client.ID(), client.RemoteAddr())
				client.CloseWithCode(context.Background(), websocket.ClosePolicyViolation, "Rate limit exceeded")
				return
			}

			// Decode protocol message
			commandID, payload, err := protocol.Decode(data)
			if err != nil {
				// Invalid protocol message, close connection
				client.CloseWithCode(context.Background(), websocket.CloseProtocolError, knet.ErrInvalidMessageFormat)
				return
			}

			// Handle the message
			s.handleProtocolMessage(client, commandID, payload)
		}
	}
}

// handleProtocolMessage handles binary protocol messages
// Handlers are executed in separate goroutines to avoid blocking the read loop
func (s *Server) handleProtocolMessage(client *Client, commandID uint32, payload []byte) {
	// Check if this is a JSON-RPC command (reserved command ID)
	if commandID == knet.CmdJSONRPC {
		// JSON-RPC also handled in goroutine
		go s.handleJSONRPCMessage(client, payload)
		return
	}

	// Handle normal protocol command
	if handler, ok := s.handlers.Load(commandID); ok {
		if handlerFunc, ok := handler.(func(knet.Client, []byte)); ok {
			// Execute handler in goroutine (async, client decides if/when to respond)
			go handlerFunc(client, payload)
		}
	}
	// Note: Unknown commands are silently ignored (fire-and-forget pattern)
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ID      interface{}            `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// handleJSONRPCMessage handles JSON-RPC messages encoded in protocol format
func (s *Server) handleJSONRPCMessage(client *Client, payload []byte) {
	var req JSONRPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		s.sendJSONRPCError(client, nil, knet.JSONRPCParseError, knet.ErrParseError, nil)
		return
	}

	if req.JSONRPC != knet.JSONRPCVersion {
		s.sendJSONRPCError(client, req.ID, knet.JSONRPCInvalidRequest, knet.ErrInvalidRequest, nil)
		return
	}

	handler, ok := s.jsonRPCHandlers.Load(req.Method)
	if !ok {
		s.sendJSONRPCError(client, req.ID, knet.JSONRPCMethodNotFound, knet.ErrMethodNotFound, nil)
		return
	}

	handlerFunc, ok := handler.(func(params map[string]interface{}) (interface{}, error))
	if !ok {
		s.sendJSONRPCError(client, req.ID, knet.JSONRPCInternalError, knet.ErrInternalError, nil)
		return
	}

	result, err := handlerFunc(req.Params)
	if err != nil {
		s.sendJSONRPCError(client, req.ID, knet.JSONRPCInternalError, err.Error(), nil)
		return
	}

	response := JSONRPCResponse{
		JSONRPC: knet.JSONRPCVersion,
		Result:  result,
		ID:      req.ID,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		s.sendJSONRPCError(client, req.ID, knet.JSONRPCInternalError, knet.ErrInternalError, nil)
		return
	}

	// Send JSON-RPC response
	client.Send(context.Background(), knet.CmdJSONRPC, responseData)
}

// sendJSONRPCError sends a JSON-RPC error response encoded in protocol format
func (s *Server) sendJSONRPCError(client *Client, id interface{}, code int, message string, data interface{}) {
	response := JSONRPCResponse{
		JSONRPC: knet.JSONRPCVersion,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		// Log error but continue - marshal errors are rare
		fmt.Printf("Failed to marshal JSON-RPC error response: %v", err)
		return
	}

	if err := client.Send(context.Background(), knet.CmdJSONRPC, responseData); err != nil {
		// Log error - client may have disconnected
		fmt.Printf("Failed to send JSON-RPC error response to client %s: %v", client.ID(), err)
	}
}

// GetClient returns a client by ID
func (s *Server) GetClient(id string) (*Client, bool) {
	if client, ok := s.clients.Load(id); ok {
		return client.(*Client), true
	}
	return nil, false
}

// SendToClient sends a protocol message to a specific client
func (s *Server) SendToClient(ctx context.Context, clientID string, commandID uint32, payload []byte) error {
	client, ok := s.GetClient(clientID)
	if !ok {
		return fmt.Errorf("%s: %s", knet.ErrClientNotFound, clientID)
	}

	return client.Send(ctx, commandID, payload)
}

// BroadcastCommand sends a command to all connected clients
func (s *Server) BroadcastCommand(ctx context.Context, commandID uint32, payload []byte) error {
	s.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			client.Send(ctx, commandID, payload)
		}
		return true
	})
	return nil
}
