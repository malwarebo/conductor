package stores

import (
	"context"
	"time"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type AuditStore struct {
	BaseStore
}

func CreateAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{BaseStore: BaseStore{db: db}}
}

func (s *AuditStore) Create(ctx context.Context, log *models.AuditLog) error {
	return s.GetDB(ctx).Create(log).Error
}

func (s *AuditStore) GetByID(ctx context.Context, id string) (*models.AuditLog, error) {
	var log models.AuditLog
	if err := s.GetDB(ctx).First(&log, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (s *AuditStore) List(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := s.GetDB(ctx).Model(&models.AuditLog{})

	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query = query.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *AuditStore) ListByResource(ctx context.Context, resourceType, resourceID string, limit int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	query := s.GetDB(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *AuditStore) CleanupOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := s.GetDB(ctx).Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}

