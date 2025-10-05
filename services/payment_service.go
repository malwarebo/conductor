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
	"github.com/malwarebo/gopay/repositories"
	"github.com/malwarebo/gopay/security"
	"github.com/malwarebo/gopay/utils"
)

type PaymentService struct {
	paymentRepo  *repositories.PaymentRepository
	provider     providers.PaymentProvider
	encryption   *security.EncryptionManager
	alertManager *monitoring.AlertManager
}

func NewPaymentService(paymentRepo *repositories.PaymentRepository, provider providers.PaymentProvider) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		provider:    provider,
	}
}

func NewPaymentServiceWithMonitoring(paymentRepo *repositories.PaymentRepository, provider providers.PaymentProvider, encryption *security.EncryptionManager, alertManager *monitoring.AlertManager) *PaymentService {
	return &PaymentService{
		paymentRepo:  paymentRepo,
		provider:     provider,
		encryption:   encryption,
		alertManager: alertManager,
	}
}

func (s *PaymentService) CreateCharge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	startTime := time.Now()
	defer func() {
		monitoring.RecordPaymentMetrics(ctx, req.Amount, req.Currency, "unknown", "processing")
		monitoring.RecordHistogram("payment_processing_duration", float64(time.Since(startTime).Milliseconds()), map[string]string{
			"currency": req.Currency,
		})
	}()

	if err := s.validateChargeRequest(req); err != nil {
		monitoring.IncrementCounter("payment_validation_errors", map[string]string{
			"error_type": "validation",
		})
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
		monitoring.IncrementCounter("payment_provider_errors", map[string]string{
			"error_type": "no_provider",
		})
		return nil, fmt.Errorf("no available provider for currency: %s", req.Currency)
	}

	var payment *models.Payment
	var chargeResp *models.ChargeResponse

	err = s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		retryConfig := utils.DefaultRetryConfig()
		retryConfig.MaxAttempts = 5
		retryConfig.BaseDelay = 200 * time.Millisecond
		retryConfig.MaxDelay = 10 * time.Second

		err := utils.Retry(txCtx, retryConfig, func() error {
			chargeResp, err = s.provider.Charge(txCtx, req)
			return err
		})
		if err != nil {
			monitoring.IncrementCounter("payment_provider_errors", map[string]string{
				"error_type": "provider_failure",
				"provider":   providerName,
			})
			return fmt.Errorf("failed to create charge with provider: %w", err)
		}

		payment = &models.Payment{
			CustomerID:       req.CustomerID,
			Amount:           req.Amount,
			Currency:         req.Currency,
			Status:           models.PaymentStatusSuccess,
			PaymentMethod:    req.PaymentMethod,
			Description:      req.Description,
			ProviderName:     providerName,
			ProviderChargeID: chargeResp.ID,
			Metadata:         req.Metadata,
		}

		if err := s.paymentRepo.Create(txCtx, payment); err != nil {
			monitoring.IncrementCounter("payment_database_errors", map[string]string{
				"error_type": "database_failure",
			})
			return fmt.Errorf("failed to store payment: %w", err)
		}

		return nil
	})

	if err != nil {
		if chargeResp != nil {
			go s.cleanupFailedPayment(ctx, chargeResp, req)
		}
		return nil, err
	}

	monitoring.IncrementCounter("payments_total", map[string]string{
		"currency": req.Currency,
		"provider": providerName,
		"status":   "success",
	})

	monitoring.SetGauge("payment_amount", float64(req.Amount), map[string]string{
		"currency": req.Currency,
		"provider": providerName,
	})

	return s.buildChargeResponse(payment), nil
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	startTime := time.Now()
	defer func() {
		monitoring.RecordHistogram("refund_processing_duration", float64(time.Since(startTime).Milliseconds()), map[string]string{
			"currency": req.Currency,
		})
	}()

	if err := s.validateRefundRequest(req); err != nil {
		monitoring.IncrementCounter("refund_validation_errors", map[string]string{
			"error_type": "validation",
		})
		return nil, err
	}

	var refund *models.Refund
	var refundResp *models.RefundResponse

	err := s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		payment, err := s.paymentRepo.GetByID(txCtx, req.PaymentID)
		if err != nil {
			monitoring.IncrementCounter("refund_database_errors", map[string]string{
				"error_type": "payment_not_found",
			})
			return fmt.Errorf("failed to get payment: %w", err)
		}

		if req.Amount > payment.Amount {
			monitoring.IncrementCounter("refund_validation_errors", map[string]string{
				"error_type": "amount_exceeds_payment",
			})
			return fmt.Errorf("refund amount %d exceeds payment amount %d", req.Amount, payment.Amount)
		}

		retryConfig := utils.DefaultRetryConfig()
		retryConfig.MaxAttempts = 3
		retryConfig.BaseDelay = 500 * time.Millisecond

		err = utils.Retry(txCtx, retryConfig, func() error {
			refundResp, err = s.provider.Refund(txCtx, req)
			return err
		})
		if err != nil {
			monitoring.IncrementCounter("refund_provider_errors", map[string]string{
				"error_type": "provider_failure",
				"provider":   payment.ProviderName,
			})
			return fmt.Errorf("failed to create refund with provider: %w", err)
		}

		payment.Status = models.PaymentStatusRefunded
		if err := s.paymentRepo.Update(txCtx, payment); err != nil {
			monitoring.IncrementCounter("refund_database_errors", map[string]string{
				"error_type": "update_payment_failed",
			})
			return fmt.Errorf("failed to update payment: %w", err)
		}

		refund = &models.Refund{
			PaymentID:        req.PaymentID,
			Amount:           req.Amount,
			Reason:           req.Reason,
			Status:           "succeeded",
			ProviderName:     payment.ProviderName,
			ProviderRefundID: refundResp.ID,
			Metadata:         req.Metadata,
		}

		if err := s.paymentRepo.CreateRefund(txCtx, refund); err != nil {
			monitoring.IncrementCounter("refund_database_errors", map[string]string{
				"error_type": "create_refund_failed",
			})
			return fmt.Errorf("failed to store refund: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	monitoring.IncrementCounter("refunds_total", map[string]string{
		"currency": req.Currency,
		"provider": refund.ProviderName,
		"status":   "success",
	})

	return s.buildRefundResponse(refund, req.Currency), nil
}

