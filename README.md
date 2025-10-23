# 🚀 KephasNet

[![Go Reference](https://pkg.go.dev/badge/github.com/luciancaetano/kephasnet.svg)](https://pkg.go.dev/github.com/luciancaetano/kephasnet)
[![Go Report Card](https://goreportcard.com/badge/github.com/luciancaetano/kephasnet)](https://goreportcard.com/report/github.com/luciancaetano/kephasnet)

A **high-performance** Go library for building game servers and real-time applications using WebSocket communication. Implements a command pattern protocol with efficient binary encoding and optional JSON-RPC 2.0 support.

## 📖 Overview

KephasNet provides a robust WebSocket server framework designed for game servers and real-time applications that require:

- **🎯 Command Pattern Protocol**: Efficient binary protocol with 4-byte command IDs + binary payload
- **📡 JSON-RPC 2.0 Support**: Optional JSON-RPC handler registration for standard RPC workflows
- **🛡️ Per-Client Rate Limiting**: Token bucket algorithm to prevent DoS attacks
- **⚡ Zero-Copy Performance**: Payload slices reference the original buffer for maximum efficiency
- **🏭 Production-Ready**: Built-in timeouts, protections, and security features
- **🔌 Connection Callbacks**: Track connections with `OnConnect` callbacks
- **📊 Broadcasting**: Send messages to all connected clients efficiently

## ✨ Key Features

- 📦 **Lightweight binary protocol**: 4 bytes (CommandID uint32 big-endian) + binary payload
- 🎯 **Command pattern architecture** for extensible message handling
- 🔄 **Optional JSON-RPC 2.0 support** for standard RPC calls
- 🛡️ **Per-client rate limiting** (token bucket algorithm)
- ⚡ **Zero-copy decode** (payload slices the original buffer — DO NOT modify)
- ⏱️ **Timeouts and protections** (read/write/pong, payload limits, race protections)
- 🔌 **OnConnect callbacks** to track and manage client connections
- 📊 **Broadcasting support** to send messages to all clients
- 🔐 **Origin validation** for CORS security

## 📦 Installation

```bash
go get github.com/luciancaetano/kephasnet
```

**Dependencies:**
```bash
go get github.com/gorilla/websocket
go get golang.org/x/time/rate
go get github.com/google/uuid
```

Or use the module system (recommended):
```bash
go mod init your-project
go get github.com/luciancaetano/kephasnet
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/luciancaetano/kephasnet"
    "github.com/luciancaetano/kephasnet/ws"
)

func main() {
    ctx := context.Background()

    // Configure rate limiting (100 messages/sec, burst 200)
    rateLimitConfig := ws.DefaultRateLimitConfig()

    // Create server with OnConnect callback
    server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), func(client kephasnet.Client) {
        log.Printf("✅ Client connected: %s from %s", client.ID(), client.RemoteAddr())
        
        // Send welcome message
        welcomeMsg := []byte("Welcome to the server!")
        client.Send(ctx, 0x0001, welcomeMsg)
    })

    // Register a command handler (0x0100 = player login)
    err := server.RegisterHandler(ctx, 0x0100, func(payload []byte) ([]byte, error) {
        log.Printf("📨 Received login request: %s", string(payload))
        return []byte("Login successful"), nil
    })
    if err != nil {
        log.Fatal(err)
    }

    // Start the server
    log.Println("🚀 Starting server on :8080...")
    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Graceful shutdown example
    time.Sleep(10 * time.Minute)
    log.Println("Shutting down...")
    _ = server.Stop(ctx)
}
```

## 📡 Protocol

### Command Pattern Binary Protocol

The library uses a lightweight binary protocol based on the command pattern:

**Message Format:**
```
┌────────────────┬──────────────────────┐
│  4 bytes       │  N bytes             │
│  CommandID     │  Payload             │
│  (uint32 BE)   │  (binary data)       │
└────────────────┴──────────────────────┘
```

**Example:** 
- CommandID `0x01` + payload `"hello"` 
- Bytes: `[0x00, 0x00, 0x00, 0x01, 'h', 'e', 'l', 'l', 'o']`

This command pattern allows you to register specific handlers for each command ID, making it easy to build game servers with different message types:
- `0x0100-0x01FF`: Player actions (movement, combat, etc.)
- `0x0200-0x02FF`: Chat messages
- `0x0300-0x03FF`: Inventory operations
- `0x0400-0x04FF`: Game state updates
- And so on...

### JSON-RPC 2.0 Support

In addition to the binary command protocol, the library provides optional JSON-RPC 2.0 support for standard RPC workflows. JSON-RPC messages use reserved command IDs and follow the [JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification).

**Reserved Command IDs for JSON-RPC:**
- `0xFFFFFFFF`: JSON-RPC requests/responses
- `0xFFFFFFFE`: JSON-RPC error responses

**Available Command IDs for your application:** `0x00000000` through `0xFFFFFFFD`

## 🏗️ Architecture Features

- ⚡ **Ultra-efficient binary protocol** (4-byte overhead per message)
- 🔄 **Asynchronous handlers** (run in goroutines; don't block the read loop)
- 📋 **Zero-copy decode** (payload is a slice of the read buffer — DO NOT modify)
- 🛡️ **Per-client rate limiting** (token bucket algorithm)
- ⏱️ **Timeouts**: Read 60s (renewed on each message), Write 10s, Ping every 54s
- 📏 **Default max payload**: 10MB
- 🔌 **Connection callbacks**: Track client lifecycle with `OnConnect`
- 📊 **Broadcasting**: Send to all connected clients with `BroadcastCommand`
- 🔐 **Optional JSON-RPC 2.0 support** (via dedicated handlers)

## 🔧 Server Configuration Examples

### Default Rate Limiting
```go
// 100 messages/second, burst 200
rateLimitConfig := ws.DefaultRateLimitConfig()
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```

### No Rate Limiting
```go
rateLimitConfig := ws.NoRateLimit()
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```

### Custom Rate Limiting
```go
rateLimitConfig := &ws.RateLimitConfig{
    MessagesPerSecond: 50,  // 50 messages per second
    Burst:             100, // Allow bursts up to 100 messages
    Enabled:           true,
}
server := ws.New(":8080", rateLimitConfig, ws.AllOrigins(), nil)
```

### Custom Origin Check (Production)
```go
checkOrigin := func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowedOrigins := []string{
        "https://yourdomain.com",
        "https://www.yourdomain.com",
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
server := ws.New(":8080", ws.DefaultRateLimitConfig(), checkOrigin, nil)
```

⚠️ **Warning**: Never use `ws.AllOrigins()` in production! It allows connections from any origin.

## 🔌 Connection Lifecycle

### OnConnect Callback

The `OnConnect` callback is invoked when a new client successfully connects to the server. It executes after the WebSocket handshake completes but before the message reading loop starts.

```go
server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(), 
    func(client kephasnet.Client) {
        log.Printf("🎉 New connection: ID=%s, RemoteAddr=%s", client.ID(), client.RemoteAddr())
        
        // Send a welcome message
        welcomeMsg := []byte("Welcome to the server!")
        client.Send(context.Background(), 0x0001, welcomeMsg)
    })
```

**Common Use Cases:**
- ✅ **Track connections**: Add client to a registry for broadcasting
- 💬 **Send welcome messages**: Greet the client or send initial state
- 🔐 **Authentication**: Verify credentials before accepting messages
- 🎮 **Initialize state**: Set up client-specific data structures
- 📊 **Metrics**: Track connection counts and rates

**Important Notes:**
- The callback is optional (can be `nil`)
- Runs synchronously during connection setup
- Avoid long-running operations that could block new connections
- The client is already added to the server's internal client map
- The client's context is active and can be used for cleanup tracking

### Connection Tracking Example

Track all connected clients and handle disconnections gracefully:

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
            log.Printf("👋 Client disconnected: %s", client.ID())
        }()
        
        log.Printf("✅ Client connected: %s (Total: %d)", client.ID(), len(clients))
    })
```

## 🎯 Message Handlers

### Binary Command Handlers

Register handlers using the command pattern - each command ID maps to a specific handler function:

```go
// Example: Player movement command (ID 0x0100)
server.RegisterHandler(ctx, 0x0100, func(payload []byte) ([]byte, error) {
    if len(payload) == 0 {
        return nil, fmt.Errorf("empty payload")
    }
    
    // Process movement data
    position := processMovement(payload)
    
    // Return response (will be sent back with same command ID)
    return position, nil
})

// Example: Chat message command (ID 0x0200)
server.RegisterHandler(ctx, 0x0200, func(payload []byte) ([]byte, error) {
    chatMsg := processChatMessage(payload)
    
    // Broadcast to all clients
    server.BroadcastCommand(ctx, 0x0200, chatMsg)
    
    // Return nil if you don't want to send a response to this specific client
    return nil, nil
})

// Example: Player stats request (ID 0x0300)
server.RegisterHandler(ctx, 0x0300, func(payload []byte) ([]byte, error) {
    playerID := string(payload)
    stats := getPlayerStats(playerID)
    
    // Marshal stats to JSON
    statsJSON, err := json.Marshal(stats)
    if err != nil {
        return nil, err
    }
    
    return statsJSON, nil
})
```

### JSON-RPC Handlers (Optional)

For standard RPC-style communication, register JSON-RPC 2.0 handlers:

```go
// Register a JSON-RPC method for getting player stats
server.RegisterJSONRPCHandler(ctx, "player.getStats", func(params map[string]interface{}) (interface{}, error) {
    playerID := params["playerId"].(string)
    stats := getPlayerStats(playerID)
    return stats, nil
})

// Register a JSON-RPC calculation method
server.RegisterJSONRPCHandler(ctx, "math.add", func(params map[string]interface{}) (interface{}, error) {
    a := params["a"].(float64)
    b := params["b"].(float64)
    return a + b, nil
})

// Register a complex method with validation
server.RegisterJSONRPCHandler(ctx, "game.createRoom", func(params map[string]interface{}) (interface{}, error) {
    roomName, ok := params["roomName"].(string)
    if !ok || roomName == "" {
        return nil, fmt.Errorf("invalid room name")
    }
    
    maxPlayers, ok := params["maxPlayers"].(float64)
    if !ok || maxPlayers < 2 {
        return nil, fmt.Errorf("invalid max players")
    }
    
    room := createGameRoom(roomName, int(maxPlayers))
    return room, nil
})
```

**Important Notes:**
- ⚠️ Handlers run in separate goroutines — no ordering guarantees
- ⚠️ DO NOT modify the payload slice (zero-copy)
- ✅ Return `nil` response if you don't want to send a reply
- ✅ Return an error to send an error response to the client

## 🛡️ Security & Limits

### Rate Limiting

Per-client rate limiting using token bucket algorithm:

- ✅ Implemented per-client (independent limits)
- 🔒 Clients exceeding limits receive close code `1008` (Policy Violation)
- ⚙️ Configure via `RateLimitConfig` when creating the server
- 📊 Default: 100 messages/second, burst 200
- 🚫 Can be disabled with `ws.NoRateLimit()`

When a client exceeds the rate limit:
1. Server logs: `"Rate limit exceeded for client client_id=xxx remote_addr=xxx"`
2. Connection is closed with code `1008` (Policy Violation)
3. Client receives the close reason: `"Rate limit exceeded"`

### Security Features

| Feature | Default | Description |
|---------|---------|-------------|
| **Max Payload** | 10MB | Prevents OOM attacks |
| **Read Timeout** | 60s | Renewed per message, prevents hanging |
| **Write Timeout** | 10s | Prevents slow clients from blocking |
| **Ping Interval** | 54s | Automatic keepalive |
| **Pong Handler** | Auto | Detects dead connections |
| **Origin Validation** | Custom | Configure via `CheckOriginFn` |
| **Rate Limiting** | 100 msg/s | Per-client DoS prevention |

### Origin Validation Example

**⚠️ NEVER use `ws.AllOrigins()` in production!**

```go
// Production-ready origin check
checkOrigin := func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowed := []string{
        "https://yourdomain.com",
        "https://www.yourdomain.com",
        "https://app.yourdomain.com",
    }
    for _, allowedOrigin := range allowed {
        if origin == allowedOrigin {
            return true
        }
    }
    return false
}
server := ws.New(":8080", ws.DefaultRateLimitConfig(), checkOrigin, nil)
```

## 📁 Project Structure

```
go-kephas-net/
├── README.md                  # This file
├── LICENSE                    # MIT License
├── Makefile                   # Build commands
├── go.mod                     # Go module definition
├── doc.go                     # Package documentation
├── kephasnet.go              # Public interfaces (WebsocketServer, Client)
├── commands.go               # Constants (command IDs, errors)
│
├── internal/                 # Internal implementation (not part of public API)
│   ├── protocol/            
│   │   └── protocol.go       # Binary encoding/decoding (Encode/Decode)
│   └── websocket/
│       ├── websocket_server.go  # Server implementation
│       └── websocket_client.go  # Client implementation
│
├── ws/                       # Public factory package
│   └── server.go             # Factory functions (New, DefaultRateLimitConfig, etc.)
│
├── examples/                 # Example applications
│   └── js-chat/              # JavaScript chat example
│       ├── main.go           # Go server
│       ├── index.html        # Chat UI
│       ├── kephas-client.js  # JS client library
│       └── go.mod
│
└── tests/                    # Test suites
    ├── unit/                 # Unit tests
    ├── e2e/                  # End-to-end tests
    └── stress/               # Stress/load tests
        └── README.md         # Stress testing guide
```

**Key Design Principles:**
- 📦 **Use `ws.New()` for server creation** (implementation is internal)
- 🔒 **Internal packages are not part of the public API**
- 📝 **Public interfaces defined in `kephasnet.go`**
- 🏭 **Factory pattern for clean API surface**

## 🎮 Use Cases

This library is ideal for:

| Use Case | Why KephasNet? |
|----------|---------------|
| **🎮 Game Servers** | Command pattern perfect for different message types (movement, combat, chat, inventory) |
| **📊 Real-time Dashboards** | Low-latency binary protocol for high-frequency updates |
| **🤝 Collaboration Tools** | Broadcasting support for multi-user interactions |
| **💹 Trading Platforms** | Efficient binary encoding for market data streams |
| **🏠 IoT Command & Control** | Command-based protocol ideal for device management |
| **🔄 Microservices** | Low-latency service-to-service communication |
| **🔀 Hybrid Systems** | Both binary commands and JSON-RPC for flexibility |

## 💡 Best Practices

### ✅ Do's

1. **Always use `ws.New()` for server creation** - implementation lives in `internal/`
2. **Register handlers before calling `Start()`** - handlers cannot be added after server starts
3. **Use context for cancellation/timeouts** - especially in handlers
4. **Design your command ID space thoughtfully**:
   - `0x0100-0x01FF`: Player actions
   - `0x0200-0x02FF`: Chat/social
   - `0x0300-0x03FF`: Inventory/items
   - `0x0400-0x04FF`: Game state
5. **Use binary commands for high-frequency messages** (game state updates)
6. **Use JSON-RPC for administrative operations** (configuration, stats)
7. **Configure rate limiting appropriately** for your use case
8. **Use custom `CheckOriginFn` in production** - never allow all origins
9. **Track client connections** using `OnConnect` callback
10. **Monitor logs** for "Rate limit exceeded" and other warnings

### ❌ Don'ts

1. **DO NOT modify the payload slice** returned by handlers (it's zero-copy)
2. **DO NOT use `ws.AllOrigins()` in production** - security risk
3. **DO NOT use reserved command IDs**:
   - `0xFFFFFFFF` - JSON-RPC requests
   - `0xFFFFFFFE` - JSON-RPC errors
4. **DO NOT perform long-running operations** in `OnConnect` callback
5. **DO NOT assume handler execution order** - they run concurrently
6. **DO NOT ignore rate limiting** - always configure appropriate limits

## 🚀 Advanced Examples

### Broadcasting to All Clients

```go
// Maintain a client registry
var (
    clientsMu sync.RWMutex
    clients   = make(map[string]kephasnet.Client)
)

// Track clients via OnConnect
server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(),
    func(client kephasnet.Client) {
        clientsMu.Lock()
        clients[client.ID()] = client
        clientsMu.Unlock()
        
        go func() {
            <-client.Context().Done()
            clientsMu.Lock()
            delete(clients, client.ID())
            clientsMu.Unlock()
        }()
    })

