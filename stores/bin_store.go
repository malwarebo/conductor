package stores

import (
	"context"
	"sync"
	"time"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type BINStore struct {
	db    *gorm.DB
	cache sync.Map
}

func NewBINStore(db *gorm.DB) *BINStore {
	return &BINStore{db: db}
}

func (s *BINStore) Migrate() error {
	return s.db.AutoMigrate(&models.BINData{}, &models.BINProviderStats{})
}

func (s *BINStore) Get(ctx context.Context, bin string) (*models.BINData, error) {
	if cached, ok := s.cache.Load(bin); ok {
		return cached.(*models.BINData), nil
	}

	var data models.BINData
	if err := s.db.WithContext(ctx).First(&data, "bin = ?", bin).Error; err != nil {
		return nil, err
	}

	stats, _ := s.GetProviderStats(ctx, bin)
	data.ProviderStats = stats

	s.cache.Store(bin, &data)
	return &data, nil
}

func (s *BINStore) GetByPrefix(ctx context.Context, prefix string) (*models.BINData, error) {
	for i := len(prefix); i >= 6; i-- {
		if data, err := s.Get(ctx, prefix[:i]); err == nil {
			return data, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *BINStore) Upsert(ctx context.Context, data *models.BINData) error {
	data.UpdatedAt = time.Now()
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}

	err := s.db.WithContext(ctx).Save(data).Error
	if err == nil {
		s.cache.Delete(data.BIN)
	}
	return err
}

func (s *BINStore) GetProviderStats(ctx context.Context, bin string) (map[string]*models.BINProviderStats, error) {
	var stats []models.BINProviderStats
	if err := s.db.WithContext(ctx).Where("bin = ?", bin).Find(&stats).Error; err != nil {
		return nil, err
	}

	result := make(map[string]*models.BINProviderStats)
	for i := range stats {
		result[stats[i].ProviderName] = &stats[i]
	}
	return result, nil
}

func (s *BINStore) UpdateProviderStats(ctx context.Context, bin, provider string, success bool, responseTime int64) error {
	var stats models.BINProviderStats
	err := s.db.WithContext(ctx).FirstOrCreate(&stats, models.BINProviderStats{
		BIN:          bin,
		ProviderName: provider,
	}).Error
	if err != nil {
		return err
	}

	stats.TotalAttempts++
	if success {
		stats.SuccessCount++
	}
	stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalAttempts)
	stats.AvgResponseTime = ((stats.AvgResponseTime * (stats.TotalAttempts - 1)) + responseTime) / stats.TotalAttempts
	stats.LastUpdated = time.Now()

	err = s.db.WithContext(ctx).Save(&stats).Error
	if err == nil {
		s.cache.Delete(bin)
	}
	return err
}

func (s *BINStore) GetBestProvider(ctx context.Context, bin string, minAttempts int64) (string, float64, error) {
	stats, err := s.GetProviderStats(ctx, bin)
	if err != nil {
		return "", 0, err
	}

	var bestProvider string
	var bestRate float64

	for provider, s := range stats {
		if s.TotalAttempts >= minAttempts && s.SuccessRate > bestRate {
			bestProvider = provider
			bestRate = s.SuccessRate
		}
	}

	return bestProvider, bestRate, nil
}

type MerchantConfigStore struct {
	db    *gorm.DB
	cache sync.Map
}

func NewMerchantConfigStore(db *gorm.DB) *MerchantConfigStore {
	return &MerchantConfigStore{db: db}
}

func (s *MerchantConfigStore) Migrate() error {
	return s.db.AutoMigrate(&models.MerchantRoutingConfig{})
}

func (s *MerchantConfigStore) Get(ctx context.Context, merchantID string) (*models.MerchantRoutingConfig, error) {
	if cached, ok := s.cache.Load(merchantID); ok {
		return cached.(*models.MerchantRoutingConfig), nil
	}

	var config models.MerchantRoutingConfig
	if err := s.db.WithContext(ctx).First(&config, "merchant_id = ?", merchantID).Error; err != nil {
		return nil, err
	}

	s.cache.Store(merchantID, &config)
	return &config, nil
}

func (s *MerchantConfigStore) GetOrDefault(ctx context.Context, merchantID string) *models.MerchantRoutingConfig {
	config, err := s.Get(ctx, merchantID)
	if err != nil {
		return &models.MerchantRoutingConfig{
			MerchantID:         merchantID,
			EnableSmartRouting: true,
			EnableRetry:        true,
			MaxRetryAttempts:   2,
			MinSuccessRate:     0.9,
		}
	}
	return config
}

func (s *MerchantConfigStore) Upsert(ctx context.Context, config *models.MerchantRoutingConfig) error {
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	err := s.db.WithContext(ctx).Save(config).Error
	if err == nil {
		s.cache.Delete(config.MerchantID)
	}
	return err
}

func (s *MerchantConfigStore) Delete(ctx context.Context, merchantID string) error {
	err := s.db.WithContext(ctx).Delete(&models.MerchantRoutingConfig{}, "merchant_id = ?", merchantID).Error
	if err == nil {
		s.cache.Delete(merchantID)
	}
	return err
}

type RoutingRuleStore struct {
	db    *gorm.DB
	cache []models.RoutingRule
	mu    sync.RWMutex
}

func NewRoutingRuleStore(db *gorm.DB) *RoutingRuleStore {
	return &RoutingRuleStore{db: db}
}

func (s *RoutingRuleStore) Migrate() error {
	return s.db.AutoMigrate(&models.RoutingRule{})
}

func (s *RoutingRuleStore) GetAll(ctx context.Context) ([]models.RoutingRule, error) {
	s.mu.RLock()
	if len(s.cache) > 0 {
		s.mu.RUnlock()
		return s.cache, nil
	}
	s.mu.RUnlock()

	var rules []models.RoutingRule
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("priority DESC").Find(&rules).Error; err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache = rules
	s.mu.Unlock()

	return rules, nil
}

func (s *RoutingRuleStore) Create(ctx context.Context, rule *models.RoutingRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	err := s.db.WithContext(ctx).Create(rule).Error
	if err == nil {
		s.invalidateCache()
	}
	return err
}

func (s *RoutingRuleStore) Update(ctx context.Context, rule *models.RoutingRule) error {
	rule.UpdatedAt = time.Now()

	err := s.db.WithContext(ctx).Save(rule).Error
	if err == nil {
		s.invalidateCache()
	}
	return err
}

func (s *RoutingRuleStore) Delete(ctx context.Context, id string) error {
	err := s.db.WithContext(ctx).Delete(&models.RoutingRule{}, "id = ?", id).Error
	if err == nil {
		s.invalidateCache()
	}
	return err
}

func (s *RoutingRuleStore) invalidateCache() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}
