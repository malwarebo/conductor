package providers

import (
	"context"
	"fmt"
	"time"
	
	"github.com/malwarebo/gopay/models"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/charge"
	"github.com/stripe/stripe-go/v76/refund"
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

func (s *StripeProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(int64(req.Amount * 100)), // Convert to cents
		Currency:    stripe.String(req.Currency),
		Description: stripe.String(req.Description),
	}

	if req.PaymentMethod != "" {
		params.Source = &stripe.SourceParams{Token: stripe.String(req.PaymentMethod)}
	}

	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}

	// Add metadata
	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			params.Metadata[k] = v
		}
	}

	ch, err := charge.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe charge failed: %w", err)
	}

	return &models.ChargeResponse{
		TransactionID: ch.ID,
		Status:        string(ch.Status),
		Amount:        float64(ch.Amount) / 100,
		Currency:      string(ch.Currency),
		PaymentMethod: string(ch.PaymentMethod),
		ProviderName:  "stripe",
		CreatedAt:     time.Unix(ch.Created, 0),
		Metadata:      ch.Metadata,
	}, nil
}

func (s *StripeProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	params := &stripe.RefundParams{
		Charge:   stripe.String(req.TransactionID),
		Amount:   stripe.Int64(int64(req.Amount * 100)), // Convert to cents
		Reason:   stripe.String(req.Reason),
		Metadata: req.Metadata,
	}

	ref, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe refund failed: %w", err)
	}

	return &models.RefundResponse{
		RefundID:      ref.ID,
		TransactionID: ref.Charge.ID,
		Status:        string(ref.Status),
		Amount:        float64(ref.Amount) / 100,
		Currency:      string(ref.Currency),
		ProviderName:  "stripe",
		CreatedAt:     time.Unix(ref.Created, 0),
		Metadata:      ref.Metadata,
	}, nil
}

func (s *StripeProvider) IsAvailable(ctx context.Context) bool {
	// Simple health check by attempting to retrieve account details
	_, err := stripe.Account.Get()
	return err == nil
}