// Manual broadcast function
func broadcastToAll(cmdID uint32, payload []byte) {
    clientsMu.RLock()
    defer clientsMu.RUnlock()
    
    for _, client := range clients {
        if client.IsAlive() {
            go client.Send(context.Background(), cmdID, payload)
        }
    }
}

// Or use built-in broadcast
server.BroadcastCommand(ctx, 0x0200, []byte("Server announcement"))
```

### Handler with Timeout

```go
server.RegisterHandler(ctx, 0x0500, func(payload []byte) ([]byte, error) {
    // Create context with timeout for this handler
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Process with timeout
    result, err := processWithTimeout(ctx, payload)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("operation timed out")
        }
        return nil, err
    }
    
    return result, nil
})
```

### Advanced Connection Tracking

```go
type ClientInfo struct {
    Client      kephasnet.Client
    Username    string
    ConnectedAt time.Time
    LastSeen    time.Time
}

var (
    clientsMu    sync.RWMutex
    clientsInfo  = make(map[string]*ClientInfo)
)

server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(),
    func(client kephasnet.Client) {
        info := &ClientInfo{
            Client:      client,
            ConnectedAt: time.Now(),
            LastSeen:    time.Now(),
        }
        
        clientsMu.Lock()
        clientsInfo[client.ID()] = info
        clientsMu.Unlock()
        
        // Cleanup on disconnect
        go func() {
            <-client.Context().Done()
            clientsMu.Lock()
            delete(clientsInfo, client.ID())
            clientsMu.Unlock()
            log.Printf("Client %s disconnected after %v", 
                client.ID(), time.Since(info.ConnectedAt))
        }()
    })

