package stores

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type TenantStore struct {
	BaseStore
}

func CreateTenantStore(db *gorm.DB) *TenantStore {
	return &TenantStore{BaseStore: BaseStore{db: db}}
}

func (s *TenantStore) Create(ctx context.Context, tenant *models.Tenant) error {
	if tenant.APIKey == "" {
		tenant.APIKey = s.generateAPIKey()
	}
	if tenant.APISecret == "" {
		tenant.APISecret = s.generateAPISecret()
	}
	return s.GetDB(ctx).Create(tenant).Error
}

func (s *TenantStore) Update(ctx context.Context, tenant *models.Tenant) error {
	return s.GetDB(ctx).Save(tenant).Error
}

func (s *TenantStore) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.GetDB(ctx).First(&tenant, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantStore) GetByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.GetDB(ctx).Where("api_key = ? AND is_active = true", apiKey).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantStore) List(ctx context.Context, activeOnly bool, limit, offset int) ([]*models.Tenant, int64, error) {
	var tenants []*models.Tenant
	var total int64

	query := s.GetDB(ctx).Model(&models.Tenant{})
	if activeOnly {
		query = query.Where("is_active = true")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (s *TenantStore) Delete(ctx context.Context, id string) error {
	return s.GetDB(ctx).Delete(&models.Tenant{}, "id = ?", id).Error
}

func (s *TenantStore) Deactivate(ctx context.Context, id string) error {
	return s.GetDB(ctx).Model(&models.Tenant{}).Where("id = ?", id).Update("is_active", false).Error
}

func (s *TenantStore) RegenerateAPISecret(ctx context.Context, id string) (string, error) {
	newSecret := s.generateAPISecret()
	err := s.GetDB(ctx).Model(&models.Tenant{}).Where("id = ?", id).Update("api_secret", newSecret).Error
	if err != nil {
		return "", err
	}
	return newSecret, nil
}

func (s *TenantStore) ValidateCredentials(ctx context.Context, apiKey, apiSecret string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := s.GetDB(ctx).Where("api_key = ? AND api_secret = ? AND is_active = true", apiKey, apiSecret).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *TenantStore) generateAPIKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "pk_" + hex.EncodeToString(bytes)
}

func (s *TenantStore) generateAPISecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "sk_" + hex.EncodeToString(bytes)
}

