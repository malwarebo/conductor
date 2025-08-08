package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/malwarebo/gopay/models"
)

type MultiProviderSelector struct {
	Providers []PaymentProvider
	mu        sync.RWMutex

	paymentProviderMap      map[string]PaymentProvider
	subscriptionProviderMap map[string]PaymentProvider
	disputeProviderMap      map[string]PaymentProvider

	providerPreferences map[string]int
}

func NewMultiProviderSelector(providers []PaymentProvider) *MultiProviderSelector {
	preferences := make(map[string]int)
	for i, provider := range providers {
		switch provider.(type) {
		case *StripeProvider:
			preferences["stripe"] = i
		case *XenditProvider:
			preferences["xendit"] = i
		}
	}

	return &MultiProviderSelector{
		Providers:               providers,
		paymentProviderMap:      make(map[string]PaymentProvider),
		subscriptionProviderMap: make(map[string]PaymentProvider),
		disputeProviderMap:      make(map[string]PaymentProvider),
		providerPreferences:     preferences,
	}
}

func getProviderFromMap(m map[string]PaymentProvider, id string) (PaymentProvider, error) {
	provider, ok := m[id]
	if !ok {
		return nil, fmt.Errorf("no provider found for id: %s", id)
	}
	return provider, nil
}

func (m *MultiProviderSelector) selectAvailableProvider(ctx context.Context, preferredProvider string) (PaymentProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if preferredProvider != "" {
		if idx, ok := m.providerPreferences[preferredProvider]; ok && idx < len(m.Providers) {
			provider := m.Providers[idx]
			if provider.IsAvailable(ctx) {
				return provider, nil
			}
		}
	}

	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no available payment provider")
}

func (m *MultiProviderSelector) selectProviderByCurrency(ctx context.Context, currency string) (PaymentProvider, error) {
	switch currency {
	case "USD", "EUR", "GBP":
		return m.selectAvailableProvider(ctx, "stripe")
	case "IDR", "SGD", "MYR", "PHP", "THB", "VND":
		return m.selectAvailableProvider(ctx, "xendit")
	default:
		return m.selectAvailableProvider(ctx, "")
	}
}

func (m *MultiProviderSelector) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	provider, err := m.selectProviderByCurrency(ctx, req.Currency)
	if err != nil {
		return nil, err
	}

	resp, err := provider.Charge(ctx, req)
	if err == nil && resp != nil && resp.ID != "" {
		m.mu.Lock()
		m.paymentProviderMap[resp.ID] = provider
		m.mu.Unlock()
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
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}

	sub, err := provider.CreateSubscription(ctx, req)
	if err == nil && sub != nil && sub.ID != "" {
		m.mu.Lock()
		m.subscriptionProviderMap[sub.ID] = provider
		m.mu.Unlock()
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
	var allSubscriptions []*models.Subscription

	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			subscriptions, err := provider.ListSubscriptions(ctx, customerID)
			if err == nil {
				allSubscriptions = append(allSubscriptions, subscriptions...)
			}
		}
	}

	if len(allSubscriptions) == 0 {
		return nil, fmt.Errorf("no subscriptions found for customer: %s", customerID)
	}

	return allSubscriptions, nil
}

func (m *MultiProviderSelector) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}
	return provider.CreatePlan(ctx, plan)
}

func (m *MultiProviderSelector) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}
	return provider.UpdatePlan(ctx, planID, plan)
}

func (m *MultiProviderSelector) DeletePlan(ctx context.Context, planID string) error {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return err
	}
	return provider.DeletePlan(ctx, planID)
}

func (m *MultiProviderSelector) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}
	return provider.GetPlan(ctx, planID)
}

func (m *MultiProviderSelector) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}
	return provider.ListPlans(ctx)
}

func (m *MultiProviderSelector) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}

	dispute, err := provider.CreateDispute(ctx, req)
	if err == nil && dispute != nil && dispute.ID != "" {
		m.mu.Lock()
		m.disputeProviderMap[dispute.ID] = provider
		m.mu.Unlock()
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
	var allDisputes []*models.Dispute

	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			disputes, err := provider.ListDisputes(ctx, customerID)
			if err == nil {
				allDisputes = append(allDisputes, disputes...)
			}
		}
	}

	if len(allDisputes) == 0 {
		return nil, fmt.Errorf("no disputes found for customer: %s", customerID)
	}

	return allDisputes, nil
}

func (m *MultiProviderSelector) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
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

func (m *MultiProviderSelector) GetProviderStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_providers"] = len(m.Providers)
	stats["payment_mappings"] = len(m.paymentProviderMap)
	stats["subscription_mappings"] = len(m.subscriptionProviderMap)
	stats["dispute_mappings"] = len(m.disputeProviderMap)

	providerStats := make(map[string]bool)
	for i, provider := range m.Providers {
		providerName := fmt.Sprintf("provider_%d", i)
		switch provider.(type) {
		case *StripeProvider:
			providerName = "stripe"
		case *XenditProvider:
			providerName = "xendit"
		}
		providerStats[providerName] = provider.IsAvailable(context.Background())
	}
	stats["provider_availability"] = providerStats

	return stats
}
