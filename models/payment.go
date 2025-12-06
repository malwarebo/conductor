package models

import (
	"time"
)

type PaymentStatus string
type CaptureMethod string

const (
	PaymentStatusPending          PaymentStatus = "pending"
	PaymentStatusRequiresAction   PaymentStatus = "requires_action"
	PaymentStatusRequiresCapture  PaymentStatus = "requires_capture"
	PaymentStatusProcessing       PaymentStatus = "processing"
	PaymentStatusSuccess          PaymentStatus = "succeeded"
	PaymentStatusFailed           PaymentStatus = "failed"
	PaymentStatusCanceled         PaymentStatus = "canceled"
	PaymentStatusRefunded         PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentStatusDisputed         PaymentStatus = "disputed"

	CaptureMethodAutomatic CaptureMethod = "automatic"
	CaptureMethodManual    CaptureMethod = "manual"
)

type Payment struct {
	ID               string        `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         *string       `json:"tenant_id" gorm:"index"`
	CustomerID       string        `json:"customer_id" gorm:"not null;index"`
	Amount           int64         `json:"amount" gorm:"not null"`
	Currency         string        `json:"currency" gorm:"not null"`
	Status           PaymentStatus `json:"status" gorm:"not null;default:'pending'"`
	PaymentMethod    string        `json:"payment_method" gorm:"not null"`
	Description      string        `json:"description"`
	ProviderName     string        `json:"provider_name" gorm:"not null"`
	ProviderChargeID string        `json:"provider_charge_id" gorm:"index"`
	CaptureMethod    CaptureMethod `json:"capture_method" gorm:"default:'automatic'"`
	CapturedAmount   int64         `json:"captured_amount" gorm:"default:0"`
	RequiresAction   bool          `json:"requires_action" gorm:"default:false"`
	NextActionType   string        `json:"next_action_type"`
	NextActionURL    string        `json:"next_action_url"`
	IdempotencyKey   string        `json:"idempotency_key" gorm:"index"`
	ClientSecret     string        `json:"client_secret,omitempty"`
	Metadata         JSON          `json:"metadata" gorm:"type:jsonb"`
	CreatedAt        time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
}

type Refund struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PaymentID       string    `json:"payment_id" gorm:"not null;index"`
	Amount          int64     `json:"amount" gorm:"not null"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status" gorm:"not null;default:'pending'"`
	ProviderName    string    `json:"provider_name" gorm:"not null"`
	ProviderRefundID string   `json:"provider_refund_id" gorm:"index"`
	Metadata        JSON      `json:"metadata" gorm:"type:jsonb"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type ChargeRequest struct {
	CustomerID     string        `json:"customer_id"`
	Amount         int64         `json:"amount"`
	Currency       string        `json:"currency"`
	PaymentMethod  string        `json:"payment_method"`
	Description    string        `json:"description"`
	CaptureMethod  CaptureMethod `json:"capture_method,omitempty"`
	Capture        *bool         `json:"capture,omitempty"`
	ReturnURL      string        `json:"return_url,omitempty"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
	Metadata       JSON          `json:"metadata,omitempty"`
}

type AuthorizeRequest struct {
	CustomerID     string `json:"customer_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	PaymentMethod  string `json:"payment_method"`
	Description    string `json:"description"`
	ReturnURL      string `json:"return_url,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	Metadata       JSON   `json:"metadata,omitempty"`
}

type CaptureRequest struct {
	PaymentID      string `json:"payment_id"`
	Amount         int64  `json:"amount,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type VoidRequest struct {
	PaymentID      string `json:"payment_id"`
	Reason         string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type Confirm3DSRequest struct {
	PaymentID string `json:"payment_id"`
}

type ChargeResponse struct {
	ID               string        `json:"id"`
	CustomerID       string        `json:"customer_id"`
	Amount           int64         `json:"amount"`
	Currency         string        `json:"currency"`
	Status           PaymentStatus `json:"status"`
	PaymentMethod    string        `json:"payment_method"`
	Description      string        `json:"description"`
	ProviderName     string        `json:"provider_name"`
	ProviderChargeID string        `json:"provider_charge_id"`
	CaptureMethod    CaptureMethod `json:"capture_method,omitempty"`
	CapturedAmount   int64         `json:"captured_amount,omitempty"`
	RequiresAction   bool          `json:"requires_action,omitempty"`
	NextActionType   string        `json:"next_action_type,omitempty"`
	NextActionURL    string        `json:"next_action_url,omitempty"`
	ClientSecret     string        `json:"client_secret,omitempty"`
	Metadata         JSON          `json:"metadata,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
}

type CaptureResponse struct {
	ID             string        `json:"id"`
	PaymentID      string        `json:"payment_id"`
	Amount         int64         `json:"amount"`
	Status         PaymentStatus `json:"status"`
	ProviderName   string        `json:"provider_name"`
	CapturedAt     time.Time     `json:"captured_at"`
}

type VoidResponse struct {
	ID           string        `json:"id"`
	PaymentID    string        `json:"payment_id"`
	Status       PaymentStatus `json:"status"`
	ProviderName string        `json:"provider_name"`
	VoidedAt     time.Time     `json:"voided_at"`
}

type PaymentIntent struct {
	ID             string                 `json:"id"`
	Amount         int64                  `json:"amount"`
	Currency       string                 `json:"currency"`
	Status         string                 `json:"status"`
	CustomerID     string                 `json:"customer_id,omitempty"`
	PaymentMethod  string                 `json:"payment_method,omitempty"`
	Description    string                 `json:"description,omitempty"`
	ClientSecret   string                 `json:"client_secret"`
	CaptureMethod  string                 `json:"capture_method"`
	CapturedAmount int64                  `json:"captured_amount"`
	RequiresAction bool                   `json:"requires_action"`
	NextActionType string                 `json:"next_action_type,omitempty"`
	NextActionURL  string                 `json:"next_action_url,omitempty"`
	ProviderName   string                 `json:"provider_name"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type CreatePaymentIntentRequest struct {
	Amount           int64         `json:"amount"`
	Currency         string        `json:"currency"`
	CustomerID       string        `json:"customer_id,omitempty"`
	PaymentMethod    string        `json:"payment_method,omitempty"`
	Description      string        `json:"description,omitempty"`
	CaptureMethod    CaptureMethod `json:"capture_method,omitempty"`
	SetupFutureUsage string        `json:"setup_future_usage,omitempty"`
	ReturnURL        string        `json:"return_url,omitempty"`
	Metadata         JSON          `json:"metadata,omitempty"`
}

type UpdatePaymentIntentRequest struct {
	Amount        *int64 `json:"amount,omitempty"`
	Currency      *string `json:"currency,omitempty"`
	Description   *string `json:"description,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	Metadata      JSON    `json:"metadata,omitempty"`
}

type ConfirmPaymentIntentRequest struct {
	PaymentMethod string `json:"payment_method,omitempty"`
	ReturnURL     string `json:"return_url,omitempty"`
}

type ListPaymentIntentsRequest struct {
	CustomerID string `json:"customer_id,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type RefundRequest struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Reason    string `json:"reason,omitempty"`
	Metadata  JSON   `json:"metadata,omitempty"`
}

type RefundResponse struct {
	ID              string    `json:"id"`
	PaymentID       string    `json:"payment_id"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason"`
	ProviderName    string    `json:"provider_name"`
	ProviderRefundID string   `json:"provider_refund_id"`
	Metadata        JSON      `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
