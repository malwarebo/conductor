package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ProviderExecutor struct {
	fuses       map[string]*Fuse
	retryConfig RetryConfig
	mu          sync.RWMutex
}

type ProviderExecutorConfig struct {
	FuseConfig  FuseConfig
	RetryConfig RetryConfig
}

func CreateProviderExecutor(cfg ProviderExecutorConfig) *ProviderExecutor {
	return &ProviderExecutor{
		fuses:       make(map[string]*Fuse),
		retryConfig: cfg.RetryConfig,
	}
}

func DefaultProviderExecutorConfig() ProviderExecutorConfig {
	return ProviderExecutorConfig{
		FuseConfig: FuseConfig{
			MaxFailures: 5,
			Timeout:     30 * time.Second,
			HalfOpenMax: 3,
		},
		RetryConfig: DefaultRetryConfig(),
	}
}

func (pe *ProviderExecutor) Execute(ctx context.Context, provider string, fn func() error) error {
	fuse := pe.getOrCreateFuse(provider)

	return fuse.Execute(ctx, func() error {
		_, err := Retry(ctx, pe.retryConfig, fn)
		return err
	})
}

func (pe *ProviderExecutor) ExecuteWithResult(ctx context.Context, provider string, fn func() (interface{}, error)) (interface{}, error) {
	fuse := pe.getOrCreateFuse(provider)

	var result interface{}
	err := fuse.Execute(ctx, func() error {
		_, retryErr := Retry(ctx, pe.retryConfig, func() error {
			var fnErr error
			result, fnErr = fn()
			return fnErr
		})
		return retryErr
	})

	return result, err
}

func (pe *ProviderExecutor) getOrCreateFuse(provider string) *Fuse {
	pe.mu.RLock()
	fuse, exists := pe.fuses[provider]
	pe.mu.RUnlock()

	if exists {
		return fuse
	}

	pe.mu.Lock()
	defer pe.mu.Unlock()

	if fuse, exists = pe.fuses[provider]; exists {
		return fuse
	}

	fuse = CreateFuse(FuseConfig{
		Name:        provider,
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		HalfOpenMax: 3,
		OnStateChange: func(name string, from, to FuseState) {
			fmt.Printf("Fuse %s: %s -> %s\n", name, from, to)
		},
	})
	pe.fuses[provider] = fuse

	return fuse
}

func (pe *ProviderExecutor) GetFuseState(provider string) FuseState {
	pe.mu.RLock()
	fuse, exists := pe.fuses[provider]
	pe.mu.RUnlock()

	if !exists {
		return FuseClosed
	}
	return fuse.State()
}

func (pe *ProviderExecutor) ResetFuse(provider string) {
	pe.mu.RLock()
	fuse, exists := pe.fuses[provider]
	pe.mu.RUnlock()

	if exists {
		fuse.Reset()
	}
}

func (pe *ProviderExecutor) IsProviderHealthy(provider string) bool {
	return pe.GetFuseState(provider) != FuseOpen
}