// Update last seen timestamp
func updateLastSeen(clientID string) {
    clientsMu.Lock()
    defer clientsMu.Unlock()
    
    if info, exists := clientsInfo[clientID]; exists {
        info.LastSeen = time.Now()
    }
}
```

### Graceful Shutdown

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    server := ws.New(":8080", ws.DefaultRateLimitConfig(), ws.AllOrigins(), nil)
    
    // Register handlers...
    server.RegisterHandler(ctx, 0x01, myHandler)
    
    // Start server in goroutine
    go func() {
        if err := server.Start(ctx); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan
    
    log.Println("Shutting down gracefully...")
    
    // Give clients 5 seconds to disconnect
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    
    if err := server.Stop(shutdownCtx); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
    
    log.Println("Server stopped")
}
```

## 💬 Running the Chat Example

A complete, production-ready chat application example is available in the `examples/js-chat` directory. It demonstrates real-world usage with both Go server and JavaScript client.

### 🌟 Features

- ✅ Real-time chat messaging
- 👤 User authentication
- 📢 Message broadcasting
- 🎨 Modern responsive UI
- 📚 JavaScript client library (`kephas-client.js`)
- 🔄 Auto-reconnection on connection loss
- 🛡️ Rate limiting protection

### 🚀 Quick Start