func (s *PaymentService) validateChargeRequest(req *models.ChargeRequest) error {
	if req.Amount <= 0 {
		return errors.New("invalid amount")
	}
	if req.Currency == "" {
		return errors.New("invalid currency")
	}
	if req.PaymentMethod == "" {
		return errors.New("invalid payment method")
	}
	if req.CustomerID == "" {
		return errors.New("customer ID is required")
	}
	return nil
}

func (s *PaymentService) validateRefundRequest(req *models.RefundRequest) error {
	if req.Amount <= 0 {
		return errors.New("invalid amount")
	}
	if req.Currency == "" {
		return errors.New("invalid currency")
	}
	if req.PaymentID == "" {
		return errors.New("payment ID is required")
	}
	return nil
}

func (s *PaymentService) selectProvider(ctx context.Context, currency string) string {
	if s.provider.IsAvailable(ctx) {
		switch currency {
		case "USD", "EUR", "GBP":
			return "stripe"
		case "IDR", "SGD", "MYR", "PHP", "THB", "VND":
			return "xendit"
		default:
			return ""
		}
	}
	return ""
}

func (s *PaymentService) generateIdempotencyKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *PaymentService) cleanupFailedPayment(ctx context.Context, chargeResp *models.ChargeResponse, req *models.ChargeRequest) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	refundReq := &models.RefundRequest{
		PaymentID: chargeResp.ID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Reason:    "Transaction rollback due to database error",
	}

	if _, refundErr := s.provider.Refund(cleanupCtx, refundReq); refundErr != nil {
		monitoring.IncrementCounter("payment_cleanup_errors", map[string]string{
			"error_type": "refund_failed",
		})

		s.alertManager.TriggerAlert(&monitoring.Alert{
			Level:   monitoring.Critical,
			Title:   "Payment Cleanup Failed",
			Message: fmt.Sprintf("Failed to reverse provider charge %s: %v", chargeResp.ID, refundErr),
			Source:  "payment_service",
		})
	}
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

func (s *PaymentService) buildRefundResponse(refund *models.Refund, currency string) *models.RefundResponse {
	return &models.RefundResponse{
		ID:               refund.ID,
		PaymentID:        refund.PaymentID,
		Amount:           refund.Amount,
		Currency:         currency,
		Status:           refund.Status,
		Reason:           refund.Reason,
		ProviderName:     refund.ProviderName,
		ProviderRefundID: refund.ProviderRefundID,
		Metadata:         refund.Metadata,
		CreatedAt:        refund.CreatedAt,
	}
}
