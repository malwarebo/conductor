package providers

import (
	"context"
	"time"
	"github.com/malwarebo/gopay/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/charge"
	"github.com/stripe/stripe-go/v72/refund"
)

type StripeProvider struct {
	apiKey string
}

func NewStripeProvider(apiKey string) *StripeProvider {
	stripe.Key = apiKey
	return &StripeProvider{
		apiKey: apiKey,
	}
}

func (p *StripeProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(req.Amount), // Amount is already in cents
		Currency:    stripe.String(req.Currency),
		Description: stripe.String(req.Description),
		Customer:    stripe.String(req.CustomerID),
	}

	// Set payment method
	if req.PaymentMethod != "" {
		params.SetSource(req.PaymentMethod)
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}

	ch, err := charge.New(params)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	for k, v := range ch.Metadata {
		metadata[k] = v
	}

	return &models.ChargeResponse{
		ID:              ch.ID,
		CustomerID:      req.CustomerID,
		Amount:          ch.Amount,
		Currency:        string(ch.Currency),
		Status:          models.PaymentStatusSuccess,
		PaymentMethod:   ch.Source.ID,
		Description:     req.Description,
		ProviderName:    "stripe",
		ProviderChargeID: ch.ID,
		Metadata:        metadata,
		CreatedAt:       time.Unix(ch.Created, 0),
	}, nil
}

func (p *StripeProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.PaymentID),
		Amount:        stripe.Int64(req.Amount), // Amount is already in cents
		Reason:        stripe.String(req.Reason),
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}

	ref, err := refund.New(params)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	for k, v := range ref.Metadata {
		metadata[k] = v
	}

	return &models.RefundResponse{
		ID:               ref.ID,
		PaymentID:        req.PaymentID,
		Amount:           ref.Amount,
		Currency:         string(ref.Currency),
		Status:           string(ref.Status),
		Reason:           req.Reason,
		ProviderName:     "stripe",
		ProviderRefundID: ref.ID,
		Metadata:         metadata,
		CreatedAt:        time.Unix(ref.Created, 0),
	}, nil
}

func (p *StripeProvider) ValidateWebhookSignature(payload []byte, signature string) error {
	// Implement webhook signature validation
	return nil
}
