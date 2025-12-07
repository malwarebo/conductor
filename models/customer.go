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

type PaymentMethodType string

const (
	PMTypeCard           PaymentMethodType = "card"
	PMTypeBankAccount    PaymentMethodType = "bank_account"
	PMTypeEWallet        PaymentMethodType = "ewallet"
	PMTypeQRCode         PaymentMethodType = "qr_code"
	PMTypeVirtualAccount PaymentMethodType = "virtual_account"
	PMTypeDirectDebit    PaymentMethodType = "direct_debit"
	PMTypeRetail         PaymentMethodType = "retail"
)

type PaymentMethod struct {
	ID                      string            `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerID              string            `json:"customer_id" gorm:"not null;index"`
	ProviderName            string            `json:"provider_name" gorm:"not null"`
	ProviderPaymentMethodID string            `json:"provider_payment_method_id" gorm:"not null"`
	Type                    PaymentMethodType `json:"type" gorm:"not null"`
	Reusable                bool              `json:"reusable" gorm:"default:true"`
	Status                  string            `json:"status" gorm:"default:'active'"`
	Last4                   string            `json:"last4,omitempty"`
	Brand                   string            `json:"brand,omitempty"`
	ExpMonth                int               `json:"exp_month,omitempty"`
	ExpYear                 int               `json:"exp_year,omitempty"`
	BankCode                string            `json:"bank_code,omitempty"`
	AccountName             string            `json:"account_name,omitempty"`
	ChannelCode             string            `json:"channel_code,omitempty"`
	IsDefault               bool              `json:"is_default" gorm:"default:false"`
	Metadata                JSON              `json:"metadata" gorm:"type:jsonb"`
	CreatedAt               time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreatePaymentMethodRequest struct {
	CustomerID   string                 `json:"customer_id" binding:"required"`
	Type         PaymentMethodType      `json:"type" binding:"required"`
	Reusable     bool                   `json:"reusable"`
	ChannelCode  string                 `json:"channel_code,omitempty"`
	CardToken    string                 `json:"card_token,omitempty"`
	ReturnURL    string                 `json:"return_url,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	IsDefault    bool                   `json:"is_default"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type PaymentMethodResponse struct {
	PaymentMethod *PaymentMethod `json:"payment_method"`
}

type PaymentMethodListResponse struct {
	PaymentMethods []*PaymentMethod `json:"payment_methods"`
	Total          int              `json:"total"`
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
