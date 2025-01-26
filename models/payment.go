package models

import (
	"time"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID              string        `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerID      string        `json:"customer_id" gorm:"not null;index"`
	Amount          int64         `json:"amount" gorm:"not null"`
	Currency        string        `json:"currency" gorm:"not null"`
	Status          PaymentStatus `json:"status" gorm:"not null;default:'pending'"`
	PaymentMethodID string        `json:"payment_method_id" gorm:"not null"`
	Description     string        `json:"description"`
	ProviderName    string        `json:"provider_name" gorm:"not null"`
	ProviderChargeID string       `json:"provider_charge_id" gorm:"index"`
	Refunds         []Refund      `json:"refunds,omitempty" gorm:"foreignKey:PaymentID"`
	Metadata        gorm.JSON     `json:"metadata" gorm:"type:jsonb"`
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
	Metadata        gorm.JSON `json:"metadata" gorm:"type:jsonb"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type ChargeRequest struct {
	CustomerID      string                 `json:"customer_id" binding:"required"`
	Amount          int64                  `json:"amount" binding:"required"`
	Currency        string                 `json:"currency" binding:"required"`
	PaymentMethodID string                 `json:"payment_method_id" binding:"required"`
	Description     string                 `json:"description,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type ChargeResponse struct {
	Payment *Payment `json:"payment"`
}

type RefundRequest struct {
	PaymentID string                 `json:"payment_id" binding:"required"`
	Amount    int64                  `json:"amount" binding:"required"`
	Reason    string                 `json:"reason,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type RefundResponse struct {
	Refund *Refund `json:"refund"`
}
