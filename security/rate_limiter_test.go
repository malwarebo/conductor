package security

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := CreateRateLimiter()
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             10,
		Window:            time.Second,
	}

	t.Run("Allow within limit", func(t *testing.T) {
		key := "test-key-1"
		for i := 0; i < 10; i++ {
			if !limiter.Allow(key, config) {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("Block after limit", func(t *testing.T) {
		key := "test-key-2"
		limitedConfig := RateLimitConfig{
			RequestsPerSecond: 5,
			Burst:             5,
			Window:            time.Second,
		}

		for i := 0; i < 5; i++ {
			limiter.Allow(key, limitedConfig)
		}

		if limiter.Allow(key, limitedConfig) {
			t.Error("Request should be blocked after limit")
		}
	})
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := CreateRateLimiter()
	key := "test-key-refill"
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             2,
		Window:            time.Second,
	}

	limiter.Allow(key, config)
	limiter.Allow(key, config)

	if limiter.Allow(key, config) {
		t.Error("Request should be blocked")
	}

	time.Sleep(150 * time.Millisecond)

	if !limiter.Allow(key, config) {
		t.Error("Request should be allowed after refill")
	}
}

func TestTieredRateLimiter_TierLimits(t *testing.T) {
	tiers := map[string]RateLimitConfig{
		"free": {
			RequestsPerSecond: 10,
			Burst:             20,
			Window:            time.Second,
		},
		"basic": {
			RequestsPerSecond: 100,
			Burst:             200,
			Window:            time.Second,
		},
		"premium": {
			RequestsPerSecond: 1000,
			Burst:             2000,
			Window:            time.Second,
		},
		"default": {
			RequestsPerSecond: 10,
			Burst:             20,
			Window:            time.Second,
		},
	}
	limiter := CreateTieredRateLimiter(tiers)

	tests := []struct {
		name string
		tier string
	}{
		{
			name: "Free tier",
			tier: "free",
		},
		{
			name: "Basic tier",
			tier: "basic",
		},
		{
			name: "Premium tier",
			tier: "premium",
		},
		{
			name: "Unknown tier defaults to free",
			tier: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "test-key-" + tt.name
			if !limiter.Allow(key, tt.tier) {
				t.Error("First request should be allowed")
			}
		})
	}
}

func TestTieredRateLimiter_FreeTierLimit(t *testing.T) {
	tiers := map[string]RateLimitConfig{
		"free": {
			RequestsPerSecond: 10,
			Burst:             20,
			Window:            time.Second,
		},
		"default": {
			RequestsPerSecond: 10,
			Burst:             20,
			Window:            time.Second,
		},
	}
	limiter := CreateTieredRateLimiter(tiers)
	key := "test-key-free-limit"

	for i := 0; i < 20; i++ {
		if !limiter.Allow(key, "free") {
			t.Errorf("Request %d should be allowed in burst", i+1)
		}
	}

	if limiter.Allow(key, "free") {
		t.Error("Request should be blocked after burst limit")
	}
}

func TestRateLimiter_GetStats(t *testing.T) {
	limiter := CreateRateLimiter()
	key := "test-key-stats"
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             10,
		Window:            time.Second,
	}

	limiter.Allow(key, config)
	limiter.Allow(key, config)
	limiter.Allow(key, config)

	burst, ttl, exists := limiter.GetStats(key)
	if !exists {
		t.Error("GetStats() should return exists=true")
	}
	if burst <= 0 {
		t.Errorf("GetStats() burst = %d, want > 0", burst)
	}
	if ttl < 0 {
		t.Errorf("GetStats() ttl = %v, want >= 0", ttl)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := CreateRateLimiter()
	config := RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             100,
		Window:            time.Second,
	}

	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func(id int) {
			key := "concurrent-key"
			limiter.Allow(key, config)
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	_, _, exists := limiter.GetStats("concurrent-key")
	if !exists {
		t.Error("GetStats() should return exists=true after concurrent access")
	}
}
