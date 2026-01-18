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
		case *RazorpayProvider:
			preferences["razorpay"] = i
		case *AirwallexProvider:
			preferences["airwallex"] = i
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
	case *RazorpayProvider:
		return "razorpay"
	case *AirwallexProvider:
		return "airwallex"
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
	case "USD", "EUR", "GBP", "CAD":
		return m.selectAvailableProvider(ctx, "stripe")
	case "IDR", "PHP", "VND", "THB", "MYR":
		return m.selectAvailableProvider(ctx, "xendit")
	case "INR":
		return m.selectAvailableProvider(ctx, "razorpay")
	case "HKD", "CNY", "AUD", "NZD", "SGD":
		return m.selectAvailableProvider(ctx, "airwallex")
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

func (m *MultiProviderSelector) AcceptDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
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

	return provider.AcceptDispute(ctx, disputeID)
}

func (m *MultiProviderSelector) ContestDispute(ctx context.Context, disputeID string, evidence map[string]interface{}) (*models.Dispute, error) {
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

	return provider.ContestDispute(ctx, disputeID, evidence)
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

func (m *MultiProviderSelector) CapturePayment(ctx context.Context, paymentID string, amount int64) error {
	m.mu.RLock()
	provider, ok := m.paymentProviderMap[paymentID]
	m.mu.RUnlock()

	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, paymentID, "payment")
		if err != nil {
			return err
		}
	}

	if capturer, ok := provider.(CaptureProvider); ok {
		return capturer.CapturePayment(ctx, paymentID, amount)
	}
	return ErrNotSupported
}

func (m *MultiProviderSelector) VoidPayment(ctx context.Context, paymentID string) error {
	m.mu.RLock()
	provider, ok := m.paymentProviderMap[paymentID]
	m.mu.RUnlock()

	if !ok {
		var err error
		provider, err = m.getProviderFromDB(ctx, paymentID, "payment")
		if err != nil {
			return err
		}
	}

	if voider, ok := provider.(VoidProvider); ok {
		return voider.VoidPayment(ctx, paymentID)
	}
	return ErrNotSupported
}

