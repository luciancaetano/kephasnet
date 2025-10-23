package knet

// Reserved command IDs for internal use.
const (
	// CmdJSONRPC is reserved for JSON-RPC 2.0 messages
	CmdJSONRPC      uint32 = 0xFFFFFFFF
	CmdJSONRPCError uint32 = 0xFFFFFFFE
)

// Standard error messages
const (
	// Protocol errors
	ErrInvalidMessageFormat = "Invalid message format"
	ErrUnknownCommand       = "unknown command"
	ErrParseError           = "Parse error"
	ErrInvalidRequest       = "Invalid Request"
	ErrMethodNotFound       = "Method not found"
	ErrInternalError        = "Internal error"

	// Connection errors
	ErrClientNotFound       = "client not found"
	ErrConnectionClosed     = "client connection is closed"
	ErrContextCancelled     = "client context cancelled"
	ErrFailedToEncode       = "failed to encode message"
	ErrServerAlreadyRunning = "server already running"
)

// JSON-RPC error codes (following JSON-RPC 2.0 specification)
const (
	JSONRPCParseError     = -32700
	JSONRPCInvalidRequest = -32600
	JSONRPCMethodNotFound = -32601
	JSONRPCInvalidParams  = -32602
	JSONRPCInternalError  = -32603
)

// JSON-RPC version
const (
	JSONRPCVersion = "2.0"
)
