package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
)

type XenditProvider struct {
	apiKey string
	client *xendit.APIClient
}

func NewXenditProvider(apiKey string) *XenditProvider {
	client := xendit.NewClient(apiKey)

	return &XenditProvider{
		apiKey: apiKey,
		client: client,
	}
}

func (p *XenditProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	// Create invoice
	payerEmail := "customer@example.com"
	data := invoice.NewCreateInvoiceRequest(req.CustomerID, float64(req.Amount))
	data.PayerEmail = &payerEmail
	data.Description = &req.Description

	inv, _, err := p.client.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*data).Execute()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	return &models.ChargeResponse{
		ID:               inv.GetId(),
		CustomerID:       req.CustomerID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           models.PaymentStatusPending,
		PaymentMethod:    req.PaymentMethod,
		Description:      req.Description,
		ProviderName:     "xendit",
		ProviderChargeID: inv.GetId(),
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	// Expire the invoice
	_, _, err := p.client.InvoiceApi.ExpireInvoice(ctx, req.PaymentID).Execute()
	if err != nil {
		return nil, err
	}

	return &models.RefundResponse{
		ID:               "ref_" + req.PaymentID,
		PaymentID:        req.PaymentID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           "succeeded",
		Reason:           req.Reason,
		ProviderName:     "xendit",
		ProviderRefundID: "ref_" + req.PaymentID,
		Metadata:         req.Metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscription creation not implemented")
}

func (p *XenditProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscription update not implemented")
}

func (p *XenditProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscription cancellation not implemented")
}

func (p *XenditProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: get subscription not implemented")
}

func (p *XenditProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: list subscriptions not implemented")
}

func (p *XenditProvider) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: create plan not implemented")
}

func (p *XenditProvider) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: update plan not implemented")
}

func (p *XenditProvider) DeletePlan(ctx context.Context, planID string) error {
	return fmt.Errorf("xendit: delete plan not implemented")
}

func (p *XenditProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: get plan not implemented")
}

func (p *XenditProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return nil, fmt.Errorf("xendit: list plans not implemented")
}

func (p *XenditProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: create dispute not implemented")
}

func (p *XenditProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: update dispute not implemented")
}

func (p *XenditProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return nil, fmt.Errorf("xendit: submit dispute evidence not implemented")
}

func (p *XenditProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: get dispute not implemented")
}

func (p *XenditProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: list disputes not implemented")
}

func (p *XenditProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return nil, fmt.Errorf("xendit: get dispute stats not implemented")
}

// TODO: telemetry should give insights if we can rely on this
func (p *XenditProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	// Try to make a simple API call to check connectivity
	_, resp, err := p.client.InvoiceApi.GetInvoices(ctx).Execute()
	if err != nil && resp == nil {
		return false
	}

	// Even if we get an authentication error (401), the API is still available
	// Just check if we could connect to the API
	return true
}
