package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/utils"
)

// PaymentProvider defines the interface for payment gateway providers
type PaymentProvider interface {
	Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error)
	Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error)

	CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error)
	ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error)

	CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error)
	UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error)
	DeletePlan(ctx context.Context, planID string) error
	GetPlan(ctx context.Context, planID string) (*models.Plan, error)
	ListPlans(ctx context.Context) ([]*models.Plan, error)

	CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error)
	UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error)
	SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error)
	GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error)
	ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error)
	GetDisputeStats(ctx context.Context) (*models.DisputeStats, error)

	CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error)
	UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error
	GetCustomer(ctx context.Context, customerID string) (*models.Customer, error)
	DeleteCustomer(ctx context.Context, customerID string) error

	CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error)
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error)
	ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error)
	AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error

	IsAvailable(ctx context.Context) bool
}

type ChargeRequest struct {
	Amount        float64
	Currency      string
	PaymentMethod string
	Description   string
	CustomerID    string
	Metadata      map[string]string
}

type ChargeResponse struct {
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	PaymentMethod string
	ProviderName  string
	CreatedAt     int64
	Metadata      map[string]string
}

type RefundRequest struct {
	TransactionID string
	Amount        float64
	Reason        string
	Metadata      map[string]string
}

type RefundResponse struct {
	RefundID      string
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	ProviderName  string
	CreatedAt     int64
	Metadata      map[string]string
}

// ProviderWrapper wraps a PaymentProvider with circuit breaker and health checking
type ProviderWrapper struct {
	provider       PaymentProvider
	circuitBreaker *utils.CircuitBreaker
	healthChecker  *utils.HealthChecker
	name           string
}

func CreateProviderWrapper(provider PaymentProvider, name string) *ProviderWrapper {
	ep := &ProviderWrapper{
		provider:       provider,
		name:           name,
		circuitBreaker: utils.CreateCircuitBreaker(5, 30*time.Second),
	}

	ep.healthChecker = utils.CreateHealthChecker(ep.healthCheck, 30*time.Second, 5*time.Second)
	ep.healthChecker.Start()

	return ep
}

func (ep *ProviderWrapper) healthCheck(ctx context.Context) error {
	if ep.provider.IsAvailable(ctx) {
		return nil
	}
	return fmt.Errorf("provider %s is not available", ep.name)
}

func (ep *ProviderWrapper) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	var resp *models.ChargeResponse
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.Charge(ctx, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	var resp *models.RefundResponse
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.Refund(ctx, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreateSubscription(ctx, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdateSubscription(ctx, subscriptionID, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CancelSubscription(ctx, subscriptionID, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	var resp *models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetSubscription(ctx, subscriptionID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	var resp []*models.Subscription
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListSubscriptions(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreatePlan(ctx, plan)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdatePlan(ctx, planID, plan)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) DeletePlan(ctx context.Context, planID string) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.DeletePlan(ctx, planID)
	})
}

func (ep *ProviderWrapper) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	var resp *models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetPlan(ctx, planID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	var resp []*models.Plan
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListPlans(ctx)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreateDispute(ctx, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.UpdateDispute(ctx, disputeID, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	var resp *models.Evidence
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.SubmitDisputeEvidence(ctx, disputeID, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	var resp *models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetDispute(ctx, disputeID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	var resp []*models.Dispute
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListDisputes(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	var resp *models.DisputeStats
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetDisputeStats(ctx)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error) {
	var id string
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		id, err = ep.provider.CreateCustomer(ctx, req)
		return err
	})
	return id, err
}

func (ep *ProviderWrapper) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.UpdateCustomer(ctx, customerID, req)
	})
}

func (ep *ProviderWrapper) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	var resp *models.Customer
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetCustomer(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) DeleteCustomer(ctx context.Context, customerID string) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.DeleteCustomer(ctx, customerID)
	})
}

func (ep *ProviderWrapper) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	var resp *models.PaymentMethod
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.CreatePaymentMethod(ctx, req)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	var resp *models.PaymentMethod
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.GetPaymentMethod(ctx, paymentMethodID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	var resp []*models.PaymentMethod
	err := ep.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = ep.provider.ListPaymentMethods(ctx, customerID)
		return err
	})
	return resp, err
}

func (ep *ProviderWrapper) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.AttachPaymentMethod(ctx, paymentMethodID, customerID)
	})
}

func (ep *ProviderWrapper) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return ep.circuitBreaker.Execute(ctx, func() error {
		return ep.provider.DetachPaymentMethod(ctx, paymentMethodID)
	})
}

func (ep *ProviderWrapper) IsAvailable(ctx context.Context) bool {
	return ep.healthChecker.IsHealthy() && ep.circuitBreaker.GetState() != utils.StateOpen
}

func (ep *ProviderWrapper) GetHealthStatus() utils.HealthStatus {
	return ep.healthChecker.GetStatus()
}

func (ep *ProviderWrapper) GetCircuitBreakerState() utils.CircuitState {
	return ep.circuitBreaker.GetState()
}

func (ep *ProviderWrapper) Stop() {
	ep.healthChecker.Stop()
}
