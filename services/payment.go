package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/monitoring"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/stores"
	"github.com/malwarebo/gopay/security"
	"github.com/malwarebo/gopay/utils"
)

type PaymentService struct {
	paymentRepo  *stores.PaymentRepository
	provider     providers.PaymentProvider
	encryption   *security.EncryptionManager
	alertManager *monitoring.AlertManager
}

func CreatePaymentService(paymentRepo *stores.PaymentRepository, provider providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		provider:    provider,
	}
}

func CreatePaymentServiceWithMonitoring(paymentRepo *stores.PaymentRepository, provider providers.PaymentProvider, encryption *security.EncryptionManager, alertManager *monitoring.AlertManager) *PaymentService {
	return &PaymentService{
		paymentRepo:  paymentRepo,
		provider:     provider,
		encryption:   encryption,
		alertManager: alertManager,
	}
}

func (s *PaymentService) CreateCharge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	if err := s.validateChargeRequest(req); err != nil {
		return nil, err
	}

	idempotencyKey := s.generateIdempotencyKey()
	ctx = context.WithValue(ctx, "idempotency_key", idempotencyKey)

	existingPayment, err := s.paymentRepo.GetByID(ctx, req.PaymentMethod)
	if err == nil && existingPayment != nil {
		return s.buildChargeResponse(existingPayment), nil
	}

	providerName := s.selectProvider(ctx, req.Currency)
	if providerName == "" {
		return nil, fmt.Errorf("no available provider for currency: %s", req.Currency)
	}

	var payment *models.Payment
	var chargeResp *models.ChargeResponse

	err = s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		retryConfig := utils.CreateDefaultRetryConfig()
		retryConfig.MaxAttempts = 5
		retryConfig.BaseDelay = 200 * time.Millisecond
		retryConfig.MaxDelay = 10 * time.Second

		err := utils.CreateRetry(txCtx, retryConfig, func() error {
			chargeResp, err = s.provider.Charge(txCtx, req)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to create charge with provider: %w", err)
		}

		payment = &models.Payment{
			ID:               chargeResp.ID,
			Amount:           chargeResp.Amount,
			Currency:         chargeResp.Currency,
			Status:           chargeResp.Status,
			PaymentMethod:    req.PaymentMethod,
			CustomerID:       req.CustomerID,
			Description:      req.Description,
			ProviderName:     providerName,
			ProviderChargeID: chargeResp.ProviderChargeID,
			Metadata:         req.Metadata,
			CreatedAt:        time.Now(),
		}

		return s.paymentRepo.Create(txCtx, payment)
	})

	if err != nil {
		return nil, err
	}

	return s.buildChargeResponse(payment), nil
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	if err := s.validateRefundRequest(req); err != nil {
		return nil, err
	}

	payment, err := s.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %v", err)
	}

	if payment.Status != "succeeded" {
		return nil, fmt.Errorf("cannot refund payment with status: %s", payment.Status)
	}

	var refundResp *models.RefundResponse

	err = s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		retryConfig := utils.CreateDefaultRetryConfig()
		retryConfig.MaxAttempts = 3
		retryConfig.BaseDelay = 500 * time.Millisecond

		err := utils.CreateRetry(txCtx, retryConfig, func() error {
			refundResp, err = s.provider.Refund(txCtx, req)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to create refund with provider: %w", err)
		}

		refund := &models.Refund{
			ID:               refundResp.ID,
			PaymentID:        req.PaymentID,
			Amount:           refundResp.Amount,
			Status:           refundResp.Status,
			Reason:           req.Reason,
			ProviderName:     refundResp.ProviderName,
			ProviderRefundID: refundResp.ProviderRefundID,
			Metadata:         req.Metadata,
			CreatedAt:        time.Now(),
		}

		return s.paymentRepo.CreateRefund(txCtx, refund)
	})

	if err != nil {
		return nil, err
	}

	return refundResp, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id string) (*models.Payment, error) {
	return s.paymentRepo.GetByID(ctx, id)
}

func (s *PaymentService) ListPayments(ctx context.Context, customerID string) ([]*models.Payment, error) {
	return s.paymentRepo.ListByCustomer(ctx, customerID)
}

func (s *PaymentService) GetRefund(ctx context.Context, id string) (*models.Refund, error) {
	return s.paymentRepo.GetRefundByID(ctx, id)
}

func (s *PaymentService) ListRefunds(ctx context.Context, paymentID string) ([]*models.Refund, error) {
	return s.paymentRepo.ListRefundsByPayment(ctx, paymentID)
}

func (s *PaymentService) validateChargeRequest(req *models.ChargeRequest) error {
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if req.Currency == "" {
		return errors.New("currency is required")
	}
	if req.PaymentMethod == "" {
		return errors.New("payment method is required")
	}
	return nil
}

func (s *PaymentService) validateRefundRequest(req *models.RefundRequest) error {
	if req.PaymentID == "" {
		return errors.New("payment ID is required")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	return nil
}

func (s *PaymentService) selectProvider(ctx context.Context, currency string) string {
	if s.provider.IsAvailable(ctx) {
		return "stripe"
	}
	return ""
}

func (s *PaymentService) generateIdempotencyKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *PaymentService) buildChargeResponse(payment *models.Payment) *models.ChargeResponse {
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
	}
}
