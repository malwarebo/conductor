package services

import (
	"context"
	"errors"
	"sync"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
)

type SubscriptionService struct {
	providers []providers.PaymentProvider
	mu        sync.RWMutex
}

func NewSubscriptionService(providers ...providers.PaymentProvider) *SubscriptionService {
	return &SubscriptionService{
		providers: providers,
	}
}

func (s *SubscriptionService) AddProvider(provider providers.PaymentProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, provider)
}

func (s *SubscriptionService) getAvailableProvider(ctx context.Context) providers.PaymentProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.providers {
		if provider.IsAvailable(ctx) {
			return provider
		}
	}
	return nil
}

// Plan Management
func (s *SubscriptionService) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.CreatePlan(ctx, plan)
}

func (s *SubscriptionService) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.UpdatePlan(ctx, planID, plan)
}

func (s *SubscriptionService) DeletePlan(ctx context.Context, planID string) error {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return ErrNoAvailableProvider
	}
	return provider.DeletePlan(ctx, planID)
}

func (s *SubscriptionService) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.GetPlan(ctx, planID)
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.ListPlans(ctx)
}

// Subscription Management
func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate plan exists
	plan, err := provider.GetPlan(ctx, req.PlanID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	// If trial days not specified, use plan's trial days
	if req.TrialDays == nil {
		trialDays := plan.TrialDays
		req.TrialDays = &trialDays
	}

	return provider.CreateSubscription(ctx, req)
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// If changing plans, validate new plan exists
	if req.PlanID != nil {
		if _, err := provider.GetPlan(ctx, *req.PlanID); err != nil {
			return nil, ErrPlanNotFound
		}
	}

	return provider.UpdateSubscription(ctx, subscriptionID, req)
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.CancelSubscription(ctx, subscriptionID, req)
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.GetSubscription(ctx, subscriptionID)
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.ListSubscriptions(ctx, customerID)
}
