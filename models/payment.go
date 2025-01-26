package models

import "time"

// ChargeRequest represents a payment charge request
type ChargeRequest struct {
	Amount        float64            `json:"amount"`
	Currency      string             `json:"currency"`
	PaymentMethod string             `json:"payment_method"`
	Description   string             `json:"description"`
	CustomerID    string             `json:"customer_id"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}

// ChargeResponse represents a payment charge response
type ChargeResponse struct {
	TransactionID string             `json:"transaction_id"`
	Status        string             `json:"status"`
	Amount        float64            `json:"amount"`
	Currency      string             `json:"currency"`
	PaymentMethod string             `json:"payment_method"`
	ProviderName  string             `json:"provider_name"`
	CreatedAt     time.Time          `json:"created_at"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}

// RefundRequest represents a refund request
type RefundRequest struct {
	TransactionID string             `json:"transaction_id"`
	Amount        float64            `json:"amount"`
	Reason        string             `json:"reason"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}

// RefundResponse represents a refund response
type RefundResponse struct {
	RefundID      string             `json:"refund_id"`
	TransactionID string             `json:"transaction_id"`
	Status        string             `json:"status"`
	Amount        float64            `json:"amount"`
	Currency      string             `json:"currency"`
	ProviderName  string             `json:"provider_name"`
	CreatedAt     time.Time          `json:"created_at"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
}
