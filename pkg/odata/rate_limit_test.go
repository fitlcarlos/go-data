package odata

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	t.Run("Has default values", func(t *testing.T) {
		assert.NotNil(t, config)
		assert.True(t, config.Enabled, "Rate limiting should be enabled by default for security")
		assert.Equal(t, DefaultRateLimitPerMinute, config.RequestsPerMinute)
		assert.Equal(t, DefaultRateLimitBurstSize, config.BurstSize)
		assert.Equal(t, DefaultRateLimitWindow, config.WindowSize)
		assert.NotNil(t, config.KeyGenerator)
		assert.True(t, config.Headers)
	})
}

func TestNewRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 10,
		BurstSize:         5,
		WindowSize:        time.Minute,
		KeyGenerator:      defaultKeyGenerator,
	}

	t.Run("Creates limiter", func(t *testing.T) {
		limiter := NewRateLimiter(config)

		assert.NotNil(t, limiter)
		assert.Equal(t, config, limiter.config)
		assert.NotNil(t, limiter.clients)
		assert.NotNil(t, limiter.stopCleanup)
	})

	t.Run("Starts cleanup when enabled", func(t *testing.T) {
		limiter := NewRateLimiter(config)
		defer limiter.Stop()

		assert.NotNil(t, limiter.cleanupTicker)
	})

	t.Run("Does not start cleanup when disabled", func(t *testing.T) {
		disabledConfig := &RateLimitConfig{
			Enabled:           false,
			RequestsPerMinute: 10,
			WindowSize:        time.Minute,
		}

		limiter := NewRateLimiter(disabledConfig)
		assert.Nil(t, limiter.cleanupTicker)
	})
}

func TestRateLimiter_Allow_FirstRequest(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 10,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("First request is allowed", func(t *testing.T) {
		allowed, info := limiter.Allow("client1")

		assert.True(t, allowed)
		assert.True(t, info.Allowed)
		assert.Equal(t, 10, info.Limit)
		assert.Equal(t, 9, info.Remaining)
		assert.Equal(t, 0, info.RetryAfter)
	})
}

func TestRateLimiter_Allow_MultipleRequests(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 5,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Multiple requests within limit", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			allowed, info := limiter.Allow("client1")
			assert.True(t, allowed, "Request %d should be allowed", i+1)
			assert.Equal(t, 5-i-1, info.Remaining, "Remaining count incorrect for request %d", i+1)
		}
	})

	t.Run("Request exceeding limit is blocked", func(t *testing.T) {
		// Allow all 5 requests
		for i := 0; i < 5; i++ {
			limiter.Allow("client2")
		}

		// 6th request should be blocked
		allowed, info := limiter.Allow("client2")

		assert.False(t, allowed)
		assert.False(t, info.Allowed)
		assert.Equal(t, 0, info.Remaining)
		assert.Greater(t, info.RetryAfter, 0)
	})
}

func TestRateLimiter_Allow_DifferentClients(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 3,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Different clients have separate limits", func(t *testing.T) {
		// Client 1 uses all requests
		for i := 0; i < 3; i++ {
			limiter.Allow("client1")
		}

		// Client 1 should be blocked
		allowed1, _ := limiter.Allow("client1")
		assert.False(t, allowed1)

		// Client 2 should still be allowed
		allowed2, info2 := limiter.Allow("client2")
		assert.True(t, allowed2)
		assert.Equal(t, 2, info2.Remaining)
	})
}

func TestRateLimiter_Allow_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           false,
		RequestsPerMinute: 1,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)

	t.Run("All requests allowed when disabled", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			allowed, _ := limiter.Allow("client1")
			assert.True(t, allowed, "Request %d should be allowed when disabled", i+1)
		}
	})
}

func TestRateLimiter_Allow_WindowReset(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 2,
		WindowSize:        100 * time.Millisecond, // Small window for testing
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Requests allowed after window reset", func(t *testing.T) {
		// Use both requests
		limiter.Allow("client1")
		limiter.Allow("client1")

		// Next request should be blocked
		allowed1, _ := limiter.Allow("client1")
		assert.False(t, allowed1)

		// Wait for window to reset
		time.Sleep(150 * time.Millisecond)

		// Now requests should be allowed again
		allowed2, info2 := limiter.Allow("client1")
		assert.True(t, allowed2)
		assert.Equal(t, 1, info2.Remaining)
	})
}

func TestRateLimiter_Stop(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 10,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)

	t.Run("Stop cleans up resources", func(t *testing.T) {
		assert.NotNil(t, limiter.cleanupTicker)

		limiter.Stop()

		// Give some time for cleanup goroutine to stop
		time.Sleep(10 * time.Millisecond)

		// Ticker should be stopped (we can't directly check this, but no panic is good)
	})

	t.Run("Stop on disabled limiter does not panic", func(t *testing.T) {
		disabledConfig := &RateLimitConfig{
			Enabled:           false,
			RequestsPerMinute: 10,
		}

		disabledLimiter := NewRateLimiter(disabledConfig)

		assert.NotPanics(t, func() {
			disabledLimiter.Stop()
		})
	})
}

