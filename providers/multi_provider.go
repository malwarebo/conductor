package providers

import (
	"context"
	"fmt"

	"github.com/malwarebo/gopay/models"
)

type MultiProviderSelector struct {
	Providers []PaymentProvider

	// In-memory maps for tracking which provider handled each entity
	paymentProviderMap      map[string]PaymentProvider // paymentID -> provider
	subscriptionProviderMap map[string]PaymentProvider // subscriptionID -> provider
	disputeProviderMap      map[string]PaymentProvider // disputeID -> provider
}

func NewMultiProviderSelector(providers []PaymentProvider) *MultiProviderSelector {
	return &MultiProviderSelector{
		Providers:               providers,
		paymentProviderMap:      make(map[string]PaymentProvider),
		subscriptionProviderMap: make(map[string]PaymentProvider),
		disputeProviderMap:      make(map[string]PaymentProvider),
	}
}

func getProviderFromMap(m map[string]PaymentProvider, id string) (PaymentProvider, error) {
	provider, ok := m[id]
	if !ok {
		return nil, fmt.Errorf("no provider found for id: %s", id)
	}
	return provider, nil
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
	resp, err := provider.Charge(ctx, req)
	if err == nil && resp != nil && resp.ID != "" {
		m.paymentProviderMap[resp.ID] = provider
	}
	return resp, err
}

func (m *MultiProviderSelector) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	provider, err := getProviderFromMap(m.paymentProviderMap, req.PaymentID)
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
	sub, err := provider.CreateSubscription(ctx, req)
	if err == nil && sub != nil && sub.ID != "" {
		m.subscriptionProviderMap[sub.ID] = provider
	}
	return sub, err
}

func (m *MultiProviderSelector) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	provider, err := getProviderFromMap(m.subscriptionProviderMap, subscriptionID)
	if err != nil {
		return nil, err
	}
	return provider.UpdateSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	provider, err := getProviderFromMap(m.subscriptionProviderMap, subscriptionID)
	if err != nil {
		return nil, err
	}
	return provider.CancelSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	provider, err := getProviderFromMap(m.subscriptionProviderMap, subscriptionID)
	if err != nil {
		return nil, err
	}
	return provider.GetSubscription(ctx, subscriptionID)
}

func (m *MultiProviderSelector) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	// This method may need to aggregate from all providers, but for now, use the first available
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
	// Plan-provider mapping not tracked; fallback to first available
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
	dispute, err := provider.CreateDispute(ctx, req)
	if err == nil && dispute != nil && dispute.ID != "" {
		m.disputeProviderMap[dispute.ID] = provider
	}
	return dispute, err
}

func (m *MultiProviderSelector) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	provider, err := getProviderFromMap(m.disputeProviderMap, disputeID)
	if err != nil {
		return nil, err
	}
	return provider.UpdateDispute(ctx, disputeID, req)
}

func (m *MultiProviderSelector) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	provider, err := getProviderFromMap(m.disputeProviderMap, disputeID)
	if err != nil {
		return nil, err
	}
	return provider.SubmitDisputeEvidence(ctx, disputeID, req)
}

func (m *MultiProviderSelector) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	provider, err := getProviderFromMap(m.disputeProviderMap, disputeID)
	if err != nil {
		return nil, err
	}
	return provider.GetDispute(ctx, disputeID)
}

func (m *MultiProviderSelector) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	// This method may need to aggregate from all providers, but for now, use the first available
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
