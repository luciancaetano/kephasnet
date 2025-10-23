package unit_test

import (
	"testing"

	"golang.org/x/time/rate"

	"github.com/luciancaetano/knet/internal/websocket"
)

// TestDefaultRateLimitConfig tests the default rate limit configuration
func TestDefaultRateLimitConfig(t *testing.T) {
	t.Parallel()

	config := websocket.DefaultRateLimitConfig()

	if config == nil {
		t.Fatal("DefaultRateLimitConfig() returned nil")
	}

	if !config.Enabled {
		t.Error("Default rate limit should be enabled")
	}

	if config.MessagesPerSecond <= 0 {
		t.Error("MessagesPerSecond should be positive")
	}

	if config.Burst <= 0 {
		t.Error("Burst should be positive")
	}

	// Verify sensible defaults
	if config.MessagesPerSecond != 100 {
		t.Errorf("Default MessagesPerSecond = %v, want 100", config.MessagesPerSecond)
	}

	if config.Burst != 200 {
		t.Errorf("Default Burst = %v, want 200", config.Burst)
	}
}

// TestNoRateLimit tests the no rate limit configuration
func TestNoRateLimit(t *testing.T) {
	t.Parallel()

	config := websocket.NoRateLimit()

	if config == nil {
		t.Fatal("NoRateLimit() returned nil")
	}

	if config.Enabled {
		t.Error("NoRateLimit should have Enabled = false")
	}
}

// TestCustomRateLimitConfig tests creating custom rate limit configurations
func TestCustomRateLimitConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		messagesPerSecond float64
		burst             int
		enabled           bool
	}{
		{
			name:              "low rate limit",
			messagesPerSecond: 10,
			burst:             20,
			enabled:           true,
		},
		{
			name:              "high rate limit",
			messagesPerSecond: 1000,
			burst:             2000,
			enabled:           true,
		},
		{
			name:              "disabled",
			messagesPerSecond: 0,
			burst:             0,
			enabled:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &websocket.RateLimitConfig{
				MessagesPerSecond: rate.Limit(tt.messagesPerSecond),
				Burst:             tt.burst,
				Enabled:           tt.enabled,
			}

			if config.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", config.Enabled, tt.enabled)
			}

			if float64(config.MessagesPerSecond) != tt.messagesPerSecond {
				t.Errorf("MessagesPerSecond = %v, want %v", config.MessagesPerSecond, tt.messagesPerSecond)
			}

			if config.Burst != tt.burst {
				t.Errorf("Burst = %v, want %v", config.Burst, tt.burst)
			}
		})
	}
}
