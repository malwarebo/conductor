package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
)

var (
	// ErrInvalidAmount is returned when the amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrInvalidCurrency is returned when the currency is invalid
	ErrInvalidCurrency = errors.New("invalid currency")
	// ErrInvalidPaymentMethod is returned when the payment method is invalid
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
)

type PaymentService struct {
	paymentRepo    *repositories.PaymentRepository
	provider       providers.PaymentProvider
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository, provider providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		provider:    provider,
	}
}

func (s *PaymentService) CreateCharge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	// Validate request
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		return nil, ErrInvalidCurrency
	}
	if req.PaymentMethod == "" {
		return nil, ErrInvalidPaymentMethod
	}

	// Create charge using provider
	chargeResp, err := s.provider.Charge(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}

	// Store payment in database
	payment := &models.Payment{
		CustomerID:       req.CustomerID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          models.PaymentStatusSuccess,
		PaymentMethod:   req.PaymentMethod,
		Description:     req.Description,
		ProviderName:    "stripe",
		ProviderChargeID: chargeResp.ID,
		Metadata:        req.Metadata,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to store payment: %w", err)
	}

	return &models.ChargeResponse{Payment: payment}, nil
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	// Validate request
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		return nil, ErrInvalidCurrency
	}
	if req.PaymentID == "" {
		return nil, errors.New("payment ID is required")
	}

	// Get the payment
	payment, err := s.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Create refund using provider
	refundResp, err := s.provider.Refund(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	// Update payment status
	payment.Status = models.PaymentStatusRefunded
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return &models.RefundResponse{Refund: &models.Refund{
		PaymentID:    req.PaymentID,
		Amount:       req.Amount,
		Reason:       req.Reason,
		Status:       "succeeded",
		ProviderName: "stripe",
		ProviderRefundID: refundResp.RefundID,
		Metadata:     req.Metadata,
	}}, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return payment, nil
}

func (s *PaymentService) ListPayments(ctx context.Context, customerID string) ([]models.Payment, error) {
	payments, err := s.paymentRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	return payments, nil
}
