package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type PaymentMethodStore struct {
	BaseStore
}

func CreatePaymentMethodStore(db *gorm.DB) *PaymentMethodStore {
	return &PaymentMethodStore{BaseStore: BaseStore{db: db}}
}

func (s *PaymentMethodStore) Create(ctx context.Context, pm *models.PaymentMethod) error {
	return s.GetDB(ctx).Create(pm).Error
}

func (s *PaymentMethodStore) Update(ctx context.Context, pm *models.PaymentMethod) error {
	return s.GetDB(ctx).Save(pm).Error
}

func (s *PaymentMethodStore) GetByID(ctx context.Context, id string) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	if err := s.GetDB(ctx).First(&pm, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &pm, nil
}

func (s *PaymentMethodStore) ListByCustomer(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	var pms []*models.PaymentMethod
	if err := s.GetDB(ctx).Where("customer_id = ?", customerID).Find(&pms).Error; err != nil {
		return nil, err
	}
	return pms, nil
}

func (s *PaymentMethodStore) Delete(ctx context.Context, id string) error {
	return s.GetDB(ctx).Delete(&models.PaymentMethod{}, "id = ?", id).Error
}

func (s *PaymentMethodStore) SetDefault(ctx context.Context, customerID, id string) error {
	return s.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
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
	if err := s.GetDB(ctx).Where("customer_id = ? AND is_default = ?", customerID, true).First(&pm).Error; err != nil {
		return nil, err
	}
	return &pm, nil
}

