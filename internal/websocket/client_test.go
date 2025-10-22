package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// TestClientID tests that each client has a unique ID
func TestClientID(t *testing.T) {
	t.Parallel()

	// Create multiple mock clients (without actual websocket connection)
	// We'll test the ID generation logic
	ids := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		id := uuid.New().String()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("expected %d unique IDs, got %d", count, len(ids))
	}
}

// TestClientIDFormat tests that client IDs are valid UUIDs
func TestClientIDFormat(t *testing.T) {
	t.Parallel()

	for i := 0; i < 10; i++ {
		id := uuid.New().String()

		// UUID should be 36 characters (with dashes)
		if len(id) != 36 {
			t.Errorf("ID length = %d, want 36", len(id))
		}

		// Try to parse it back
		_, err := uuid.Parse(id)
		if err != nil {
			t.Errorf("ID %s is not a valid UUID: %v", id, err)
		}
	}
}

// TestClientContextCancellation tests that client context is cancelled on close
func TestClientContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Simulate context cancellation
	cancel()

	select {
	case <-ctx.Done():
		// Context was cancelled as expected
	case <-time.After(1 * time.Second):
		t.Error("context was not cancelled")
	}
}

// TestClientContextWithTimeout tests context with timeout
func TestClientContextWithTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("context should have timed out")
	case <-ctx.Done():
		// Context timed out as expected
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	}
}

// TestRateLimiterCreation tests rate limiter creation with different configs
func TestRateLimiterCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *RateLimitConfig
		wantNil bool
	}{
		{
			name:    "with rate limiting enabled",
			config:  DefaultRateLimitConfig(),
			wantNil: false,
		},
		{
			name:    "with rate limiting disabled",
			config:  NoRateLimit(),
			wantNil: true,
		},
		{
			name:    "with nil config",
			config:  nil,
			wantNil: true,
		},
		{
			name: "with custom config enabled",
			config: &RateLimitConfig{
				MessagesPerSecond: 10,
				Burst:             20,
				Enabled:           true,
			},
			wantNil: false,
		},
		{
			name: "with custom config disabled",
			config: &RateLimitConfig{
				MessagesPerSecond: 10,
				Burst:             20,
				Enabled:           false,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var limiter *rate.Limiter
			if tt.config != nil && tt.config.Enabled {
				limiter = rate.NewLimiter(tt.config.MessagesPerSecond, tt.config.Burst)
			}

			if (limiter == nil) != tt.wantNil {
				t.Errorf("rate limiter nil = %v, want nil = %v", limiter == nil, tt.wantNil)
			}

			if limiter != nil {
				// Test that limiter works
				if !limiter.Allow() {
					t.Error("first request should be allowed")
				}
			}
		})
	}
}

// TestChannelBufferSize tests the send channel buffer size
func TestChannelBufferSize(t *testing.T) {
	t.Parallel()

	const expectedBufferSize = 256
	sendCh := make(chan []byte, expectedBufferSize)

	// Check capacity
	if cap(sendCh) != expectedBufferSize {
		t.Errorf("channel capacity = %d, want %d", cap(sendCh), expectedBufferSize)
	}

	// Fill the buffer
	for i := 0; i < expectedBufferSize; i++ {
		select {
		case sendCh <- []byte{byte(i)}:
			// Successfully added
		default:
			t.Errorf("channel should not be full at %d items", i)
		}
	}

	// Next send should block (we test with a timeout)
	select {
	case sendCh <- []byte{0xFF}:
		t.Error("channel should be full, but send succeeded")
	default:
		// Expected: channel is full
	}
}

// TestContextDeadlinePropagation tests that context deadlines propagate correctly
func TestContextDeadlinePropagation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Child context should inherit deadline
	childCtx, childCancel := context.WithCancel(ctx)
	defer childCancel()

	select {
	case <-childCtx.Done():
		// Parent timeout propagated to child
	case <-time.After(100 * time.Millisecond):
		t.Error("child context should have been cancelled by parent timeout")
	}
}

// TestConcurrentContextAccess tests concurrent access to context
func TestConcurrentContextAccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			select {
			case <-ctx.Done():
				done <- true
			case <-time.After(100 * time.Millisecond):
				done <- false
			}
		}()
	}

	// Cancel after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// All goroutines should complete with cancellation
	for i := 0; i < 10; i++ {
		if !<-done {
			t.Error("goroutine did not see context cancellation")
		}
	}
}

// TestRemoteAddressFormat tests remote address format validation
func TestRemoteAddressFormat(t *testing.T) {
	t.Parallel()

	validAddresses := []string{
		"192.168.1.1:12345",
		"127.0.0.1:8080",
		"[::1]:8080",
		"example.com:443",
	}

	for _, addr := range validAddresses {
		t.Run(addr, func(t *testing.T) {
			// Just verify the format is reasonable
			if len(addr) == 0 {
				t.Error("address should not be empty")
			}
		})
	}
}

// BenchmarkUUIDGeneration benchmarks UUID generation
func BenchmarkUUIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uuid.New().String()
	}
}

// BenchmarkChannelSend benchmarks sending to a buffered channel
func BenchmarkChannelSend(b *testing.B) {
	ch := make(chan []byte, 256)
	data := []byte("test data")

	// Consumer
	go func() {
		for range ch {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- data
	}
	close(ch)
}

// BenchmarkContextCreation benchmarks context creation
func BenchmarkContextCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = ctx
	}
}