func (m *MultiProviderSelector) CreateInvoice(ctx context.Context, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	provider, err := m.selectProviderByCurrency(ctx, req.Currency)
	if err != nil {
		return nil, err
	}

	if invProvider, ok := provider.(InvoiceProvider); ok {
		inv, err := invProvider.CreateInvoice(ctx, req)
		if err == nil && inv != nil {
			providerName := m.getProviderName(provider)
			m.saveProviderMapping(ctx, inv.ProviderID, "invoice", providerName, inv.ProviderID)
		}
		return inv, err
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	provider, err := m.getProviderFromDB(ctx, invoiceID, "invoice")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if invProvider, ok := p.(InvoiceProvider); ok {
					inv, err := invProvider.GetInvoice(ctx, invoiceID)
					if err == nil {
						return inv, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("invoice not found")
	}

	if invProvider, ok := provider.(InvoiceProvider); ok {
		return invProvider.GetInvoice(ctx, invoiceID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) ListInvoices(ctx context.Context, req *models.ListInvoicesRequest) ([]*models.Invoice, error) {
	var allInvoices []*models.Invoice
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if invProvider, ok := provider.(InvoiceProvider); ok {
				invoices, err := invProvider.ListInvoices(ctx, req)
				if err == nil {
					allInvoices = append(allInvoices, invoices...)
				}
			}
		}
	}
	return allInvoices, nil
}

func (m *MultiProviderSelector) CancelInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	provider, err := m.getProviderFromDB(ctx, invoiceID, "invoice")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if invProvider, ok := p.(InvoiceProvider); ok {
					inv, err := invProvider.CancelInvoice(ctx, invoiceID)
					if err == nil {
						return inv, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("invoice not found")
	}

	if invProvider, ok := provider.(InvoiceProvider); ok {
		return invProvider.CancelInvoice(ctx, invoiceID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) CreatePayout(ctx context.Context, req *models.CreatePayoutRequest) (*models.Payout, error) {
	provider, err := m.selectProviderByCurrency(ctx, req.Currency)
	if err != nil {
		return nil, err
	}

	if payoutProvider, ok := provider.(PayoutProvider); ok {
		payout, err := payoutProvider.CreatePayout(ctx, req)
		if err == nil && payout != nil {
			providerName := m.getProviderName(provider)
			m.saveProviderMapping(ctx, payout.ProviderID, "payout", providerName, payout.ProviderID)
		}
		return payout, err
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	provider, err := m.getProviderFromDB(ctx, payoutID, "payout")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if payoutProvider, ok := p.(PayoutProvider); ok {
					payout, err := payoutProvider.GetPayout(ctx, payoutID)
					if err == nil {
						return payout, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payout not found")
	}

	if payoutProvider, ok := provider.(PayoutProvider); ok {
		return payoutProvider.GetPayout(ctx, payoutID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) ListPayouts(ctx context.Context, req *models.ListPayoutsRequest) ([]*models.Payout, error) {
	var allPayouts []*models.Payout
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if payoutProvider, ok := provider.(PayoutProvider); ok {
				payouts, err := payoutProvider.ListPayouts(ctx, req)
				if err == nil {
					allPayouts = append(allPayouts, payouts...)
				}
			}
		}
	}
	return allPayouts, nil
}

func (m *MultiProviderSelector) CancelPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	provider, err := m.getProviderFromDB(ctx, payoutID, "payout")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if payoutProvider, ok := p.(PayoutProvider); ok {
					payout, err := payoutProvider.CancelPayout(ctx, payoutID)
					if err == nil {
						return payout, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payout not found")
	}

	if payoutProvider, ok := provider.(PayoutProvider); ok {
		return payoutProvider.CancelPayout(ctx, payoutID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetPayoutChannels(ctx context.Context, currency string) ([]*models.PayoutChannel, error) {
	provider, err := m.selectProviderByCurrency(ctx, currency)
	if err != nil {
		return nil, err
	}

	if payoutProvider, ok := provider.(PayoutProvider); ok {
		return payoutProvider.GetPayoutChannels(ctx, currency)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetBalance(ctx context.Context, currency string) (*models.Balance, error) {
	provider, err := m.selectProviderByCurrency(ctx, currency)
	if err != nil {
		return nil, err
	}

	if balanceProvider, ok := provider.(BalanceProvider); ok {
		return balanceProvider.GetBalance(ctx, currency)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) CreatePaymentSession(ctx context.Context, req *models.CreatePaymentSessionRequest) (*models.PaymentSession, error) {
	provider, err := m.selectProviderByCurrency(ctx, req.Currency)
	if err != nil {
		return nil, err
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		session, err := sessionProvider.CreatePaymentSession(ctx, req)
		if err == nil && session != nil {
			providerName := m.getProviderName(provider)
			m.saveProviderMapping(ctx, session.ProviderID, "payment_session", providerName, session.ProviderID)
		}
		return session, err
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) GetPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	provider, err := m.getProviderFromDB(ctx, sessionID, "payment_session")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if sessionProvider, ok := p.(PaymentSessionProvider); ok {
					session, err := sessionProvider.GetPaymentSession(ctx, sessionID)
					if err == nil {
						return session, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payment session not found")
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		return sessionProvider.GetPaymentSession(ctx, sessionID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) UpdatePaymentSession(ctx context.Context, sessionID string, req *models.UpdatePaymentSessionRequest) (*models.PaymentSession, error) {
	provider, err := m.getProviderFromDB(ctx, sessionID, "payment_session")
	if err != nil {
		return nil, err
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		return sessionProvider.UpdatePaymentSession(ctx, sessionID, req)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) ConfirmPaymentSession(ctx context.Context, sessionID string, req *models.ConfirmPaymentSessionRequest) (*models.PaymentSession, error) {
	provider, err := m.getProviderFromDB(ctx, sessionID, "payment_session")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if sessionProvider, ok := p.(PaymentSessionProvider); ok {
					session, err := sessionProvider.ConfirmPaymentSession(ctx, sessionID, req)
					if err == nil {
						return session, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payment session not found")
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		return sessionProvider.ConfirmPaymentSession(ctx, sessionID, req)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) CapturePaymentSession(ctx context.Context, sessionID string, amount *int64) (*models.PaymentSession, error) {
	provider, err := m.getProviderFromDB(ctx, sessionID, "payment_session")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if sessionProvider, ok := p.(PaymentSessionProvider); ok {
					session, err := sessionProvider.CapturePaymentSession(ctx, sessionID, amount)
					if err == nil {
						return session, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payment session not found")
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		return sessionProvider.CapturePaymentSession(ctx, sessionID, amount)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) CancelPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	provider, err := m.getProviderFromDB(ctx, sessionID, "payment_session")
	if err != nil {
		for _, p := range m.Providers {
			if p.IsAvailable(ctx) {
				if sessionProvider, ok := p.(PaymentSessionProvider); ok {
					session, err := sessionProvider.CancelPaymentSession(ctx, sessionID)
					if err == nil {
						return session, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("payment session not found")
	}

	if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
		return sessionProvider.CancelPaymentSession(ctx, sessionID)
	}
	return nil, ErrNotSupported
}

func (m *MultiProviderSelector) ListPaymentSessions(ctx context.Context, req *models.ListPaymentSessionsRequest) ([]*models.PaymentSession, error) {
	var allSessions []*models.PaymentSession
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if sessionProvider, ok := provider.(PaymentSessionProvider); ok {
				sessions, err := sessionProvider.ListPaymentSessions(ctx, req)
				if err == nil {
					allSessions = append(allSessions, sessions...)
				}
			}
		}
	}
	return allSessions, nil
}

func (m *MultiProviderSelector) ExpirePaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	for _, provider := range m.Providers {
		if provider.IsAvailable(ctx) {
			if pmProvider, ok := provider.(PaymentMethodProvider); ok {
				pm, err := pmProvider.ExpirePaymentMethod(ctx, paymentMethodID)
				if err == nil {
					return pm, nil
				}
			}
		}
	}
	return nil, ErrNotSupported
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
		case *RazorpayProvider:
			providerName = "razorpay"
		case *AirwallexProvider:
			providerName = "airwallex"
		}
		providerStats[providerName] = provider.IsAvailable(context.Background())
	}
	stats["provider_availability"] = providerStats

	return stats
}
