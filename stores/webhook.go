package stores

import (
	"context"
	"time"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type WebhookStore struct {
	BaseStore
}

func CreateWebhookStore(db *gorm.DB) *WebhookStore {
	return &WebhookStore{BaseStore: BaseStore{db: db}}
}

func (s *WebhookStore) Create(ctx context.Context, event *models.WebhookEvent) error {
	return s.GetDB(ctx).Create(event).Error
}

func (s *WebhookStore) Update(ctx context.Context, event *models.WebhookEvent) error {
	return s.GetDB(ctx).Save(event).Error
}

func (s *WebhookStore) GetByID(ctx context.Context, id string) (*models.WebhookEvent, error) {
	var event models.WebhookEvent
	if err := s.GetDB(ctx).First(&event, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *WebhookStore) GetByEventID(ctx context.Context, provider, eventID string) (*models.WebhookEvent, error) {
	var event models.WebhookEvent
	if err := s.GetDB(ctx).Where("provider = ? AND event_id = ?", provider, eventID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *WebhookStore) GetPendingEvents(ctx context.Context, limit int) ([]*models.WebhookEvent, error) {
	var events []*models.WebhookEvent
	now := time.Now()

	err := s.GetDB(ctx).
		Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?) AND attempts < max_attempts",
			[]string{string(models.WebhookEventStatusPending), string(models.WebhookEventStatusRetrying)}, now).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *WebhookStore) MarkProcessing(ctx context.Context, id string) error {
	now := time.Now()
	return s.GetDB(ctx).Model(&models.WebhookEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         models.WebhookEventStatusProcessing,
			"last_attempt_at": now,
			"attempts":       gorm.Expr("attempts + 1"),
		}).Error
}

func (s *WebhookStore) MarkCompleted(ctx context.Context, id string) error {
	now := time.Now()
	return s.GetDB(ctx).Model(&models.WebhookEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.WebhookEventStatusCompleted,
			"processed_at": now,
		}).Error
}

func (s *WebhookStore) MarkFailed(ctx context.Context, id string, errMsg string, scheduleRetry bool) error {
	updates := map[string]interface{}{
		"error_message": errMsg,
	}

	if scheduleRetry {
		updates["status"] = models.WebhookEventStatusRetrying
		updates["next_attempt_at"] = s.calculateNextAttempt(ctx, id)
	} else {
		updates["status"] = models.WebhookEventStatusFailed
	}

	return s.GetDB(ctx).Model(&models.WebhookEvent{}).Where("id = ?", id).Updates(updates).Error
}

func (s *WebhookStore) calculateNextAttempt(ctx context.Context, id string) time.Time {
	var event models.WebhookEvent
	s.GetDB(ctx).Select("attempts").First(&event, "id = ?", id)

	delays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		24 * time.Hour,
	}

	idx := event.Attempts
	if idx >= len(delays) {
		idx = len(delays) - 1
	}

	return time.Now().Add(delays[idx])
}

func (s *WebhookStore) ListByProvider(ctx context.Context, provider string, status *models.WebhookEventStatus, limit, offset int) ([]*models.WebhookEvent, error) {
	var events []*models.WebhookEvent
	query := s.GetDB(ctx).Where("provider = ?", provider)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (s *WebhookStore) CleanupOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := s.GetDB(ctx).
		Where("created_at < ? AND status = ?", cutoff, models.WebhookEventStatusCompleted).
		Delete(&models.WebhookEvent{})
	return result.RowsAffected, result.Error
}

