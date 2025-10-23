# go-kephas-net

A high-performance Go library for building game servers and real-time applications using WebSocket communication. Implements a command pattern protocol with efficient binary encoding and optional JSON-RPC 2.0 support.

## Overview

This library provides a robust WebSocket server framework designed for game servers and real-time applications that require:
- **Command Pattern Protocol**: Efficient binary protocol with 4-byte command IDs + binary payload
- **JSON-RPC 2.0 Support**: Optional JSON-RPC handler registration for standard RPC workflows
- **Per-Client Rate Limiting**: Token bucket algorithm to prevent DoS attacks
- **Zero-Copy Performance**: Payload slices reference the original buffer for maximum efficiency
- **Production-Ready**: Built-in timeouts, protections, and security features

## Key Features

- Lightweight binary protocol: 4 bytes (CommandID uint32 big-endian) + binary payload
- Command pattern architecture for extensible message handling
- Optional JSON-RPC 2.0 support for standard RPC calls
- Per-client rate limiting (token bucket)
- Zero-copy decode (payload slices the original buffer — DO NOT modify)
- Timeouts and protections (read/write/pong, payload limits, race protections)

## Installation
```bash
go get github.com/luciancaetano/kephasnet
go get github.com/gorilla/websocket
go get golang.org/x/time/rate
```

## Quick start
```go
import (
    "context"
    "log"
    "time"

    "github.com/luciancaetano/kephasnet/ws"
)

func main() {
    ctx := context.Background()

    // Configure rate limiting
    rateLimitConfig := &ws.RateLimitConfig{
        MessagesPerSecond: 100,
        Burst:             200,
        Enabled:           true,
    }

    // Create server with rate limiting, origin check, and connection callback
    server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), func(client kephasnet.Client) {
        log.Printf("Client connected: %s from %s", client.ID(), client.RemoteAddr())
    })

    // Register a handler
    err := server.RegisterHandler(ctx, 0x0100, func(payload []byte) ([]byte, error) {
        log.Printf("Received: %s", string(payload))
        return []byte("ACK"), nil
    })
    if err != nil {
        log.Fatal(err)
    }

    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Graceful stop example
    time.Sleep(10 * time.Minute)
    _ = server.Stop(ctx)
}
```

## Protocol

### Command Pattern Binary Protocol

The library uses a lightweight binary protocol based on the command pattern:

**Message Format:**
- 4 bytes: CommandID (uint32, big-endian) - identifies the command/action
- N bytes: Payload (binary) - command-specific data

**Example:** CommandID 0x01 + payload "hello" => bytes: [0x00,0x00,0x00,0x01,'h','e','l','l','o']

This command pattern allows you to register specific handlers for each command ID, making it easy to build game servers with different message types (player movement, chat, inventory, etc.).

### JSON-RPC 2.0 Support

In addition to the binary command protocol, the library provides optional JSON-RPC 2.0 support for standard RPC workflows. JSON-RPC messages use reserved command IDs and follow the JSON-RPC 2.0 specification.

## Architecture Features

- **Command Pattern**: Register handlers for specific command IDs, perfect for game server message routing
- **Ultra-efficient binary protocol** (4-byte overhead per message)
- **Asynchronous handlers** (run in goroutines; do not block the read loop)
- **Zero-copy decode** (payload is a slice of the read buffer)
- **Per-client rate limiting** (token bucket algorithm)
- **Timeouts**: Read 60s (renewed on each message), Write 10s, Ping every 54s
- **Default max payload**: 10MB
- **Optional JSON-RPC 2.0 support** (via dedicated handlers for RPC-style communication)

## Creation examples
Default limits:
```go
rateLimitConfig := ws.DefaultRateLimitConfig() // 100 msg/s, burst 200
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```
No rate limiting:
```go
rateLimitConfig := ws.NoRateLimit()
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```
Custom server:
```go
rateLimitConfig := &ws.RateLimitConfig{
    MessagesPerSecond: 50,
    Burst:             100,
    Enabled:           true,
}
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```
Custom origin check:
```go
checkOrigin := func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    return origin == "https://yourdomain.com"
}
server := ws.New(":8080", ws.DefaultRateLimitConfig(), checkOrigin, nil)
```

