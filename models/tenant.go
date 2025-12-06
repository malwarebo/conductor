package models

import (
	"time"
)

type Tenant struct {
	ID            string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name          string                 `json:"name" gorm:"not null"`
	APIKey        string                 `json:"api_key" gorm:"uniqueIndex;not null"`
	APISecret     string                 `json:"-" gorm:"not null"`
	WebhookURL    string                 `json:"webhook_url"`
	WebhookSecret string                 `json:"-"`
	IsActive      bool                   `json:"is_active" gorm:"default:true"`
	Settings      map[string]interface{} `json:"settings" gorm:"type:jsonb;default:'{}'"`
	Metadata      map[string]interface{} `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

type TenantSettings struct {
	DefaultProvider     string   `json:"default_provider"`
	EnabledProviders    []string `json:"enabled_providers"`
	Enable3DS           bool     `json:"enable_3ds"`
	DefaultCaptureMethod string  `json:"default_capture_method"`
	WebhookRetryCount   int      `json:"webhook_retry_count"`
}

type CreateTenantRequest struct {
	Name       string                 `json:"name" binding:"required"`
	WebhookURL string                 `json:"webhook_url"`
	Settings   map[string]interface{} `json:"settings"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type UpdateTenantRequest struct {
	Name          string                 `json:"name"`
	WebhookURL    string                 `json:"webhook_url"`
	WebhookSecret string                 `json:"webhook_secret"`
	IsActive      *bool                  `json:"is_active"`
	Settings      map[string]interface{} `json:"settings"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type TenantResponse struct {
	Tenant *Tenant `json:"tenant"`
}

