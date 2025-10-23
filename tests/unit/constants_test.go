package unit_test

import (
	"testing"

	"github.com/luciancaetano/knet"
)

// TestConstants verifies that all constants are defined with expected values
func TestConstants(t *testing.T) {
	t.Parallel()

	t.Run("command IDs", func(t *testing.T) {
		// Verify reserved command IDs are distinct
		if knet.CmdJSONRPC == knet.CmdJSONRPCError {
			t.Error("CmdJSONRPC and CmdJSONRPCError should be different")
		}

		// Verify they're using reserved high values
		if knet.CmdJSONRPC != 0xFFFFFFFF {
			t.Errorf("CmdJSONRPC = %v, want 0xFFFFFFFF", knet.CmdJSONRPC)
		}

		if knet.CmdJSONRPCError != 0xFFFFFFFE {
			t.Errorf("CmdJSONRPCError = %v, want 0xFFFFFFFE", knet.CmdJSONRPCError)
		}
	})

	t.Run("error messages", func(t *testing.T) {
		// Verify error messages are non-empty
		errorMessages := []struct {
			name  string
			value string
		}{
			{"ErrInvalidMessageFormat", knet.ErrInvalidMessageFormat},
			{"ErrUnknownCommand", knet.ErrUnknownCommand},
			{"ErrParseError", knet.ErrParseError},
			{"ErrInvalidRequest", knet.ErrInvalidRequest},
			{"ErrMethodNotFound", knet.ErrMethodNotFound},
			{"ErrInternalError", knet.ErrInternalError},
			{"ErrClientNotFound", knet.ErrClientNotFound},
			{"ErrConnectionClosed", knet.ErrConnectionClosed},
			{"ErrContextCancelled", knet.ErrContextCancelled},
			{"ErrFailedToEncode", knet.ErrFailedToEncode},
			{"ErrServerAlreadyRunning", knet.ErrServerAlreadyRunning},
		}

		for _, em := range errorMessages {
			t.Run(em.name, func(t *testing.T) {
				if em.value == "" {
					t.Errorf("%s should not be empty", em.name)
				}
			})
		}
	})

	t.Run("JSON-RPC error codes", func(t *testing.T) {
		// Verify JSON-RPC error codes follow specification
		errorCodes := map[string]int{
			"JSONRPCParseError":     knet.JSONRPCParseError,
			"JSONRPCInvalidRequest": knet.JSONRPCInvalidRequest,
			"JSONRPCMethodNotFound": knet.JSONRPCMethodNotFound,
			"JSONRPCInvalidParams":  knet.JSONRPCInvalidParams,
			"JSONRPCInternalError":  knet.JSONRPCInternalError,
		}

		expectedCodes := map[string]int{
			"JSONRPCParseError":     -32700,
			"JSONRPCInvalidRequest": -32600,
			"JSONRPCMethodNotFound": -32601,
			"JSONRPCInvalidParams":  -32602,
			"JSONRPCInternalError":  -32603,
		}

		for name, got := range errorCodes {
			if want := expectedCodes[name]; got != want {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
	})

	t.Run("JSON-RPC version", func(t *testing.T) {
		if knet.JSONRPCVersion != "2.0" {
			t.Errorf("JSONRPCVersion = %v, want 2.0", knet.JSONRPCVersion)
		}
	})
}
