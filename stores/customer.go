package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type CustomerStore struct {
	db *gorm.DB
}

func CreateCustomerStore(db *gorm.DB) *CustomerStore {
	return &CustomerStore{db: db}
}

func (s *CustomerStore) Create(ctx context.Context, customer *models.Customer) error {
	return s.getDB(ctx).Create(customer).Error
}

func (s *CustomerStore) Update(ctx context.Context, customer *models.Customer) error {
	return s.getDB(ctx).Save(customer).Error
}

func (s *CustomerStore) GetByID(ctx context.Context, id string) (*models.Customer, error) {
	var customer models.Customer
	if err := s.getDB(ctx).First(&customer, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (s *CustomerStore) GetByExternalID(ctx context.Context, externalID string) (*models.Customer, error) {
	var customer models.Customer
	if err := s.getDB(ctx).First(&customer, "external_id = ?", externalID).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (s *CustomerStore) GetByEmail(ctx context.Context, email string) (*models.Customer, error) {
	var customer models.Customer
	if err := s.getDB(ctx).First(&customer, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (s *CustomerStore) Delete(ctx context.Context, id string) error {
	return s.getDB(ctx).Delete(&models.Customer{}, "id = ?", id).Error
}

func (s *CustomerStore) List(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	var customers []*models.Customer
	query := s.getDB(ctx)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&customers).Error; err != nil {
		return nil, err
	}
	return customers, nil
}

func (s *CustomerStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return s.db.WithContext(ctx)
}