## Connection Lifecycle

### OnConnect Callback

The `OnConnect` callback is called when a new client successfully connects to the server. It's executed after the WebSocket handshake completes but before the message reading loop starts.

```go
server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(), 
    func(client kephasnet.Client) {
        log.Printf("New connection: ID=%s, RemoteAddr=%s", client.ID(), client.RemoteAddr())
        
        // Send a welcome message
        welcomeMsg := []byte("Welcome to the server!")
        client.Send(context.Background(), 0x0001, welcomeMsg)
    })
```

**Common use cases:**
- **Track connections**: Add client to a registry for broadcasting
- **Send welcome messages**: Greet the client or send initial state
- **Authentication**: Verify credentials before accepting messages
- **Initialize state**: Set up client-specific data structures

**Important notes:**
- The callback is optional (can be `nil`)
- Runs synchronously during connection setup
- Avoid long-running operations that could block other connections
- The client is already added to the server's internal client map
- The client's context is active and can be used for cleanup tracking

### Connection Tracking Example

```go
var (
    clientsMu sync.RWMutex
    clients   = make(map[string]kephasnet.Client)
)

server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(),
    func(client kephasnet.Client) {
        // Add client to tracking map
        clientsMu.Lock()
        clients[client.ID()] = client
        clientsMu.Unlock()
        
        // Remove client when disconnected
        go func() {
            <-client.Context().Done()
            clientsMu.Lock()
            delete(clients, client.ID())
            clientsMu.Unlock()
            log.Printf("Client disconnected: %s", client.ID())
        }()
        
        log.Printf("Client connected: %s (Total: %d)", client.ID(), len(clients))
    })
```


## Handlers

### Binary Command Handlers

Register handlers using the command pattern - each command ID maps to a specific handler function:

```go
// Example: Player movement command (ID 0x0100)
server.RegisterHandler(ctx, 0x0100, func(payload []byte) ([]byte, error) {
    if len(payload) == 0 {
        return nil, fmt.Errorf("empty payload")
    }
    return processMovement(payload), nil
})

// Example: Chat message command (ID 0x0200)
server.RegisterHandler(ctx, 0x0200, func(payload []byte) ([]byte, error) {
    return processChatMessage(payload), nil
})
```

### JSON-RPC Handlers (Optional)

For standard RPC-style communication, register JSON-RPC 2.0 handlers:

```go
// Register a JSON-RPC method
server.RegisterJSONRPCHandler(ctx, "player.getStats", func(params map[string]interface{}) (interface{}, error) {
    playerID := params["playerId"].(string)
    stats := getPlayerStats(playerID)
    return stats, nil
})

// Register a JSON-RPC calculation method
server.RegisterJSONRPCHandler(ctx, "add", func(params map[string]interface{}) (interface{}, error) {
    a := params["a"].(float64)
    b := params["b"].(float64)
    return a + b, nil
})
```

Notes:
- Handlers run in separate goroutines — no ordering guarantees.
- Do NOT modify the payload slice (zero-copy).

## Reserved IDs
Internal use — do not use these IDs:
- 0xFFFFFFFF — JSON-RPC
- 0xFFFFFFFE — JSON-RPC Error

Available IDs: 0x00000000 through 0xFFFFFFFD

## Rate limiting
- Implemented per-client (token bucket). Clients that exceed limits receive close code 1008 (Policy Violation) and the connection is closed.
- Configure via `RateLimitConfig` struct when creating the server.
- Use `ws.DefaultRateLimitConfig()` for default settings (100 msg/s, burst 200)
- Use `ws.NoRateLimit()` to disable rate limiting

## Security & limits
- Default max payload: 10MB (constant in protocol.go)
- Default read timeout: 60s (renewed per message)
- Default write timeout: 10s
- Pong handler and automatic keepalive
- Protections against race conditions and deadlocks
- Origin validation: configure via `CheckOriginFn` parameter in `ws.New()`
- Use `ws.AllOrigins()` to allow all origins (development only)

