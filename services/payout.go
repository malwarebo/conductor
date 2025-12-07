package services

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
)

type PayoutService struct {
	provider providers.PaymentProvider
}

func CreatePayoutService(provider providers.PaymentProvider) *PayoutService {
	return &PayoutService{
		provider: provider,
	}
}

func (s *PayoutService) CreatePayout(ctx context.Context, req *models.CreatePayoutRequest) (*models.Payout, error) {
	if payoutProvider, ok := s.provider.(providers.PayoutProvider); ok {
		return payoutProvider.CreatePayout(ctx, req)
	}
	return nil, providers.ErrNotSupported
}

func (s *PayoutService) GetPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	if payoutProvider, ok := s.provider.(providers.PayoutProvider); ok {
		return payoutProvider.GetPayout(ctx, payoutID)
	}
	return nil, providers.ErrNotSupported
}

func (s *PayoutService) ListPayouts(ctx context.Context, req *models.ListPayoutsRequest) ([]*models.Payout, error) {
	if payoutProvider, ok := s.provider.(providers.PayoutProvider); ok {
		return payoutProvider.ListPayouts(ctx, req)
	}
	return nil, providers.ErrNotSupported
}

func (s *PayoutService) CancelPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	if payoutProvider, ok := s.provider.(providers.PayoutProvider); ok {
		return payoutProvider.CancelPayout(ctx, payoutID)
	}
	return nil, providers.ErrNotSupported
}

func (s *PayoutService) GetPayoutChannels(ctx context.Context, currency string) ([]*models.PayoutChannel, error) {
	if payoutProvider, ok := s.provider.(providers.PayoutProvider); ok {
		return payoutProvider.GetPayoutChannels(ctx, currency)
	}
	return nil, providers.ErrNotSupported
}

