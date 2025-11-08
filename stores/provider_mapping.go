package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type ProviderMappingStore struct {
	db *gorm.DB
}

func CreateProviderMappingStore(db *gorm.DB) *ProviderMappingStore {
	return &ProviderMappingStore{db: db}
}

func (s *ProviderMappingStore) Create(ctx context.Context, mapping *models.ProviderMapping) error {
	return s.getDB(ctx).Create(mapping).Error
}

func (s *ProviderMappingStore) Update(ctx context.Context, mapping *models.ProviderMapping) error {
	return s.getDB(ctx).Save(mapping).Error
}

func (s *ProviderMappingStore) GetByEntity(ctx context.Context, entityID, entityType string) (*models.ProviderMapping, error) {
	var mapping models.ProviderMapping
	if err := s.getDB(ctx).Where("entity_id = ? AND entity_type = ?", entityID, entityType).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (s *ProviderMappingStore) Delete(ctx context.Context, entityID, entityType string) error {
	return s.getDB(ctx).Where("entity_id = ? AND entity_type = ?", entityID, entityType).Delete(&models.ProviderMapping{}).Error
}

func (s *ProviderMappingStore) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return s.db.WithContext(ctx)
}