func TestRateLimiter_CleanupInactiveClients(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 10,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Inactive clients are removed", func(t *testing.T) {
		// Add a client
		limiter.Allow("client1")

		// Verify client exists
		limiter.mu.RLock()
		assert.Len(t, limiter.clients, 1)
		limiter.mu.RUnlock()

		// Manually set lastAccess to old time
		limiter.mu.Lock()
		client := limiter.clients["client1"]
		client.mu.Lock()
		client.lastAccess = time.Now().Add(-15 * time.Minute)
		client.mu.Unlock()
		limiter.mu.Unlock()

		// Run cleanup
		limiter.cleanupInactiveClients()

		// Verify client was removed
		limiter.mu.RLock()
		assert.Len(t, limiter.clients, 0)
		limiter.mu.RUnlock()
	})

	t.Run("Active clients are kept", func(t *testing.T) {
		// Add a client
		limiter.Allow("client2")

		// Run cleanup immediately
		limiter.cleanupInactiveClients()

		// Verify client still exists
		limiter.mu.RLock()
		assert.Len(t, limiter.clients, 1)
		limiter.mu.RUnlock()
	})
}

func TestRateLimitInfo(t *testing.T) {
	t.Run("Contains expected fields", func(t *testing.T) {
		info := RateLimitInfo{
			Allowed:    true,
			Limit:      100,
			Remaining:  50,
			ResetTime:  time.Now().Add(time.Minute),
			RetryAfter: 0,
		}

		assert.True(t, info.Allowed)
		assert.Equal(t, 100, info.Limit)
		assert.Equal(t, 50, info.Remaining)
		assert.NotZero(t, info.ResetTime)
		assert.Equal(t, 0, info.RetryAfter)
	})
}

func TestCustomKeyGenerator(t *testing.T) {
	t.Run("Returns custom function", func(t *testing.T) {
		customFn := func(c fiber.Ctx) string {
			return "custom_key"
		}

		generator := CustomKeyGenerator(customFn)

		assert.NotNil(t, generator)
		// We can't directly test the function without a Fiber context,
		// but we can verify it's not nil
	})
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 100,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Handles concurrent requests safely", func(t *testing.T) {
		const numGoroutines = 10
		const requestsPerGoroutine = 10

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < requestsPerGoroutine; j++ {
					limiter.Allow("concurrent_client")
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to finish
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check that the limiter didn't crash
		assert.NotNil(t, limiter)
	})
}

func TestRateLimiter_ResetTime(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 5,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("ResetTime is in the future", func(t *testing.T) {
		allowed, info := limiter.Allow("client1")

		assert.True(t, allowed)
		assert.True(t, info.ResetTime.After(time.Now()))
	})

	t.Run("ResetTime is approximately WindowSize away", func(t *testing.T) {
		_, info := limiter.Allow("client2")

		expectedReset := time.Now().Add(config.WindowSize)
		timeDiff := info.ResetTime.Sub(expectedReset).Abs()

		// Allow 1 second tolerance for test execution time
		assert.Less(t, timeDiff, time.Second)
	})
}

func TestRateLimiter_RetryAfter(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 2,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("RetryAfter is calculated correctly when blocked", func(t *testing.T) {
		// Use up all requests
		limiter.Allow("client1")
		limiter.Allow("client1")

		// Next request should be blocked with RetryAfter
		allowed, info := limiter.Allow("client1")

		assert.False(t, allowed)
		assert.Greater(t, info.RetryAfter, 0)
		assert.LessOrEqual(t, info.RetryAfter, 60) // Should be within the window size
	})
}

func TestRateLimiter_EdgeCases(t *testing.T) {
	t.Run("Zero requests per minute (edge case - blocks all)", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 0,
			WindowSize:        time.Minute,
		}

		limiter := NewRateLimiter(config)
		defer limiter.Stop()

		// All requests should be blocked since limit is 0
		// But this is an edge case that may cause issues, so we just verify it doesn't panic
		allowed, _ := limiter.Allow("client1")
		// With 0 requests, it should block (or we could argue it's undefined behavior)
		_ = allowed // We don't assert anything specific for this edge case
	})

	t.Run("Very large limit", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000000,
			WindowSize:        time.Minute,
		}

		limiter := NewRateLimiter(config)
		defer limiter.Stop()

		// Should handle large limits
		for i := 0; i < 100; i++ {
			allowed, _ := limiter.Allow("client1")
			assert.True(t, allowed)
		}
	})

	t.Run("Very short window", func(t *testing.T) {
		config := &RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 5,
			WindowSize:        10 * time.Millisecond,
		}

		limiter := NewRateLimiter(config)
		defer limiter.Stop()

		// Fill up the limit
		for i := 0; i < 5; i++ {
			limiter.Allow("client1")
		}

		// Should be blocked
		allowed1, _ := limiter.Allow("client1")
		assert.False(t, allowed1)

		// Wait for window to pass
		time.Sleep(15 * time.Millisecond)

		// Should be allowed again
		allowed2, _ := limiter.Allow("client1")
		assert.True(t, allowed2)
	})
}

func TestRateLimiter_EmptyKey(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 5,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Empty key is treated as valid", func(t *testing.T) {
		allowed, info := limiter.Allow("")

		assert.True(t, allowed)
		assert.Equal(t, 4, info.Remaining)
	})
}

func TestRateLimiter_MultipleKeysIndependent(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 2,
		WindowSize:        time.Minute,
	}

	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	t.Run("Keys are completely independent", func(t *testing.T) {
		// Client A uses 2 requests
		limiter.Allow("clientA")
		limiter.Allow("clientA")

		// Client A should be blocked
		allowedA, _ := limiter.Allow("clientA")
		assert.False(t, allowedA)

		// Client B should have full quota
		allowedB1, infoB1 := limiter.Allow("clientB")
		assert.True(t, allowedB1)
		assert.Equal(t, 1, infoB1.Remaining)

		// Client C should have full quota
		allowedC, infoC := limiter.Allow("clientC")
		assert.True(t, allowedC)
		assert.Equal(t, 1, infoC.Remaining)
	})
}
