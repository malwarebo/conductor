package circuitbreaker

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	return [...]string{"closed", "open", "half_open"}[s]
}

type Config struct {
	FailureThreshold   int
	SuccessThreshold   int
	Timeout            time.Duration
	RollingWindowSize  time.Duration
	MinRequestsInWindow int
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold:   5,
		SuccessThreshold:   3,
		Timeout:            30 * time.Second,
		RollingWindowSize:  60 * time.Second,
		MinRequestsInWindow: 10,
	}
}

type CircuitBreaker struct {
	mu              sync.RWMutex
	name            string
	config          Config
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time
	requests        []requestResult
}

type requestResult struct {
	timestamp time.Time
	success   bool
}

func New(name string, config Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
		requests:        make([]requestResult, 0),
	}
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastStateChange) >= cb.config.Timeout {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.addRequest(true)

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
		}
	case StateClosed:
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.addRequest(false)
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) setState(state State) {
	cb.state = state
	cb.lastStateChange = time.Now()
	cb.failures = 0
	cb.successes = 0
}

func (cb *CircuitBreaker) addRequest(success bool) {
	now := time.Now()
	cb.requests = append(cb.requests, requestResult{timestamp: now, success: success})
	cb.pruneOldRequests(now)
}

func (cb *CircuitBreaker) pruneOldRequests(now time.Time) {
	cutoff := now.Add(-cb.config.RollingWindowSize)
	idx := 0
	for i, r := range cb.requests {
		if r.timestamp.After(cutoff) {
			idx = i
			break
		}
	}
	if idx > 0 {
		cb.requests = cb.requests[idx:]
	}
}

func (cb *CircuitBreaker) SuccessRate() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	cb.pruneOldRequests(time.Now())

	if len(cb.requests) == 0 {
		return 1.0
	}

	successes := 0
	for _, r := range cb.requests {
		if r.success {
			successes++
		}
	}
	return float64(successes) / float64(len(cb.requests))
}

func (cb *CircuitBreaker) RequestCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return len(cb.requests)
}

func (cb *CircuitBreaker) IsHealthy() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateOpen {
		return false
	}

	if len(cb.requests) < cb.config.MinRequestsInWindow {
		return true
	}

	return cb.SuccessRate() >= 0.9
}

func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":          cb.name,
		"state":         cb.state.String(),
		"success_rate":  cb.SuccessRate(),
		"request_count": len(cb.requests),
		"failures":      cb.failures,
		"successes":     cb.successes,
	}
}

type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   Config
}

func NewManager(config Config) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

func (m *Manager) Get(name string) *CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.breakers[name]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists = m.breakers[name]; exists {
		return cb
	}

	cb = New(name, m.config)
	m.breakers[name] = cb
	return cb
}

func (m *Manager) AllStats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for name, cb := range m.breakers {
		stats[name] = cb.Stats()
	}
	return stats
}

func (m *Manager) HealthyProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthy := make([]string, 0)
	for name, cb := range m.breakers {
		if cb.IsHealthy() {
			healthy = append(healthy, name)
		}
	}
	return healthy
}
