package models

import (
	"time"
)

type IdempotencyKey struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Key          string    `json:"key" gorm:"not null"`
	TenantID     *string   `json:"tenant_id"`
	RequestPath  string    `json:"request_path" gorm:"not null"`
	RequestHash  string    `json:"request_hash" gorm:"not null"`
	ResponseCode *int      `json:"response_code"`
	ResponseBody JSON      `json:"response_body" gorm:"type:jsonb"`
	LockedAt     *time.Time `json:"locked_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"not null"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

type IdempotencyResult struct {
	IsNew        bool
	Key          *IdempotencyKey
	ResponseCode int
	ResponseBody []byte
}

