package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
	refund "github.com/xendit/xendit-go/v6/refund"
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
	payerEmail := "customer@example.com"
	data := invoice.NewCreateInvoiceRequest(req.CustomerID, float64(req.Amount))
	data.PayerEmail = &payerEmail
	data.Description = &req.Description

	if req.Metadata != nil {
		metadata := make(map[string]interface{})
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		data.SetMetadata(metadata)
	}

	inv, _, err := p.client.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*data).Execute()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	status := models.PaymentStatusPending
	if inv.GetStatus() == "PAID" {
		status = models.PaymentStatusSuccess
	} else if inv.GetStatus() == "EXPIRED" {
		status = models.PaymentStatusFailed
	}

	return &models.ChargeResponse{
		ID:               inv.GetId(),
		CustomerID:       req.CustomerID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           status,
		PaymentMethod:    req.PaymentMethod,
		Description:      req.Description,
		ProviderName:     "xendit",
		ProviderChargeID: inv.GetId(),
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	refundData := refund.NewCreateRefund()
	refundData.SetInvoiceId(req.PaymentID)
	amount := float64(req.Amount)
	refundData.SetAmount(amount)
	refundData.SetReason(req.Reason)

	if req.Metadata != nil {
		metadata := make(map[string]interface{})
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		refundData.SetMetadata(metadata)
	}

	ref, _, err := p.client.RefundApi.CreateRefund(ctx).CreateRefund(*refundData).Execute()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	return &models.RefundResponse{
		ID:               ref.GetId(),
		PaymentID:        req.PaymentID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           "succeeded",
		Reason:           req.Reason,
		ProviderName:     "xendit",
		ProviderRefundID: ref.GetId(),
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscriptions not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscriptions not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscriptions not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscriptions not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return nil, fmt.Errorf("xendit: subscriptions not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) CreatePlan(ctx context.Context, planReq *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: plans not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) UpdatePlan(ctx context.Context, planID string, planReq *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: plans not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) DeletePlan(ctx context.Context, planID string) error {
	return fmt.Errorf("xendit: plans not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return nil, fmt.Errorf("xendit: plans not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return nil, fmt.Errorf("xendit: plans not supported, use payment requests for recurring payments")
}

func (p *XenditProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return nil, fmt.Errorf("xendit: disputes not supported, contact Xendit support for chargeback handling")
}

func (p *XenditProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	_, resp, err := p.client.InvoiceApi.GetInvoices(ctx).Execute()
	if err != nil && resp == nil {
		return false
	}

	return true
}