Example custom origin check:
```go
checkOrigin := func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowed := []string{"https://yourdomain.com", "https://www.yourdomain.com"}
    for _, a := range allowed {
        if origin == a {
            return true
        }
    }
    return false
}
server := ws.New(":8080", ws.DefaultRateLimitConfig(), checkOrigin)
```

## Package layout
```
go-kephas-net/
├── README.md
├── doc.go
├── kephasnet.go           # Public interfaces
├── commands.go            # Constants
├── internal/
│   ├── protocol/
│   │   └── protocol.go    # Encode/Decode
│   └── websocket/
│       ├── websocket_server.go
│       └── websocket_client.go
└── ws/
    └── server.go          # Public factory (use ws.New)
```

## Use Cases

This library is ideal for:

- **Game Servers**: Multiplayer game backends with different message types (movement, combat, chat, inventory)
- **Real-time Applications**: Live dashboards, collaboration tools, trading platforms
- **IoT Command & Control**: Device management with command-based protocols
- **Microservices**: Low-latency service-to-service communication with binary efficiency
- **Hybrid Systems**: Applications needing both efficient binary commands and standard JSON-RPC calls

## Best practices
- Always create servers via `ws.New()` (implementation lives in internal/)
- Register handlers before calling `Start()`
- Use context for cancellation/timeouts in handlers
- Design your command ID space thoughtfully (e.g., 0x0100-0x01FF for player actions, 0x0200-0x02FF for chat)
- Use binary commands for high-frequency messages (game state updates) and JSON-RPC for administrative operations
- Configure rate limiting appropriately for your use case
- Use custom `CheckOriginFn` in production (never use `ws.AllOrigins()` in production)
- Monitor logs for "Rate limit exceeded" and other warnings
- DO NOT modify the payload slice returned by handlers (it's zero-copy)

## Advanced examples

### Broadcasting to all clients
```go
// Assuming you maintain a client list
func broadcastMessage(clients map[string]kephasnet.Client, cmdID uint32, payload []byte) {
    for _, client := range clients {
        if client.IsAlive() {
            _ = client.Send(context.Background(), cmdID, payload)
        }
    }
}
```

### Handler with timeout
```go
server.RegisterHandler(ctx, 0x01, func(payload []byte) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return processWithTimeout(ctx, payload)
})
```

### Client connection tracking
```go
// Track client connections
var clientsMu sync.RWMutex
var clients = make(map[string]kephasnet.Client)

// In your connection handler
func handleNewClient(client kephasnet.Client) {
    clientsMu.Lock()
    clients[client.ID()] = client
    clientsMu.Unlock()

    // Clean up on disconnect
    go func() {
        <-client.Context().Done()
        clientsMu.Lock()
        delete(clients, client.ID())
        clientsMu.Unlock()
    }()
}
```

## Running the Chat Example

A complete chat application example is available in the `examples/js-chat` directory, demonstrating real-world usage of the library with both Go server and JavaScript client.

### Features
- Real-time chat messaging
- User authentication
- Message broadcasting
- Modern responsive UI
- JavaScript client library (`kephas-client.js`)

### How to Run

1. **Navigate to the example directory:**
   ```bash
   cd examples/js-chat
   ```

2. **Start the server:**
   ```bash
   go run main.go
   ```
   
   The server will start on `http://localhost:8080`

3. **Open the chat client:**
   - Open your browser and navigate to `http://localhost:8080`
   - Enter a username and connect
   - Open multiple browser tabs/windows to test multi-user chat

### Example Structure
```
examples/js-chat/
├── main.go              # Go WebSocket server with chat logic
├── index.html           # Chat UI (HTML/CSS/JS)
├── kephas-client.js     # JavaScript client library
├── go.mod               # Go module dependencies
└── go.sum
```

### Key Implementation Details

The chat example demonstrates:
- **Command-based messaging**: Uses command ID `0x0001` for chat messages
- **JSON payloads**: Messages are sent as JSON with username, message, and timestamp
- **Broadcasting**: Server broadcasts messages to all connected clients
- **Client tracking**: Maintains a list of active clients and handles disconnections
- **Rate limiting**: Protects against message spam
- **Auto-reconnection**: Client automatically reconnects on connection loss

You can use this example as a starting point for building your own real-time applications.

