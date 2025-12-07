package models

import (
	"time"
)

type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusSucceeded  PayoutStatus = "succeeded"
	PayoutStatusFailed     PayoutStatus = "failed"
	PayoutStatusCanceled   PayoutStatus = "canceled"
	PayoutStatusReversed   PayoutStatus = "reversed"
)

type DestinationType string

const (
	DestinationBankAccount    DestinationType = "bank_account"
	DestinationCard           DestinationType = "card"
	DestinationEWallet        DestinationType = "ewallet"
)

type Payout struct {
	ID                   string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             *string                `json:"tenant_id" gorm:"index"`
	ReferenceID          string                 `json:"reference_id" gorm:"index"`
	ProviderID           string                 `json:"provider_id" gorm:"index"`
	ProviderName         string                 `json:"provider_name" gorm:"not null"`
	Amount               int64                  `json:"amount" gorm:"not null"`
	Currency             string                 `json:"currency" gorm:"not null"`
	Status               PayoutStatus           `json:"status" gorm:"not null;default:'pending'"`
	Description          string                 `json:"description"`
	DestinationType      DestinationType        `json:"destination_type" gorm:"not null"`
	DestinationAccount   string                 `json:"destination_account"`
	DestinationName      string                 `json:"destination_name"`
	DestinationBank      string                 `json:"destination_bank"`
	DestinationChannel   string                 `json:"destination_channel"`
	FailureReason        string                 `json:"failure_reason"`
	EstimatedArrival     *time.Time             `json:"estimated_arrival"`
	Metadata             JSON                   `json:"metadata" gorm:"type:jsonb"`
	CreatedAt            time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreatePayoutRequest struct {
	ReferenceID        string                 `json:"reference_id"`
	Amount             int64                  `json:"amount"`
	Currency           string                 `json:"currency"`
	Description        string                 `json:"description,omitempty"`
	DestinationType    DestinationType        `json:"destination_type"`
	DestinationAccount string                 `json:"destination_account"`
	DestinationName    string                 `json:"destination_name,omitempty"`
	DestinationBank    string                 `json:"destination_bank,omitempty"`
	DestinationChannel string                 `json:"destination_channel,omitempty"`
	Provider           string                 `json:"provider,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type ListPayoutsRequest struct {
	ReferenceID string `json:"reference_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

type PayoutResponse struct {
	Payout *Payout `json:"payout"`
}

type PayoutListResponse struct {
	Payouts []*Payout `json:"payouts"`
	Total   int       `json:"total"`
}

type PayoutChannel struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Currency    string  `json:"currency"`
	MinAmount   float64 `json:"min_amount"`
	MaxAmount   float64 `json:"max_amount"`
}

