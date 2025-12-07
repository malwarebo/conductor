package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/malwarebo/conductor/models"
)

var (
	ErrNotSupported = errors.New("feature not supported by provider")
)

type ProviderCapabilities struct {
	SupportsInvoices        bool
	SupportsPayouts         bool
	SupportsPaymentSessions bool
	Supports3DS             bool
	SupportsManualCapture   bool
	SupportsBalance         bool
	SupportedCurrencies     []string
	SupportedPaymentMethods []models.PaymentMethodType
}

type PaymentProvider interface {
	Name() string
	Capabilities() ProviderCapabilities

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

	IsAvailable(ctx context.Context) bool
}

type InvoiceProvider interface {
	CreateInvoice(ctx context.Context, req *models.CreateInvoiceRequest) (*models.Invoice, error)
	GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error)
	ListInvoices(ctx context.Context, req *models.ListInvoicesRequest) ([]*models.Invoice, error)
	CancelInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error)
}

type PayoutProvider interface {
	CreatePayout(ctx context.Context, req *models.CreatePayoutRequest) (*models.Payout, error)
	GetPayout(ctx context.Context, payoutID string) (*models.Payout, error)
	ListPayouts(ctx context.Context, req *models.ListPayoutsRequest) ([]*models.Payout, error)
	CancelPayout(ctx context.Context, payoutID string) (*models.Payout, error)
	GetPayoutChannels(ctx context.Context, currency string) ([]*models.PayoutChannel, error)
}

type PaymentSessionProvider interface {
	CreatePaymentSession(ctx context.Context, req *models.CreatePaymentSessionRequest) (*models.PaymentSession, error)
	GetPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error)
	UpdatePaymentSession(ctx context.Context, sessionID string, req *models.UpdatePaymentSessionRequest) (*models.PaymentSession, error)
	ConfirmPaymentSession(ctx context.Context, sessionID string, req *models.ConfirmPaymentSessionRequest) (*models.PaymentSession, error)
	CapturePaymentSession(ctx context.Context, sessionID string, amount *int64) (*models.PaymentSession, error)
	CancelPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error)
	ListPaymentSessions(ctx context.Context, req *models.ListPaymentSessionsRequest) ([]*models.PaymentSession, error)
}

type PaymentMethodProvider interface {
	CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error)
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error)
	ListPaymentMethods(ctx context.Context, customerID string, pmType *models.PaymentMethodType) ([]*models.PaymentMethod, error)
	AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	ExpirePaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error)
}

type BalanceProvider interface {
	GetBalance(ctx context.Context, currency string) (*models.Balance, error)
}

type CaptureProvider interface {
	CapturePayment(ctx context.Context, paymentID string, amount int64) error
}

type VoidProvider interface {
	VoidPayment(ctx context.Context, paymentID string) error
}

type ThreeDSecureProvider interface {
	Create3DSSession(ctx context.Context, paymentID string, returnURL string) (*ThreeDSecureSession, error)
	Confirm3DSPayment(ctx context.Context, paymentID string) (*models.ChargeResponse, error)
}

type ThreeDSecureSession struct {
	PaymentID    string `json:"payment_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	Status       string `json:"status"`
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

func ConvertMetadataToStringMap(m map[string]interface{}) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func ConvertInterfaceMetadataToStringMap(m interface{}) map[string]string {
	if m == nil {
		return nil
	}
	if metadataMap, ok := m.(map[string]interface{}); ok {
		return ConvertMetadataToStringMap(metadataMap)
	}
	return nil
}

func ConvertStringMapToMetadata(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}