1. **Navigate to the example directory:**
   ```bash
   cd examples/js-chat
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Start the server:**
   ```bash
   go run main.go
   ```
   
   The server will start on:
   - WebSocket: `ws://localhost:8080/ws`
   - HTTP/Static: `http://localhost:3000`

4. **Open the chat client:**
   - Navigate to `http://localhost:3000` in your browser
   - Enter a username and click "Connect"
   - Open multiple browser tabs/windows to test multi-user chat

### 📁 Example Structure

```
examples/js-chat/
├── main.go              # Go WebSocket server with chat logic
├── index.html           # Chat UI (HTML/CSS/JS)
├── kephas-client.js     # JavaScript client library
├── go.mod               # Go module dependencies
└── go.sum               # Dependency checksums
```

### 🔍 Key Implementation Details

The chat example demonstrates:

| Feature | Command ID | Description |
|---------|-----------|-------------|
| **Chat Messages** | `0x0001` | JSON messages with username, text, and timestamp |
| **User Joined** | `0x0003` | Notification when a user connects |
| **User Left** | `0x0004` | Notification when a user disconnects |
| **Get Users** | `0x0005` | Request list of online users |
| **Users List** | `0x0006` | Response with online users array |

**Server Implementation Highlights:**
- Command-based message routing
- Client tracking with connection callbacks
- Broadcasting to all connected clients
- Graceful disconnect handling
- Rate limiting (100 msg/s per client)

