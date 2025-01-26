package models

import (
	"time"
)

type DisputeStatus string
type DisputeReason string

const (
	DisputeStatusOpen       DisputeStatus = "open"
	DisputeStatusUnderReview DisputeStatus = "under_review"
	DisputeStatusWon        DisputeStatus = "won"
	DisputeStatusLost       DisputeStatus = "lost"
	DisputeStatusCanceled   DisputeStatus = "canceled"

	DisputeReasonFraudulent        DisputeReason = "fraudulent"
	DisputeReasonDuplicate         DisputeReason = "duplicate"
	DisputeReasonProductNotReceived DisputeReason = "product_not_received"
	DisputeReasonProductUnacceptable DisputeReason = "product_unacceptable"
	DisputeReasonCreditNotProcessed DisputeReason = "credit_not_processed"
	DisputeReasonGeneral           DisputeReason = "general"
)

type Dispute struct {
	ID              string         `json:"id"`
	TransactionID   string         `json:"transaction_id"`
	CustomerID      string         `json:"customer_id"`
	Amount          float64        `json:"amount"`
	Currency        string         `json:"currency"`
	Status          DisputeStatus  `json:"status"`
	Reason          DisputeReason  `json:"reason"`
	Evidence        []Evidence     `json:"evidence"`
	DueBy           time.Time      `json:"due_by"`
	ProviderName    string         `json:"provider_name"`
	ProviderDispute interface{}    `json:"provider_dispute"` // Provider-specific dispute data
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Evidence struct {
	ID          string    `json:"id"`
	DisputeID   string    `json:"dispute_id"`
	Type        string    `json:"type"` // shipping_documentation, service_documentation, etc.
	Description string    `json:"description"`
	FileURL     string    `json:"file_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateDisputeRequest struct {
	TransactionID string         `json:"transaction_id"`
	Reason        DisputeReason  `json:"reason"`
	Description   string         `json:"description"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type UpdateDisputeRequest struct {
	Status      DisputeStatus  `json:"status,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SubmitEvidenceRequest struct {
	DisputeID   string    `json:"dispute_id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	FileData    []byte    `json:"file_data,omitempty"`
}

type DisputeEvent struct {
	ID         string      `json:"id"`
	DisputeID  string      `json:"dispute_id"`
	Type       string      `json:"type"` // created, updated, evidence_added, won, lost, etc.
	Data       interface{} `json:"data"`
	CreatedAt  time.Time   `json:"created_at"`
}

type DisputeStats struct {
	TotalDisputes    int     `json:"total_disputes"`
	OpenDisputes     int     `json:"open_disputes"`
	WonDisputes      int     `json:"won_disputes"`
	LostDisputes     int     `json:"lost_disputes"`
	DisputeRatio     float64 `json:"dispute_ratio"` // Disputes / Total Transactions
	AverageResolutionTime float64 `json:"average_resolution_time"` // In days
}
