package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	BaseStore
}

func CreateSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{BaseStore: BaseStore{db: db}}
}

func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	return r.GetDB(ctx).Create(subscription).Error
}

func (r *SubscriptionRepository) Update(ctx context.Context, subscription *models.Subscription) error {
	return r.GetDB(ctx).Save(subscription).Error
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id string) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.GetDB(ctx).Preload("Plan").First(&subscription, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	if err := r.GetDB(ctx).Preload("Plan").Where("customer_id = ?", customerID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) ListActive(ctx context.Context) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	if err := r.GetDB(ctx).Preload("Plan").Where("status = ?", "active").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
	return r.GetDB(ctx).Delete(&models.Subscription{}, "id = ?", id).Error
}
