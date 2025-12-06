package models

import (
	"time"
)

type AuditLog struct {
	ID            string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      *string                `json:"tenant_id"`
	UserID        string                 `json:"user_id"`
	Action        string                 `json:"action" gorm:"not null"`
	ResourceType  string                 `json:"resource_type" gorm:"not null"`
	ResourceID    string                 `json:"resource_id"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	RequestMethod string                 `json:"request_method"`
	RequestPath   string                 `json:"request_path"`
	RequestBody   JSON                   `json:"request_body" gorm:"type:jsonb"`
	ResponseCode  int                    `json:"response_code"`
	Success       bool                   `json:"success" gorm:"not null"`
	ErrorMessage  string                 `json:"error_message"`
	Metadata      map[string]interface{} `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     time.Time              `json:"created_at" gorm:"autoCreateTime"`
}

type AuditAction string

const (
	AuditActionCreate       AuditAction = "create"
	AuditActionUpdate       AuditAction = "update"
	AuditActionDelete       AuditAction = "delete"
	AuditActionRead         AuditAction = "read"
	AuditActionCharge       AuditAction = "charge"
	AuditActionAuthorize    AuditAction = "authorize"
	AuditActionCapture      AuditAction = "capture"
	AuditActionRefund       AuditAction = "refund"
	AuditActionVoid         AuditAction = "void"
	AuditAction3DSChallenge AuditAction = "3ds_challenge"
	AuditActionWebhook      AuditAction = "webhook"
	AuditActionLogin        AuditAction = "login"
	AuditActionLogout       AuditAction = "logout"
)

type AuditResourceType string

const (
	AuditResourcePayment      AuditResourceType = "payment"
	AuditResourceRefund       AuditResourceType = "refund"
	AuditResourceSubscription AuditResourceType = "subscription"
	AuditResourceDispute      AuditResourceType = "dispute"
	AuditResourceCustomer     AuditResourceType = "customer"
	AuditResourceTenant       AuditResourceType = "tenant"
	AuditResourceWebhook      AuditResourceType = "webhook"
)

type AuditLogFilter struct {
	TenantID     string
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