**Client Implementation Highlights:**
- Binary protocol encoding/decoding
- Automatic reconnection logic
- Event-driven message handling
- Clean separation of concerns

You can use this example as a **starting point** for building your own real-time applications! 🚀

## 🧪 Testing

The project includes comprehensive test suites to ensure reliability and performance.

### Running Tests

```bash
# Run all tests
make test

# Or using go test directly
go test ./tests/... -v

# Run unit tests only
go test ./tests/unit/... -v

# Run end-to-end tests
go test ./tests/e2e/... -v

# Run with coverage
go test ./tests/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Stress Tests

Stress tests validate performance under high load conditions. See [`tests/stress/README.md`](tests/stress/README.md) for detailed information.

**Quick stress test:**
```bash
cd tests/stress

# May need to increase file descriptor limit
ulimit -n 65536

# Run all stress tests (can take 15-30 minutes)
go test -v -timeout 30m

# Run specific stress test
go test -v -run TestStress5000Connections -timeout 10m
```

**Stress test scenarios:**
- ✅ 5,000 simultaneous connections
- ✅ 10,000 simultaneous connections
- ✅ 100,000 messages (100 clients × 1,000 messages)
- ✅ Message latency and throughput benchmarks

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

### How to Contribute

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Make your changes**
   - Write tests for new features
   - Update documentation
   - Follow existing code style
4. **Run tests**
   ```bash
   make test
   ```
5. **Commit your changes**
   ```bash
   git commit -m "Add amazing feature"
   ```
6. **Push to the branch**
   ```bash
   git push origin feature/amazing-feature
   ```
7. **Open a Pull Request**

### Development Guidelines

- ✅ Write tests for all new features
- ✅ Update documentation (README, godoc comments)
- ✅ Follow Go best practices and conventions
- ✅ Keep the public API minimal and clean
- ✅ Add examples for new features
- ✅ Ensure all tests pass before submitting PR

### Reporting Issues

Found a bug? Have a feature request? Please [open an issue](https://github.com/luciancaetano/kephasnet/issues) with:
- Clear description of the problem/feature
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Go version and OS
- Code samples if applicable

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with these excellent libraries:
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) - Rate limiting
- [google/uuid](https://github.com/google/uuid) - UUID generation

## 📚 Additional Resources

- [📖 Go Package Documentation](https://pkg.go.dev/github.com/luciancaetano/kephasnet)
- [🐛 Issue Tracker](https://github.com/luciancaetano/kephasnet/issues)
- [💬 Discussions](https://github.com/luciancaetano/kephasnet/discussions)
- [📋 Changelog](https://github.com/luciancaetano/kephasnet/releases)

## ⭐ Support

If you find this library useful, please consider giving it a star on GitHub! ⭐

---

**Made with ❤️ for the Go community**
