package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
)

type MultiProviderSelector struct {
	Providers []PaymentProvider
	mu        sync.RWMutex

	paymentProviderMap      map[string]PaymentProvider
	subscriptionProviderMap map[string]PaymentProvider
	disputeProviderMap      map[string]PaymentProvider

	providerPreferences map[string]int
	mappingStore        *stores.ProviderMappingStore
}

func CreateMultiProviderSelector(providers []PaymentProvider, mappingStore *stores.ProviderMappingStore) *MultiProviderSelector {
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
		mappingStore:            mappingStore,
	}
}

func (m *MultiProviderSelector) Name() string {
	return "multi_provider"
}

func (m *MultiProviderSelector) Capabilities() ProviderCapabilities {
	caps := ProviderCapabilities{
		SupportedCurrencies:     []string{},
		SupportedPaymentMethods: []models.PaymentMethodType{},
	}

	for _, provider := range m.Providers {
		providerCaps := provider.Capabilities()
		caps.SupportsInvoices = caps.SupportsInvoices || providerCaps.SupportsInvoices
		caps.SupportsPayouts = caps.SupportsPayouts || providerCaps.SupportsPayouts
		caps.SupportsPaymentSessions = caps.SupportsPaymentSessions || providerCaps.SupportsPaymentSessions
		caps.Supports3DS = caps.Supports3DS || providerCaps.Supports3DS
		caps.SupportsManualCapture = caps.SupportsManualCapture || providerCaps.SupportsManualCapture
		caps.SupportsBalance = caps.SupportsBalance || providerCaps.SupportsBalance
		caps.SupportedCurrencies = append(caps.SupportedCurrencies, providerCaps.SupportedCurrencies...)
		caps.SupportedPaymentMethods = append(caps.SupportedPaymentMethods, providerCaps.SupportedPaymentMethods...)
	}

	return caps
}

func (m *MultiProviderSelector) getProviderFromDB(ctx context.Context, entityID, entityType string) (PaymentProvider, error) {
	mapping, err := m.mappingStore.GetByEntity(ctx, entityID, entityType)
	if err != nil {
		return nil, fmt.Errorf("no provider mapping found for %s: %s", entityType, entityID)
	}

	if idx, ok := m.providerPreferences[mapping.ProviderName]; ok && idx < len(m.Providers) {
		return m.Providers[idx], nil
	}

	return nil, fmt.Errorf("provider %s not available", mapping.ProviderName)
}

func (m *MultiProviderSelector) saveProviderMapping(ctx context.Context, entityID, entityType, providerName, providerEntityID string) error {
	mapping := &models.ProviderMapping{
		EntityID:         entityID,
		EntityType:       entityType,
		ProviderName:     providerName,
		ProviderEntityID: providerEntityID,
	}
	return m.mappingStore.Create(ctx, mapping)
}

func (m *MultiProviderSelector) getProviderName(provider PaymentProvider) string {
	switch provider.(type) {
	case *StripeProvider:
		return "stripe"
	case *XenditProvider:
		return "xendit"
	default:
		return "unknown"
	}
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
		
		providerName := m.getProviderName(provider)
		m.saveProviderMapping(ctx, resp.ID, "payment", providerName, resp.ProviderChargeID)
	}
	return resp, err
}

func (m *MultiProviderSelector) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	m.mu.RLock()
	provider, ok := m.paymentProviderMap[req.PaymentID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, req.PaymentID, "payment")
		if err != nil {
			return nil, err
		}
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
		
		providerName := m.getProviderName(provider)
		m.saveProviderMapping(ctx, sub.ID, "subscription", providerName, sub.ID)
	}
	return sub, err
}

func (m *MultiProviderSelector) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	m.mu.RLock()
	provider, ok := m.subscriptionProviderMap[subscriptionID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, subscriptionID, "subscription")
		if err != nil {
			return nil, err
		}
	}
	
	return provider.UpdateSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	m.mu.RLock()
	provider, ok := m.subscriptionProviderMap[subscriptionID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, subscriptionID, "subscription")
		if err != nil {
			return nil, err
		}
	}
	
	return provider.CancelSubscription(ctx, subscriptionID, req)
}

func (m *MultiProviderSelector) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	m.mu.RLock()
	provider, ok := m.subscriptionProviderMap[subscriptionID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, subscriptionID, "subscription")
		if err != nil {
			return nil, err
		}
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
		
		providerName := m.getProviderName(provider)
		m.saveProviderMapping(ctx, dispute.ID, "dispute", providerName, dispute.ID)
	}
	return dispute, err
}

func (m *MultiProviderSelector) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	m.mu.RLock()
	provider, ok := m.disputeProviderMap[disputeID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, disputeID, "dispute")
		if err != nil {
			return nil, err
		}
	}
	
	return provider.UpdateDispute(ctx, disputeID, req)
}

func (m *MultiProviderSelector) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	m.mu.RLock()
	provider, ok := m.disputeProviderMap[disputeID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, disputeID, "dispute")
		if err != nil {
			return nil, err
		}
	}
	
	return provider.SubmitDisputeEvidence(ctx, disputeID, req)
}

func (m *MultiProviderSelector) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	m.mu.RLock()
	provider, ok := m.disputeProviderMap[disputeID]
	m.mu.RUnlock()
	
	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, disputeID, "dispute")
		if err != nil {
			return nil, err
		}
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

func (m *MultiProviderSelector) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return "", err
	}
	return provider.CreateCustomer(ctx, req)
}

func (m *MultiProviderSelector) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return err
	}
	return provider.UpdateCustomer(ctx, customerID, req)
}

func (m *MultiProviderSelector) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return nil, err
	}
	return provider.GetCustomer(ctx, customerID)
}

func (m *MultiProviderSelector) DeleteCustomer(ctx context.Context, customerID string) error {
	provider, err := m.selectAvailableProvider(ctx, "stripe")
	if err != nil {
		return err
	}
	return provider.DeleteCustomer(ctx, customerID)
}

func (m *MultiProviderSelector) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	providerName := req.Provider
	if providerName == "" {
		providerName = "stripe"
	}

	provider, err := m.selectAvailableProvider(ctx, providerName)
	if err != nil {
		return nil, err
	}

	if pmProvider, ok := provider.(PaymentMethodProvider); ok {
		return pmProvider.CreatePaymentMethod(ctx, req)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if pmProvider, ok := provider.(PaymentMethodProvider); ok {
				pm, err := pmProvider.GetPaymentMethod(ctx, paymentMethodID)
				if err == nil {
					return pm, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("payment method not found")
}

func (m *MultiProviderSelector) ListPaymentMethods(ctx context.Context, customerID string, pmType *models.PaymentMethodType) ([]*models.PaymentMethod, error) {
	var allMethods []*models.PaymentMethod
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if pmProvider, ok := provider.(PaymentMethodProvider); ok {
				methods, err := pmProvider.ListPaymentMethods(ctx, customerID, pmType)
				if err == nil {
					allMethods = append(allMethods, methods...)
				}
			}
		}
	}
	return allMethods, nil
}

func (m *MultiProviderSelector) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if pmProvider, ok := provider.(PaymentMethodProvider); ok {
				err := pmProvider.AttachPaymentMethod(ctx, paymentMethodID, customerID)
				if err == nil {
					return nil
				}
			}
		}
	}
	return ErrNotSupported
}

func (m *MultiProviderSelector) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if pmProvider, ok := provider.(PaymentMethodProvider); ok {
				err := pmProvider.DetachPaymentMethod(ctx, paymentMethodID)
				if err == nil {
					return nil
				}
			}
		}
	}
	return ErrNotSupported
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
