package utils

import (
	"context"
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	state        CircuitState
	failureCount int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

func CreateCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return &APIError{
				Code:    503,
				Message: "Circuit breaker is open",
			}
		}
	}

	err := operation()

	if err != nil {
		cb.failureCount++
		cb.lastFailTime = time.Now()

		if cb.failureCount >= cb.maxFailures {
			cb.state = StateOpen
		}
		return err
	}

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failureCount = 0
	}

	return nil
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}
