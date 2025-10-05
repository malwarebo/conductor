package security

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	cleanup  *time.Timer
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
	Window            time.Duration
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
	rl.startCleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string, config RateLimitConfig) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
		rl.limiters[key] = limiter
	}

	return limiter.Allow()
}

func (rl *RateLimiter) Wait(ctx context.Context, key string, config RateLimitConfig) error {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
		rl.limiters[key] = limiter
		rl.mu.Unlock()
	}

	return limiter.Wait(ctx)
}

func (rl *RateLimiter) Reserve(key string, config RateLimitConfig) *rate.Reservation {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
		rl.limiters[key] = limiter
		rl.mu.Unlock()
	}

	return limiter.Reserve()
}

func (rl *RateLimiter) GetStats(key string) (int, time.Duration, bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		return 0, 0, false
	}

	tokens := int(limiter.Tokens())
	next := time.Duration(limiter.TokensAt(time.Now()))
	return tokens, next, true
}

func (rl *RateLimiter) startCleanup() {
	rl.cleanup = time.AfterFunc(5*time.Minute, func() {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		now := time.Now()
		for key, limiter := range rl.limiters {
			if limiter.TokensAt(now) == float64(limiter.Limit()) {
				delete(rl.limiters, key)
			}
		}

		rl.startCleanup()
	})
}

func (rl *RateLimiter) Close() {
	if rl.cleanup != nil {
		rl.cleanup.Stop()
	}
}

type TieredRateLimiter struct {
	tiers map[string]RateLimitConfig
	rl    *RateLimiter
}

func NewTieredRateLimiter(tiers map[string]RateLimitConfig) *TieredRateLimiter {
	return &TieredRateLimiter{
		tiers: tiers,
		rl:    NewRateLimiter(),
	}
}

func (trl *TieredRateLimiter) Allow(key, tier string) bool {
	config, exists := trl.tiers[tier]
	if !exists {
		config = trl.tiers["default"]
	}

	return trl.rl.Allow(key, config)
}

func (trl *TieredRateLimiter) Wait(ctx context.Context, key, tier string) error {
	config, exists := trl.tiers[tier]
	if !exists {
		config = trl.tiers["default"]
	}

	return trl.rl.Wait(ctx, key, config)
}

func (trl *TieredRateLimiter) GetStats(key, tier string) (int, time.Duration, bool) {
	_, exists := trl.tiers[tier]
	if !exists {
		tier = "default"
	}

	return trl.rl.GetStats(key)
}

func (trl *TieredRateLimiter) Close() {
	trl.rl.Close()
}
