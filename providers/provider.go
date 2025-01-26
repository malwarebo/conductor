package providers

import (
	"context"
	"github.com/malwarebo/gopay/models"
)

// PaymentProvider defines the interface for payment gateway providers
type PaymentProvider interface {
	// Charge processes a payment charge
	Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error)
	// Refund processes a refund
	Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error)
	// IsAvailable checks if the provider is currently available
	IsAvailable(ctx context.Context) bool
}

type ChargeRequest struct {
	Amount      float64
	Currency    string
	PaymentMethod string
	Description string
	CustomerID  string
	Metadata    map[string]string
}

type ChargeResponse struct {
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	PaymentMethod string
	ProviderName  string
	CreatedAt     int64
	Metadata      map[string]string
}

type RefundRequest struct {
	TransactionID string
	Amount        float64
	Reason        string
	Metadata      map[string]string
}

type RefundResponse struct {
	RefundID      string
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	ProviderName  string
	CreatedAt     int64
	Metadata      map[string]string
}
