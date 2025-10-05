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
	ErrPlanNotFound        = errors.New("plan not found")
	ErrNoAvailableProvider = errors.New("no available payment provider")
)

type SubscriptionService struct {
	providers []providers.PaymentProvider
	planRepo  *repositories.PlanRepository
	subRepo   *repositories.SubscriptionRepository
	mu        sync.RWMutex
}

func CreateSubscriptionService(planRepo *repositories.PlanRepository, subRepo *repositories.SubscriptionRepository, providers ...providers.PaymentProvider) *SubscriptionService {
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

func (s *SubscriptionService) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	providerPlan, err := provider.CreatePlan(ctx, plan)
	if err != nil {
		return nil, err
	}

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

	updatedPlan, err := provider.UpdatePlan(ctx, planID, plan)
	if err != nil {
		return nil, err
	}

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

	if err := provider.DeletePlan(ctx, planID); err != nil {
		return err
	}

	return s.planRepo.Delete(ctx, planID)
}

func (s *SubscriptionService) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return s.planRepo.GetByID(ctx, planID)
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return s.planRepo.List(ctx)
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	if req.TrialDays == nil {
		trialDays := plan.TrialDays
		req.TrialDays = &trialDays
	}

	subscription, err := provider.CreateSubscription(ctx, req)
	if err != nil {
		return nil, err
	}

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

	if req.PlanID != nil {
		if _, err := s.planRepo.GetByID(ctx, *req.PlanID); err != nil {
			return nil, ErrPlanNotFound
		}
	}

	subscription, err := provider.UpdateSubscription(ctx, subscriptionID, req)
	if err != nil {
		return nil, err
	}

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

	subscription, err := provider.CancelSubscription(ctx, subscriptionID, req)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	subscription.CanceledAt = &now
	if err := s.subRepo.Update(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return s.subRepo.GetByID(ctx, subscriptionID)
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return s.subRepo.ListByCustomer(ctx, customerID)
}
