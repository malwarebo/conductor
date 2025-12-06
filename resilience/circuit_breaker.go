package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrCircuitTimeout = errors.New("circuit breaker timeout")
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	halfOpenMax   int
	state         CircuitState
	failures      int
	successes     int
	lastFailure   time.Time
	mu            sync.RWMutex
	onStateChange func(name string, from, to CircuitState)
}

type CircuitBreakerConfig struct {
	Name          string
	MaxFailures   int
	Timeout       time.Duration
	HalfOpenMax   int
	OnStateChange func(name string, from, to CircuitState)
}

func CreateCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 3
	}

	return &CircuitBreaker{
		name:          cfg.Name,
		maxFailures:   cfg.MaxFailures,
		timeout:       cfg.Timeout,
		halfOpenMax:   cfg.HalfOpenMax,
		state:         CircuitClosed,
		onStateChange: cfg.OnStateChange,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		cb.recordResult(err)
		return err
	case <-ctx.Done():
		cb.recordResult(ctx.Err())
		return ErrCircuitTimeout
	}
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.successes < cb.halfOpenMax
	}
	return false
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		switch cb.state {
		case CircuitClosed:
			if cb.failures >= cb.maxFailures {
				cb.transitionTo(CircuitOpen)
			}
		case CircuitHalfOpen:
			cb.transitionTo(CircuitOpen)
		}
	} else {
		switch cb.state {
		case CircuitClosed:
			cb.failures = 0
		case CircuitHalfOpen:
			cb.successes++
			if cb.successes >= cb.halfOpenMax {
				cb.transitionTo(CircuitClosed)
			}
		}
	}
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.failures = 0
	cb.successes = 0

	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
}

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

