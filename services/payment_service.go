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

	providerName := s.getProviderName(ctx, req.Currency)
	if providerName == "" {
		return nil, fmt.Errorf("no available provider for currency: %s", req.Currency)
	}

	var payment *models.Payment
	var chargeResp *models.ChargeResponse

	err := s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		var err error
		chargeResp, err = s.provider.Charge(txCtx, req)
		if err != nil {
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
			return fmt.Errorf("failed to store payment: %w", err)
		}

		return nil
	})

	if err != nil {
		if chargeResp != nil {
			go func() {
				cleanupCtx := context.Background()
				refundReq := &models.RefundRequest{
					PaymentID: chargeResp.ID,
					Amount:    req.Amount,
					Currency:  req.Currency,
					Reason:    "Transaction rollback due to database error",
				}
				if _, refundErr := s.provider.Refund(cleanupCtx, refundReq); refundErr != nil {
					fmt.Printf("Warning: Failed to reverse provider charge %s: %v\n", chargeResp.ID, refundErr)
				}
			}()
		}
		return nil, err
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

	var refund *models.Refund
	var refundResp *models.RefundResponse

	err := s.paymentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		payment, err := s.paymentRepo.GetByID(txCtx, req.PaymentID)
		if err != nil {
			return fmt.Errorf("failed to get payment: %w", err)
		}

		if req.Amount > payment.Amount {
			return fmt.Errorf("refund amount %d exceeds payment amount %d", req.Amount, payment.Amount)
		}

		refundResp, err = s.provider.Refund(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create refund with provider: %w", err)
		}

		payment.Status = models.PaymentStatusRefunded
		if err := s.paymentRepo.Update(txCtx, payment); err != nil {
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
			return fmt.Errorf("failed to store refund: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
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

func (s *PaymentService) getProviderName(ctx context.Context, currency string) string {
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
