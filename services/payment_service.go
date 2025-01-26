package services

import (
	"context"
	"errors"
	"sync"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
)

var (
	ErrNoAvailableProvider = errors.New("no available payment provider")
	ErrPaymentNotFound     = errors.New("payment not found")
)

type PaymentService struct {
	providers []providers.PaymentProvider
	paymentRepo *repositories.PaymentRepository
	mu        sync.RWMutex
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository, providers ...providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		providers: providers,
		paymentRepo: paymentRepo,
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

func (s *PaymentService) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Create payment record in pending state
	payment := &models.Payment{
		CustomerID:      req.CustomerID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         models.PaymentStatusPending,
		PaymentMethodID: req.PaymentMethodID,
		Description:    req.Description,
		ProviderName:   provider.Name(),
		Metadata:       req.Metadata,
	}

	// Save initial payment record
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Process payment with provider
	providerCharge, err := provider.Charge(ctx, req)
	if err != nil {
		// Update payment status to failed
		payment.Status = models.PaymentStatusFailed
		_ = s.paymentRepo.Update(ctx, payment)
		return nil, err
	}

	// Update payment with provider's charge ID and status
	payment.Status = models.PaymentStatusSucceeded
	payment.ProviderChargeID = providerCharge.TransactionID
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	return &models.ChargeResponse{Payment: payment}, nil
}

func (s *PaymentService) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Get the payment
	payment, err := s.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, ErrPaymentNotFound
	}

	// Create refund record in pending state
	refund := &models.Refund{
		PaymentID:    req.PaymentID,
		Amount:       req.Amount,
		Reason:       req.Reason,
		Status:       "pending",
		ProviderName: provider.Name(),
		Metadata:     req.Metadata,
	}

	// Save initial refund record
	if err := s.paymentRepo.CreateRefund(ctx, refund); err != nil {
		return nil, err
	}

	// Process refund with provider
	providerRefund, err := provider.Refund(ctx, &models.RefundRequest{
		PaymentID: payment.ProviderChargeID, // Use provider's charge ID
		Amount:    req.Amount,
		Reason:    req.Reason,
		Metadata:  req.Metadata,
	})
	if err != nil {
		// Update refund status to failed
		refund.Status = "failed"
		_ = s.paymentRepo.Update(ctx, refund)
		return nil, err
	}

	// Update refund with provider's refund ID and status
	refund.Status = "succeeded"
	refund.ProviderRefundID = providerRefund.RefundID
	if err := s.paymentRepo.Update(ctx, refund); err != nil {
		return nil, err
	}

	// Update payment status if fully refunded
	totalRefunded := int64(0)
	refunds, err := s.paymentRepo.ListRefundsByPayment(ctx, payment.ID)
	if err != nil {
		return nil, err
	}
	for _, r := range refunds {
		if r.Status == "succeeded" {
			totalRefunded += r.Amount
		}
	}
	if totalRefunded >= payment.Amount {
		payment.Status = models.PaymentStatusRefunded
		if err := s.paymentRepo.Update(ctx, payment); err != nil {
			return nil, err
		}
	}

	return &models.RefundResponse{Refund: refund}, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	return s.paymentRepo.GetByID(ctx, id)
}

func (s *PaymentService) ListPayments(ctx context.Context, customerID string) ([]*models.Payment, error) {
	return s.paymentRepo.ListByCustomer(ctx, customerID)
}
