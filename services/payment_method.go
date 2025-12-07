package services

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
	"github.com/malwarebo/conductor/stores"
)

type PaymentMethodService struct {
	paymentMethodStore *stores.PaymentMethodStore
	provider           providers.PaymentProvider
}

func CreatePaymentMethodService(paymentMethodStore *stores.PaymentMethodStore, provider providers.PaymentProvider) *PaymentMethodService {
	return &PaymentMethodService{
		paymentMethodStore: paymentMethodStore,
		provider:           provider,
	}
}

func (s *PaymentMethodService) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		pm, err := pmProvider.CreatePaymentMethod(ctx, req)
		if err != nil {
			return nil, err
		}

		if s.paymentMethodStore != nil {
			if err := s.paymentMethodStore.Create(ctx, pm); err != nil {
				return nil, err
			}
		}

		return pm, nil
	}
	return nil, providers.ErrNotSupported
}

func (s *PaymentMethodService) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		return pmProvider.GetPaymentMethod(ctx, paymentMethodID)
	}
	return nil, providers.ErrNotSupported
}

func (s *PaymentMethodService) ListPaymentMethods(ctx context.Context, customerID string, pmType *models.PaymentMethodType) ([]*models.PaymentMethod, error) {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		return pmProvider.ListPaymentMethods(ctx, customerID, pmType)
	}
	return nil, providers.ErrNotSupported
}

func (s *PaymentMethodService) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		return pmProvider.AttachPaymentMethod(ctx, paymentMethodID, customerID)
	}
	return providers.ErrNotSupported
}

func (s *PaymentMethodService) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		return pmProvider.DetachPaymentMethod(ctx, paymentMethodID)
	}
	return providers.ErrNotSupported
}

func (s *PaymentMethodService) ExpirePaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	if pmProvider, ok := s.provider.(providers.PaymentMethodProvider); ok {
		return pmProvider.ExpirePaymentMethod(ctx, paymentMethodID)
	}
	return nil, providers.ErrNotSupported
}

