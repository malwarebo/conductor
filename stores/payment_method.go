package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type PaymentMethodStore struct {
	db *gorm.DB
}

func CreatePaymentMethodStore(db *gorm.DB) *PaymentMethodStore {
	return &PaymentMethodStore{db: db}
}

func (s *PaymentMethodStore) Create(ctx context.Context, pm *models.PaymentMethod) error {
	return s.getDB(ctx).Create(pm).Error
}

func (s *PaymentMethodStore) Update(ctx context.Context, pm *models.PaymentMethod) error {
	return s.getDB(ctx).Save(pm).Error
}

func (s *PaymentMethodStore) GetByID(ctx context.Context, id string) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	if err := s.getDB(ctx).First(&pm, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &pm, nil
}

func (s *PaymentMethodStore) ListByCustomer(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	var pms []*models.PaymentMethod
	if err := s.getDB(ctx).Where("customer_id = ?", customerID).Find(&pms).Error; err != nil {
		return nil, err
	}
	return pms, nil
}

func (s *PaymentMethodStore) Delete(ctx context.Context, id string) error {
	return s.getDB(ctx).Delete(&models.PaymentMethod{}, "id = ?", id).Error
}

func (s *PaymentMethodStore) SetDefault(ctx context.Context, customerID, id string) error {
	return s.getDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PaymentMethod{}).
			Where("customer_id = ?", customerID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&models.PaymentMethod{}).
			Where("id = ?", id).
			Update("is_default", true).Error
	})
}

func (s *PaymentMethodStore) GetDefault(ctx context.Context, customerID string) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	if err := s.getDB(ctx).Where("customer_id = ? AND is_default = ?", customerID, true).First(&pm).Error; err != nil {
		return nil, err
	}
	return &pm, nil
}

func (s *PaymentMethodStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return s.db.WithContext(ctx)
}

