package models

import (
	"time"
)

type DisputeStatus string

const (
	DisputeStatusOpen     DisputeStatus = "open"
	DisputeStatusWon      DisputeStatus = "won"
	DisputeStatusLost     DisputeStatus = "lost"
	DisputeStatusCanceled DisputeStatus = "canceled"
)

type Dispute struct {
	ID             string        `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CustomerID     string        `json:"customer_id" gorm:"not null;index"`
	TransactionID  string        `json:"transaction_id" gorm:"not null;index"`
	Amount         int64         `json:"amount" gorm:"not null"`
	Currency       string        `json:"currency" gorm:"not null"`
	Reason         string        `json:"reason" gorm:"not null"`
	Status         DisputeStatus `json:"status" gorm:"not null;default:'open'"`
	Evidence       map[string]interface{} `json:"evidence" gorm:"type:jsonb"`
	DueBy          time.Time     `json:"due_by" gorm:"not null"`
	ClosedAt       *time.Time    `json:"closed_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt      time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
}

type Evidence struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DisputeID   string    `json:"dispute_id" gorm:"not null;index"`
	Type        string    `json:"type" gorm:"not null"`
	Description string    `json:"description" gorm:"not null"`
	Files       []string  `json:"files,omitempty" gorm:"type:text[]"`
	Metadata    map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CreateDisputeRequest struct {
	CustomerID    string                 `json:"customer_id" binding:"required"`
	TransactionID string                 `json:"transaction_id" binding:"required"`
	Amount        int64                  `json:"amount" binding:"required"`
	Currency      string                 `json:"currency" binding:"required"`
	Reason        string                 `json:"reason" binding:"required"`
	DueBy         time.Time              `json:"due_by" binding:"required"`
	Evidence      map[string]interface{} `json:"evidence,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateDisputeRequest struct {
	Status   DisputeStatus `json:"status,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SubmitEvidenceRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description" binding:"required"`
	Files       []string               `json:"files,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type DisputeResponse struct {
	Dispute *Dispute `json:"dispute"`
}

type DisputeStats struct {
	Total     int64 `json:"total"`
	Open      int64 `json:"open"`
	Won       int64 `json:"won"`
	Lost      int64 `json:"lost"`
	Canceled  int64 `json:"canceled"`
}
