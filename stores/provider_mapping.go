package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type ProviderMappingStore struct {
	BaseStore
}

func CreateProviderMappingStore(db *gorm.DB) *ProviderMappingStore {
	return &ProviderMappingStore{BaseStore: BaseStore{db: db}}
}

func (s *ProviderMappingStore) Create(ctx context.Context, mapping *models.ProviderMapping) error {
	return s.GetDB(ctx).Create(mapping).Error
}

func (s *ProviderMappingStore) Update(ctx context.Context, mapping *models.ProviderMapping) error {
	return s.GetDB(ctx).Save(mapping).Error
}

func (s *ProviderMappingStore) GetByEntity(ctx context.Context, entityID, entityType string) (*models.ProviderMapping, error) {
	var mapping models.ProviderMapping
	if err := s.GetDB(ctx).Where("entity_id = ? AND entity_type = ?", entityID, entityType).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (s *ProviderMappingStore) Delete(ctx context.Context, entityID, entityType string) error {
	return s.GetDB(ctx).Where("entity_id = ? AND entity_type = ?", entityID, entityType).Delete(&models.ProviderMapping{}).Error
}

