package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ProviderExecutor struct {
	breakers    map[string]*CircuitBreaker
	retryConfig RetryConfig
	mu          sync.RWMutex
}

type ProviderExecutorConfig struct {
	CircuitBreakerConfig CircuitBreakerConfig
	RetryConfig          RetryConfig
}

func CreateProviderExecutor(cfg ProviderExecutorConfig) *ProviderExecutor {
	return &ProviderExecutor{
		breakers:    make(map[string]*CircuitBreaker),
		retryConfig: cfg.RetryConfig,
	}
}

func DefaultProviderExecutorConfig() ProviderExecutorConfig {
	return ProviderExecutorConfig{
		CircuitBreakerConfig: CircuitBreakerConfig{
			MaxFailures: 5,
			Timeout:     30 * time.Second,
			HalfOpenMax: 3,
		},
		RetryConfig: DefaultRetryConfig(),
	}
}

func (pe *ProviderExecutor) Execute(ctx context.Context, provider string, fn func() error) error {
	breaker := pe.getOrCreateBreaker(provider)

	return breaker.Execute(ctx, func() error {
		_, err := Retry(ctx, pe.retryConfig, fn)
		return err
	})
}

func (pe *ProviderExecutor) ExecuteWithResult(ctx context.Context, provider string, fn func() (interface{}, error)) (interface{}, error) {
	breaker := pe.getOrCreateBreaker(provider)

	var result interface{}
	err := breaker.Execute(ctx, func() error {
		_, retryErr := Retry(ctx, pe.retryConfig, func() error {
			var fnErr error
			result, fnErr = fn()
			return fnErr
		})
		return retryErr
	})

	return result, err
}

func (pe *ProviderExecutor) getOrCreateBreaker(provider string) *CircuitBreaker {
	pe.mu.RLock()
	breaker, exists := pe.breakers[provider]
	pe.mu.RUnlock()

	if exists {
		return breaker
	}

	pe.mu.Lock()
	defer pe.mu.Unlock()

	if breaker, exists = pe.breakers[provider]; exists {
		return breaker
	}

	breaker = CreateCircuitBreaker(CircuitBreakerConfig{
		Name:        provider,
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		HalfOpenMax: 3,
		OnStateChange: func(name string, from, to CircuitState) {
			fmt.Printf("Circuit breaker %s: %s -> %s\n", name, from, to)
		},
	})
	pe.breakers[provider] = breaker

	return breaker
}

func (pe *ProviderExecutor) GetBreakerState(provider string) CircuitState {
	pe.mu.RLock()
	breaker, exists := pe.breakers[provider]
	pe.mu.RUnlock()

	if !exists {
		return CircuitClosed
	}
	return breaker.State()
}

func (pe *ProviderExecutor) ResetBreaker(provider string) {
	pe.mu.RLock()
	breaker, exists := pe.breakers[provider]
	pe.mu.RUnlock()

	if exists {
		breaker.Reset()
	}
}

func (pe *ProviderExecutor) IsProviderHealthy(provider string) bool {
	return pe.GetBreakerState(provider) != CircuitOpen
}

