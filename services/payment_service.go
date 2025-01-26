package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/malwarebo/gopay/providers"
)

var (
	ErrNoAvailableProvider = errors.New("no available payment provider")
)

type PaymentService struct {
	providers []providers.PaymentProvider
	mu        sync.RWMutex
}

func NewPaymentService(providers ...providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		providers: providers,
	}
}

func (s *PaymentService) AddProvider(provider providers.PaymentProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, provider)
}

func (s *PaymentService) getAvailableProvider(ctx context.Context) providers.PaymentProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.providers {
		if provider.IsAvailable(ctx) {
			return provider
		}
	}
	return nil
}

func (s *PaymentService) Charge(ctx context.Context, req *providers.ChargeRequest) (*providers.ChargeResponse, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	return provider.Charge(ctx, req)
}

func (s *PaymentService) Refund(ctx context.Context, req *providers.RefundRequest) (*providers.RefundResponse, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	return provider.Refund(ctx, req)
}
