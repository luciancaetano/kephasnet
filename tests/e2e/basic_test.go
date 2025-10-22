package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/luciancaetano/kephasnet/internal/protocol"
	"github.com/luciancaetano/kephasnet/ws"
)

func TestBasicEcho(t *testing.T) {
	t.Parallel()

	server := ws.New(":18080", ws.DefaultRateLimitConfig(), ws.AllOrigins())
	ctx := context.Background()

	const cmdEcho uint32 = 0x0001
	server.RegisterHandler(ctx, cmdEcho, func(payload []byte) ([]byte, error) {
		return payload, nil
	})

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(stopCtx)
	}()

	time.Sleep(200 * time.Millisecond)

	conn, _, err := newDialer().Dial("ws://localhost:18080/ws", nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	testPayload := []byte("Hello!")
	encoded, _ := protocol.Encode(cmdEcho, testPayload)

	if err := conn.WriteMessage(websocket.BinaryMessage, encoded); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	_, respPayload, err := protocol.Decode(response)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if string(respPayload) != string(testPayload) {
		t.Errorf("got %q, want %q", respPayload, testPayload)
	}
}
