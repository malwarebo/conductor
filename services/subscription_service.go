package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
	ErrNoAvailableProvider = errors.New("no available payment provider")
)

type SubscriptionService struct {
	providers    []providers.PaymentProvider
	planRepo     *repositories.PlanRepository
	subRepo      *repositories.SubscriptionRepository
	mu           sync.RWMutex
}

func NewSubscriptionService(planRepo *repositories.PlanRepository, subRepo *repositories.SubscriptionRepository, providers ...providers.PaymentProvider) *SubscriptionService {
	return &SubscriptionService{
		providers: providers,
		planRepo:  planRepo,
		subRepo:   subRepo,
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

	// Create plan in payment provider
	providerPlan, err := provider.CreatePlan(ctx, plan)
	if err != nil {
		return nil, err
	}

	// Store plan in database
	if err := s.planRepo.Create(ctx, providerPlan); err != nil {
		return nil, err
	}

	return providerPlan, nil
}

func (s *SubscriptionService) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Update plan in payment provider
	updatedPlan, err := provider.UpdatePlan(ctx, planID, plan)
	if err != nil {
		return nil, err
	}

	// Update plan in database
	if err := s.planRepo.Update(ctx, updatedPlan); err != nil {
		return nil, err
	}

	return updatedPlan, nil
}

func (s *SubscriptionService) DeletePlan(ctx context.Context, planID string) error {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return ErrNoAvailableProvider
	}

	// Delete plan from payment provider
	if err := provider.DeletePlan(ctx, planID); err != nil {
		return err
	}

	// Delete plan from database
	return s.planRepo.Delete(ctx, planID)
}

func (s *SubscriptionService) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	// Get plan from database
	return s.planRepo.GetByID(ctx, planID)
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	// Get plans from database
	return s.planRepo.List(ctx)
}

// Subscription Management
func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate plan exists
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	// If trial days not specified, use plan's trial days
	if req.TrialDays == nil {
		trialDays := plan.TrialDays
		req.TrialDays = &trialDays
	}

	// Create subscription in payment provider
	subscription, err := provider.CreateSubscription(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store subscription in database
	if err := s.subRepo.Create(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// If changing plans, validate new plan exists
	if req.PlanID != nil {
		if _, err := s.planRepo.GetByID(ctx, *req.PlanID); err != nil {
			return nil, ErrPlanNotFound
		}
	}

	// Update subscription in payment provider
	subscription, err := provider.UpdateSubscription(ctx, subscriptionID, req)
	if err != nil {
		return nil, err
	}

	// Update subscription in database
	if err := s.subRepo.Update(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Cancel subscription in payment provider
	subscription, err := provider.CancelSubscription(ctx, subscriptionID, req)
	if err != nil {
		return nil, err
	}

	// Update subscription in database
	now := time.Now()
	subscription.CanceledAt = &now
	if err := s.subRepo.Update(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	// Get subscription from database
	return s.subRepo.GetByID(ctx, subscriptionID)
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	// Get subscriptions from database
	return s.subRepo.ListByCustomer(ctx, customerID)
}
