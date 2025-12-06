package services

import (
	"context"
	"errors"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
)

var (
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrTenantInactive   = errors.New("tenant is inactive")
	ErrInvalidAPIKey    = errors.New("invalid api key")
)

type TenantService struct {
	store *stores.TenantStore
}

func CreateTenantService(store *stores.TenantStore) *TenantService {
	return &TenantService{store: store}
}

func (s *TenantService) Create(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	tenant := &models.Tenant{
		Name:       req.Name,
		WebhookURL: req.WebhookURL,
		IsActive:   true,
		Settings:   req.Settings,
		Metadata:   req.Metadata,
	}

	if err := s.store.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) Update(ctx context.Context, id string, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	tenant, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.WebhookURL != "" {
		tenant.WebhookURL = req.WebhookURL
	}
	if req.WebhookSecret != "" {
		tenant.WebhookSecret = req.WebhookSecret
	}
	if req.IsActive != nil {
		tenant.IsActive = *req.IsActive
	}
	if req.Settings != nil {
		tenant.Settings = req.Settings
	}
	if req.Metadata != nil {
		tenant.Metadata = req.Metadata
	}

	if err := s.store.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	tenant, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

func (s *TenantService) GetByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	tenant, err := s.store.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	if !tenant.IsActive {
		return nil, ErrTenantInactive
	}
	return tenant, nil
}

func (s *TenantService) List(ctx context.Context, activeOnly bool, limit, offset int) ([]*models.Tenant, int64, error) {
	return s.store.List(ctx, activeOnly, limit, offset)
}

func (s *TenantService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *TenantService) Deactivate(ctx context.Context, id string) error {
	return s.store.Deactivate(ctx, id)
}

func (s *TenantService) RegenerateAPISecret(ctx context.Context, id string) (string, error) {
	return s.store.RegenerateAPISecret(ctx, id)
}

func (s *TenantService) ValidateCredentials(ctx context.Context, apiKey, apiSecret string) (*models.Tenant, error) {
	tenant, err := s.store.ValidateCredentials(ctx, apiKey, apiSecret)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	if !tenant.IsActive {
		return nil, ErrTenantInactive
	}
	return tenant, nil
}

func (s *TenantService) GetSettings(ctx context.Context, id string) (*models.TenantSettings, error) {
	tenant, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	settings := &models.TenantSettings{
		DefaultProvider:      "stripe",
		EnabledProviders:     []string{"stripe", "xendit"},
		Enable3DS:            true,
		DefaultCaptureMethod: "automatic",
		WebhookRetryCount:    5,
	}

	if tenant.Settings != nil {
		if dp, ok := tenant.Settings["default_provider"].(string); ok {
			settings.DefaultProvider = dp
		}
		if ep, ok := tenant.Settings["enabled_providers"].([]interface{}); ok {
			settings.EnabledProviders = make([]string, len(ep))
			for i, p := range ep {
				settings.EnabledProviders[i] = p.(string)
			}
		}
		if e3ds, ok := tenant.Settings["enable_3ds"].(bool); ok {
			settings.Enable3DS = e3ds
		}
		if dcm, ok := tenant.Settings["default_capture_method"].(string); ok {
			settings.DefaultCaptureMethod = dcm
		}
		if wrc, ok := tenant.Settings["webhook_retry_count"].(float64); ok {
			settings.WebhookRetryCount = int(wrc)
		}
	}

	return settings, nil
}

