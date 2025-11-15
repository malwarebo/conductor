package providers

import (
	"context"

	"github.com/malwarebo/conductor/models"
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
