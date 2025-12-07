package providers

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

type RetryConfig struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	Jitter         bool
	RetryableCheck func(error) bool
}

type RetryResult struct {
	Attempts int
	LastErr  error
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableCheck: func(err error) bool {
			return err != nil
		},
	}
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() error) (*RetryResult, error) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 100 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 10 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.RetryableCheck == nil {
		cfg.RetryableCheck = func(err error) bool { return err != nil }
	}

	result := &RetryResult{}
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		select {
		case <-ctx.Done():
			result.LastErr = ctx.Err()
			return result, ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		result.LastErr = err

		if !cfg.RetryableCheck(err) {
			return result, err
		}

		if attempt < cfg.MaxRetries {
			delay := calculateDelay(cfg, attempt)
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return result, lastErr
}

func calculateDelay(cfg RetryConfig, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	if cfg.Jitter {
		jitter := rand.Float64() * 0.3 * delay
		delay = delay + jitter - (0.15 * delay)
	}

	return time.Duration(delay)
}

type RetryableFunc func(ctx context.Context) error

func WithRetry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	_, err := Retry(ctx, cfg, func() error {
		return fn(ctx)
	})
	return err
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

func CreateRetryableCheck(nonRetryableErrors ...error) func(error) bool {
	return func(err error) bool {
		if err == nil {
			return false
		}
		for _, nonRetryable := range nonRetryableErrors {
			if errors.Is(err, nonRetryable) {
				return false
			}
		}
		return true
	}
}

