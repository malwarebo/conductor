package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

type RetryConfig struct {
	MaxAttempts     int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	BackoffType     BackoffType
	RetryableErrors []error
}

type BackoffType int

const (
	Linear BackoffType = iota
	Exponential
	ExponentialJitter
	Fixed
)

type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
		BackoffType:     ExponentialJitter,
		RetryableErrors: []error{},
	}
}

func Retry(ctx context.Context, config *RetryConfig, operation func() error) error {
	var lastErr error
	attempt := 0

	for attempt < config.MaxAttempts {
		if attempt > 0 {
			delay := calculateDelay(config, attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryableError(err, config.RetryableErrors) {
			return err
		}

		attempt++
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

func RetryWithResult(ctx context.Context, config *RetryConfig, operation func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var lastErr error
	attempt := 0

	for attempt < config.MaxAttempts {
		if attempt > 0 {
			delay := calculateDelay(config, attempt)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := operation()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryableError(err, config.RetryableErrors) {
			return result, err
		}

		attempt++
	}

	return result, fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	var delay time.Duration

	switch config.BackoffType {
	case Linear:
		delay = config.BaseDelay * time.Duration(attempt)
	case Exponential:
		delay = time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1)))
	case ExponentialJitter:
		baseDelay := time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1)))
		if config.Jitter {
			jitter := time.Duration(rand.Float64() * float64(baseDelay) * 0.1)
			delay = baseDelay + jitter
		} else {
			delay = baseDelay
		}
	case Fixed:
		delay = config.BaseDelay
	default:
		delay = config.BaseDelay
	}

	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	return delay
}

func isRetryableError(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		return true
	}

	for _, retryableErr := range retryableErrors {
		if fmt.Sprintf("%T", err) == fmt.Sprintf("%T", retryableErr) {
			return true
		}
	}

	return false
}

func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

type CircuitBreakerRetry struct {
	config         *RetryConfig
	circuitBreaker *CircuitBreaker
}

func NewCircuitBreakerRetry(config *RetryConfig, cb *CircuitBreaker) *CircuitBreakerRetry {
	return &CircuitBreakerRetry{
		config:         config,
		circuitBreaker: cb,
	}
}

func (cbr *CircuitBreakerRetry) Execute(ctx context.Context, operation func() error) error {
	return cbr.circuitBreaker.Execute(ctx, func() error {
		return Retry(ctx, cbr.config, operation)
	})
}

func (cbr *CircuitBreakerRetry) ExecuteWithResult(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var err error

	executeErr := cbr.circuitBreaker.Execute(ctx, func() error {
		result, err = RetryWithResult(ctx, cbr.config, operation)
		return err
	})

	return result, executeErr
}
