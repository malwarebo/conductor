package models

import (
	"time"
)

type WebhookEventStatus string

const (
	WebhookEventStatusPending    WebhookEventStatus = "pending"
	WebhookEventStatusProcessing WebhookEventStatus = "processing"
	WebhookEventStatusCompleted  WebhookEventStatus = "completed"
	WebhookEventStatusFailed     WebhookEventStatus = "failed"
	WebhookEventStatusRetrying   WebhookEventStatus = "retrying"
)

type WebhookEvent struct {
	ID            string             `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      *string            `json:"tenant_id"`
	Provider      string             `json:"provider" gorm:"not null"`
	EventType     string             `json:"event_type" gorm:"not null"`
	EventID       string             `json:"event_id"`
	Payload       JSON               `json:"payload" gorm:"type:jsonb;not null"`
	Status        WebhookEventStatus `json:"status" gorm:"not null;default:'pending'"`
	Attempts      int                `json:"attempts" gorm:"default:0"`
	MaxAttempts   int                `json:"max_attempts" gorm:"default:5"`
	LastAttemptAt *time.Time         `json:"last_attempt_at"`
	NextAttemptAt *time.Time         `json:"next_attempt_at"`
	ProcessedAt   *time.Time         `json:"processed_at"`
	ErrorMessage  string             `json:"error_message"`
	CreatedAt     time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time          `json:"updated_at" gorm:"autoUpdateTime"`
}

type WebhookPayload struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
}

type OutboundWebhook struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature"`
}

