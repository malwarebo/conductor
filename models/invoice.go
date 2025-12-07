package models

import (
	"time"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusExpired   InvoiceStatus = "expired"
	InvoiceStatusCanceled  InvoiceStatus = "canceled"
	InvoiceStatusVoid      InvoiceStatus = "void"
)

type Invoice struct {
	ID                 string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           *string                `json:"tenant_id" gorm:"index"`
	ExternalID         string                 `json:"external_id" gorm:"index"`
	ProviderID         string                 `json:"provider_id" gorm:"index"`
	ProviderName       string                 `json:"provider_name" gorm:"not null"`
	CustomerID         string                 `json:"customer_id" gorm:"index"`
	CustomerEmail      string                 `json:"customer_email"`
	Amount             int64                  `json:"amount" gorm:"not null"`
	Currency           string                 `json:"currency" gorm:"not null"`
	Status             InvoiceStatus          `json:"status" gorm:"not null;default:'pending'"`
	Description        string                 `json:"description"`
	InvoiceURL         string                 `json:"invoice_url"`
	DueDate            *time.Time             `json:"due_date"`
	PaidAt             *time.Time             `json:"paid_at"`
	SuccessRedirectURL string                 `json:"success_redirect_url"`
	FailureRedirectURL string                 `json:"failure_redirect_url"`
	PaymentMethods     []string               `json:"payment_methods" gorm:"type:text[]"`
	Metadata           JSON                   `json:"metadata" gorm:"type:jsonb"`
	CreatedAt          time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreateInvoiceRequest struct {
	ExternalID         string                 `json:"external_id"`
	CustomerID         string                 `json:"customer_id,omitempty"`
	CustomerEmail      string                 `json:"customer_email,omitempty"`
	Amount             int64                  `json:"amount"`
	Currency           string                 `json:"currency"`
	Description        string                 `json:"description,omitempty"`
	DueDate            *time.Time             `json:"due_date,omitempty"`
	DurationSeconds    int                    `json:"duration_seconds,omitempty"`
	SuccessRedirectURL string                 `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string                 `json:"failure_redirect_url,omitempty"`
	PaymentMethods     []string               `json:"payment_methods,omitempty"`
	SendEmail          bool                   `json:"send_email,omitempty"`
	Provider           string                 `json:"provider,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type ListInvoicesRequest struct {
	CustomerID string `json:"customer_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type InvoiceResponse struct {
	Invoice *Invoice `json:"invoice"`
}

type InvoiceListResponse struct {
	Invoices []*Invoice `json:"invoices"`
	Total    int        `json:"total"`
}

