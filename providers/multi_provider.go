package providers

import (
	"context"
	"fmt"

	"github.com/malwarebo/gopay/models"
)

type MultiProviderSelector struct {
	Providers []PaymentProvider
}

func (m *MultiProviderSelector) selectAvailableProvider(ctx context.Context) (PaymentProvider, error) {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no available payment provider")
}

// Implement PaymentProvider interface methods with provider selection logic

func (m *MultiProviderSelector) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.Charge(ctx, req)
}

func (m *MultiProviderSelector) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.Refund(ctx, req)
}

func (m *MultiProviderSelector) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.CreateSubscription(ctx, req)
}

func (m *MultiProviderSelector) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.UpdateSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.CancelSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetSubscription(ctx, subscriptionID)
}

func (m *MultiProviderSelector) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.ListSubscriptions(ctx, customerID)
}

func (m *MultiProviderSelector) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.CreatePlan(ctx, plan)
}

func (m *MultiProviderSelector) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.UpdatePlan(ctx, planID, plan)
}

func (m *MultiProviderSelector) DeletePlan(ctx context.Context, planID string) error {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return err
	}
	return provider.DeletePlan(ctx, planID)
}

func (m *MultiProviderSelector) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetPlan(ctx, planID)
}

func (m *MultiProviderSelector) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.ListPlans(ctx)
}

func (m *MultiProviderSelector) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.CreateDispute(ctx, req)
}

func (m *MultiProviderSelector) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.UpdateDispute(ctx, disputeID, req)
}

func (m *MultiProviderSelector) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.SubmitDisputeEvidence(ctx, disputeID, req)
}

func (m *MultiProviderSelector) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetDispute(ctx, disputeID)
}

func (m *MultiProviderSelector) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.ListDisputes(ctx, customerID)
}

func (m *MultiProviderSelector) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	provider, err := m.selectAvailableProvider(ctx)
	if err != nil {
		return nil, err
	}
	return provider.GetDisputeStats(ctx)
}

func (m *MultiProviderSelector) IsAvailable(ctx context.Context) bool {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			return true
		}
	}
	return false
}
