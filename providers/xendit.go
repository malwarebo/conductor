package providers

import (
	"context"
	"time"
	"github.com/malwarebo/gopay/models"
	"github.com/xendit/xendit-go/v6"
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

func (p *XenditProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	// Create invoice
	data := xendit.CreateInvoiceData{
		ExternalID:  req.CustomerID,
		Amount:      float64(req.Amount),
		PayerEmail: "customer@example.com",
		Description: req.Description,
	}

	inv, err := xendit.CreateInvoice(&data)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	return &models.ChargeResponse{
		ID:              inv.ID,
		CustomerID:      req.CustomerID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          models.PaymentStatusPending,
		PaymentMethod:   req.PaymentMethod,
		Description:     req.Description,
		ProviderName:    "xendit",
		ProviderChargeID: inv.ID,
		Metadata:        metadata,
		CreatedAt:       time.Now(),
	}, nil
}

func (p *XenditProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	// Expire the invoice as refund
	_, err := xendit.ExpireInvoice(req.PaymentID)
	if err != nil {
		return nil, err
	}

	return &models.RefundResponse{
		ID:               "ref_" + req.PaymentID,
		PaymentID:        req.PaymentID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           "succeeded",
		Reason:           req.Reason,
		ProviderName:     "xendit",
		ProviderRefundID: "ref_" + req.PaymentID,
		Metadata:         req.Metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) IsAvailable(ctx context.Context) bool {
	return true
}
