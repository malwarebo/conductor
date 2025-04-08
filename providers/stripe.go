package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/charge"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/subscription"
)

type StripeProvider struct {
	apiKey string
}

func NewStripeProvider(apiKey string) *StripeProvider {
	stripe.Key = apiKey
	return &StripeProvider{
		apiKey: apiKey,
	}
}

func (p *StripeProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(req.Amount), // Amount is already in cents
		Currency:    stripe.String(req.Currency),
		Description: stripe.String(req.Description),
		Customer:    stripe.String(req.CustomerID),
	}

	// Set payment method
	if req.PaymentMethod != "" {
		params.SetSource(req.PaymentMethod)
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}

	ch, err := charge.New(params)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	for k, v := range ch.Metadata {
		metadata[k] = v
	}

	return &models.ChargeResponse{
		ID:               ch.ID,
		CustomerID:       req.CustomerID,
		Amount:           ch.Amount,
		Currency:         string(ch.Currency),
		Status:           models.PaymentStatusSuccess,
		PaymentMethod:    ch.Source.ID,
		Description:      req.Description,
		ProviderName:     "stripe",
		ProviderChargeID: ch.ID,
		Metadata:         metadata,
		CreatedAt:        time.Unix(ch.Created, 0),
	}, nil
}

func (p *StripeProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.PaymentID),
		Amount:        stripe.Int64(req.Amount), // Amount is already in cents
		Reason:        stripe.String(req.Reason),
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}

	ref, err := refund.New(params)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	for k, v := range ref.Metadata {
		metadata[k] = v
	}

	return &models.RefundResponse{
		ID:               ref.ID,
		PaymentID:        req.PaymentID,
		Amount:           ref.Amount,
		Currency:         string(ref.Currency),
		Status:           string(ref.Status),
		Reason:           req.Reason,
		ProviderName:     "stripe",
		ProviderRefundID: ref.ID,
		Metadata:         metadata,
		CreatedAt:        time.Unix(ref.Created, 0),
	}, nil
}

func (p *StripeProvider) ValidateWebhookSignature(payload []byte, signature string) error {
	// Implement webhook signature validation
	return nil
}
func (p *StripeProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(req.CustomerID),
	}

	if req.PlanID != "" {
		itemsParams := &stripe.SubscriptionItemsParams{
			Plan: stripe.String(req.PlanID),
		}

		if req.Quantity > 0 {
			itemsParams.Quantity = stripe.Int64(int64(req.Quantity))
		}

		params.Items = []*stripe.SubscriptionItemsParams{itemsParams}
	}

	if req.TrialDays != nil && *req.TrialDays > 0 {
		params.TrialPeriodDays = stripe.Int64(int64(*req.TrialDays))
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		if metadataMap, ok := req.Metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				if str, ok := v.(string); ok {
					params.Metadata[k] = str
				} else {
					params.Metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	sub, err := subscription.New(params)
	if err != nil {
		return nil, err
	}

	result := &models.Subscription{
		ID:                 sub.ID,
		CustomerID:         req.CustomerID,
		PlanID:             req.PlanID,
		Status:             models.SubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.Created, 0),
		CurrentPeriodEnd:   time.Unix(sub.CanceledAt, 0),
		Quantity:           req.Quantity,
		ProviderName:       "stripe",
		CreatedAt:          time.Unix(sub.Created, 0),
		UpdatedAt:          time.Unix(sub.Created, 0),
	}

	// Add trial dates if present
	if sub.TrialStart > 0 {
		trialStart := time.Unix(sub.TrialStart, 0)
		result.TrialStart = &trialStart
	}

	if sub.TrialEnd > 0 {
		trialEnd := time.Unix(sub.TrialEnd, 0)
		result.TrialEnd = &trialEnd
	}

	return result, nil
}

func (p *StripeProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("stripe: subscription update not implemented")
}

func (p *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	return nil, fmt.Errorf("stripe: subscription cancellation not implemented")
}

func (p *StripeProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return nil, fmt.Errorf("stripe: get subscription not implemented")
}

func (p *StripeProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return nil, fmt.Errorf("stripe: list subscriptions not implemented")
}

func (p *StripeProvider) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("stripe: create plan not implemented")
}

func (p *StripeProvider) UpdatePlan(ctx context.Context, planID string, plan *models.Plan) (*models.Plan, error) {
	return nil, fmt.Errorf("stripe: update plan not implemented")
}

func (p *StripeProvider) DeletePlan(ctx context.Context, planID string) error {
	return fmt.Errorf("stripe: delete plan not implemented")
}

func (p *StripeProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return nil, fmt.Errorf("stripe: get plan not implemented")
}

func (p *StripeProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return nil, fmt.Errorf("stripe: list plans not implemented")
}

func (p *StripeProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("stripe: create dispute not implemented")
}

func (p *StripeProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("stripe: update dispute not implemented")
}

func (p *StripeProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return nil, fmt.Errorf("stripe: submit dispute evidence not implemented")
}

func (p *StripeProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, fmt.Errorf("stripe: get dispute not implemented")
}

func (p *StripeProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return nil, fmt.Errorf("stripe: list disputes not implemented")
}

func (p *StripeProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return nil, fmt.Errorf("stripe: get dispute stats not implemented")
}

// TODO: need to add a better way to check if the provider is available
func (p *StripeProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}

	// Try to make a simple API call to check connectivity
	var account stripe.Account
	err := stripe.GetBackend(stripe.APIBackend).Call(
		"GET",
		"/v1/account",
		p.apiKey,
		nil,
		&account,
	)

	return err == nil
}
