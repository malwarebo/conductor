package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
)

type AuditService struct {
	store *stores.AuditStore
}

func CreateAuditService(store *stores.AuditStore) *AuditService {
	return &AuditService{store: store}
}

func (s *AuditService) LogAction(ctx context.Context, log *models.AuditLog) error {
	return s.store.Create(ctx, log)
}

func (s *AuditService) LogPaymentAction(ctx context.Context, tenantID, userID, action, paymentID, ip, userAgent string, success bool, errMsg string, metadata map[string]interface{}) error {
	log := &models.AuditLog{
		TenantID:     stringPtr(tenantID),
		UserID:       userID,
		Action:       action,
		ResourceType: string(models.AuditResourcePayment),
		ResourceID:   paymentID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Success:      success,
		ErrorMessage: errMsg,
		Metadata:     metadata,
	}
	return s.store.Create(ctx, log)
}

func (s *AuditService) LogAPIRequest(ctx context.Context, tenantID, userID, method, path, ip, userAgent string, requestBody interface{}, responseCode int, success bool, errMsg string) error {
	var reqBodyJSON models.JSON
	if requestBody != nil {
		if m, ok := requestBody.(map[string]interface{}); ok {
			reqBodyJSON = models.JSON(m)
		} else {
			bytes, _ := json.Marshal(requestBody)
			json.Unmarshal(bytes, &reqBodyJSON)
		}
	}

	log := &models.AuditLog{
		TenantID:      stringPtr(tenantID),
		UserID:        userID,
		Action:        "api_request",
		ResourceType:  "api",
		RequestMethod: method,
		RequestPath:   path,
		RequestBody:   reqBodyJSON,
		ResponseCode:  responseCode,
		IPAddress:     ip,
		UserAgent:     userAgent,
		Success:       success,
		ErrorMessage:  errMsg,
	}
	return s.store.Create(ctx, log)
}

func (s *AuditService) LogWebhookEvent(ctx context.Context, tenantID, provider, eventType, eventID string, success bool, errMsg string) error {
	log := &models.AuditLog{
		TenantID:     stringPtr(tenantID),
		Action:       string(models.AuditActionWebhook),
		ResourceType: string(models.AuditResourceWebhook),
		ResourceID:   eventID,
		Success:      success,
		ErrorMessage: errMsg,
		Metadata: map[string]interface{}{
			"provider":   provider,
			"event_type": eventType,
		},
	}
	return s.store.Create(ctx, log)
}

func (s *AuditService) GetAuditLogs(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int64, error) {
	return s.store.List(ctx, filter)
}

func (s *AuditService) GetResourceHistory(ctx context.Context, resourceType, resourceID string, limit int) ([]*models.AuditLog, error) {
	return s.store.ListByResource(ctx, resourceType, resourceID, limit)
}

func (s *AuditService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	retention := time.Duration(retentionDays) * 24 * time.Hour
	return s.store.CleanupOld(ctx, retention)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

