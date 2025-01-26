package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/xendit/xendit-go"
	"github.com/xendit/xendit-go/charge"
)

type XenditProvider struct {
	apiKey string
}

func NewXenditProvider(apiKey string) *XenditProvider {
	xendit.Opt.SecretKey = apiKey
	return &XenditProvider{
		apiKey: apiKey,
	}
}

func (x *XenditProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	chargeParams := charge.CreateParams{
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		Description:   req.Description,
		CustomerID:    req.CustomerID,
		Metadata:      req.Metadata,
	}

	ch, err := charge.Create(&chargeParams)
	if err != nil {
		return nil, fmt.Errorf("xendit charge failed: %w", err)
	}

	return &models.ChargeResponse{
		TransactionID: ch.ID,
		Status:        ch.Status,
		Amount:        ch.Amount,
		Currency:      ch.Currency,
		PaymentMethod: ch.PaymentMethod,
		ProviderName:  "xendit",
		CreatedAt:     time.Now(), // Xendit API might provide this, adjust accordingly
		Metadata:      ch.Metadata,
	}, nil
}

func (x *XenditProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	refundParams := charge.CreateRefundParams{
		ChargeID: req.TransactionID,
		Amount:   req.Amount,
		Metadata: req.Metadata,
	}

	ref, err := charge.CreateRefund(&refundParams)
	if err != nil {
		return nil, fmt.Errorf("xendit refund failed: %w", err)
	}

	return &models.RefundResponse{
		RefundID:      ref.ID,
		TransactionID: ref.ChargeID,
		Status:        ref.Status,
		Amount:        ref.Amount,
		Currency:      ref.Currency,
		ProviderName:  "xendit",
		CreatedAt:     time.Now(), // Xendit API might provide this, adjust accordingly
		Metadata:      ref.Metadata,
	}, nil
}

func (x *XenditProvider) IsAvailable(ctx context.Context) bool {
	// Simple health check by attempting to get balance
	_, err := xendit.Balance.Get()
	return err == nil
}
