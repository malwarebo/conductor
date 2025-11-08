package models

import (
	"time"
)

type Customer struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ExternalID string    `json:"external_id" gorm:"uniqueIndex;not null"`
	Email      string    `json:"email" gorm:"not null;index"`
	Name       string    `json:"name"`
	Phone      string    `json:"phone"`
	Metadata   JSON      `json:"metadata" gorm:"type:jsonb"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreateCustomerRequest struct {
	ExternalID string                 `json:"external_id" binding:"required"`
	Email      string                 `json:"email" binding:"required,email"`
	Name       string                 `json:"name"`
	Phone      string                 `json:"phone"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateCustomerRequest struct {
	Email    string                 `json:"email,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentMethod struct {
	ID                       string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerID               string    `json:"customer_id" gorm:"not null;index"`
	ProviderName             string    `json:"provider_name" gorm:"not null"`
	ProviderPaymentMethodID  string    `json:"provider_payment_method_id" gorm:"not null"`
	Type                     string    `json:"type" gorm:"not null"`
	Last4                    string    `json:"last4,omitempty"`
	Brand                    string    `json:"brand,omitempty"`
	ExpMonth                 int       `json:"exp_month,omitempty"`
	ExpYear                  int       `json:"exp_year,omitempty"`
	IsDefault                bool      `json:"is_default" gorm:"default:false"`
	Metadata                 JSON      `json:"metadata" gorm:"type:jsonb"`
	CreatedAt                time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreatePaymentMethodRequest struct {
	CustomerID      string                 `json:"customer_id" binding:"required"`
	ProviderName    string                 `json:"provider_name" binding:"required"`
	PaymentMethodID string                 `json:"payment_method_id" binding:"required"`
	Type            string                 `json:"type" binding:"required"`
	IsDefault       bool                   `json:"is_default"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type ProviderMapping struct {
	ID               string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EntityID         string    `json:"entity_id" gorm:"not null;index:idx_entity"`
	EntityType       string    `json:"entity_type" gorm:"not null;index:idx_entity"`
	ProviderName     string    `json:"provider_name" gorm:"not null;index"`
	ProviderEntityID string    `json:"provider_entity_id" gorm:"not null"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

