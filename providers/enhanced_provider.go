package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/utils"
)

type EnhancedProvider struct {
	provider       PaymentProvider
	circuitBreaker *utils.CircuitBreaker
	healthChecker  *utils.HealthChecker
	name           string
}

func NewEnhancedProvider(provider PaymentProvider, name string) *EnhancedProvider {
	ep := &EnhancedProvider{
		provider:       provider,
		name:           name,
		circuitBreaker: utils.NewCircuitBreaker(5, 30*time.Second),
	}

	ep.healthChecker = utils.NewHealthChecker(ep.healthCheck, 30*time.Second, 5*time.Second)
	ep.healthChecker.Start()

	return ep
}

func (ep *EnhancedProvider) healthCheck(ctx context.Context) error {
	if ep.provider.IsAvailable(ctx) {
		return nil
	}
	return fmt.Errorf("provider %s is not available", ep.name)
}

func (ep *EnhancedProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	var resp *models.ChargeResponse
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.Charge(ctx, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	var resp *models.RefundResponse
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.Refund(ctx, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreateSubscription(ctx, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdateSubscription(ctx, subscriptionID, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CancelSubscription(ctx, subscriptionID, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetSubscription(ctx, subscriptionID)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	var resp []*models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListSubscriptions(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreatePlan(ctx, plan)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdatePlan(ctx, planID, plan)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) DeletePlan(ctx context.Context, planID string) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.DeletePlan(ctx, planID)
	})
}

func (ep *EnhancedProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetPlan(ctx, planID)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	var resp []*models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListPlans(ctx)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreateDispute(ctx, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdateDispute(ctx, disputeID, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	var resp *models.Evidence
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.SubmitDisputeEvidence(ctx, disputeID, req)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetDispute(ctx, disputeID)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	var resp []*models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListDisputes(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	var resp *models.DisputeStats
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetDisputeStats(ctx)
		return err
	})
	return resp, err
}

func (ep *EnhancedProvider) IsAvailable(ctx context.Context) bool {
	return ep.healthChecker.IsHealthy() && ep.circuitBreaker.GetState() != utils.StateOpen
}

func (ep *EnhancedProvider) GetHealthStatus() utils.HealthStatus {
	return ep.healthChecker.GetStatus()
}

func (ep *EnhancedProvider) GetCircuitBreakerState() utils.CircuitState {
	return ep.circuitBreaker.GetState()
}

func (ep *EnhancedProvider) Stop() {
	ep.healthChecker.Stop()
}
