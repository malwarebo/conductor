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
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrInvalidCurrency      = errors.New("invalid currency")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
)

type PaymentService struct {
	paymentRepo *repositories.PaymentRepository
	provider    providers.PaymentProvider
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository, provider providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		provider:    provider,
	}
}

func (s *PaymentService) CreateCharge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		return nil, ErrInvalidCurrency
	}
	if req.PaymentMethod == "" {
		return nil, ErrInvalidPaymentMethod
	}

	chargeResp, err := s.provider.Charge(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create charge: %w", err)
	}

	payment := &models.Payment{
		CustomerID:       req.CustomerID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           models.PaymentStatusSuccess,
		PaymentMethod:    req.PaymentMethod,
		Description:      req.Description,
		ProviderName:     "stripe",
		ProviderChargeID: chargeResp.ID,
		Metadata:         req.Metadata,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to store payment: %w", err)
	}

	return &models.ChargeResponse{
		ID:               payment.ID,
		CustomerID:       payment.CustomerID,
		Amount:           payment.Amount,
		Currency:         payment.Currency,
		Status:           payment.Status,
		PaymentMethod:    payment.PaymentMethod,
		Description:      payment.Description,
		ProviderName:     payment.ProviderName,
		ProviderChargeID: payment.ProviderChargeID,
		Metadata:         payment.Metadata,
		CreatedAt:        payment.CreatedAt,
	}, nil
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		return nil, ErrInvalidCurrency
	}
	if req.PaymentID == "" {
		return nil, errors.New("payment ID is required")
	}

	payment, err := s.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	refundResp, err := s.provider.Refund(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	payment.Status = models.PaymentStatusRefunded
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	refund := &models.Refund{
		PaymentID:        req.PaymentID,
		Amount:           req.Amount,
		Reason:           req.Reason,
		Status:           "succeeded",
		ProviderName:     "stripe",
		ProviderRefundID: refundResp.ID,
		Metadata:         req.Metadata,
	}

	return &models.RefundResponse{
		ID:               refund.ID,
		PaymentID:        refund.PaymentID,
		Amount:           refund.Amount,
		Currency:         req.Currency,
		Status:           refund.Status,
		Reason:           refund.Reason,
		ProviderName:     refund.ProviderName,
		ProviderRefundID: refund.ProviderRefundID,
		Metadata:         refund.Metadata,
		CreatedAt:        refund.CreatedAt,
	}, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return payment, nil
}

func (s *PaymentService) ListPayments(ctx context.Context, customerID string) ([]*models.Payment, error) {
	payments, err := s.paymentRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	return payments, nil
}
