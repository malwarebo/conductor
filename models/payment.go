package models

import (
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSuccess   PaymentStatus = "success"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID              string        `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerID      string        `json:"customer_id" gorm:"not null;index"`
	Amount          int64         `json:"amount" gorm:"not null"`
	Currency        string        `json:"currency" gorm:"not null"`
	Status          PaymentStatus `json:"status" gorm:"not null;default:'pending'"`
	PaymentMethod   string        `json:"payment_method" gorm:"not null"`
	Description     string        `json:"description"`
	ProviderName    string        `json:"provider_name" gorm:"not null"`
	ProviderChargeID string       `json:"provider_charge_id" gorm:"index"`
	Metadata        JSON          `json:"metadata" gorm:"type:jsonb"`
	CreatedAt       time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
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
	CustomerID    string `json:"customer_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	Description   string `json:"description"`
	Metadata      JSON   `json:"metadata,omitempty"`
}

type ChargeResponse struct {
	ID              string        `json:"id"`
	CustomerID      string        `json:"customer_id"`
	Amount          int64         `json:"amount"`
	Currency        string        `json:"currency"`
	Status          PaymentStatus `json:"status"`
	PaymentMethod   string        `json:"payment_method"`
	Description     string        `json:"description"`
	ProviderName    string        `json:"provider_name"`
	ProviderChargeID string       `json:"provider_charge_id"`
	Metadata        JSON          `json:"metadata,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
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
