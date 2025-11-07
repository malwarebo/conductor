package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/dispute"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/plan"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeProvider struct {
	apiKey        string
	webhookSecret string
}

func CreateStripeProvider(apiKey string) *StripeProvider {
	stripe.Key = apiKey
	return &StripeProvider{
		apiKey: apiKey,
	}
}

func CreateStripeProviderWithWebhook(apiKey, webhookSecret string) *StripeProvider {
	stripe.Key = apiKey
	return &StripeProvider{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
	}
}

func (p *StripeProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(req.Amount),
		Currency:    stripe.String(req.Currency),
		Description: stripe.String(req.Description),
		Customer:    stripe.String(req.CustomerID),
	}

	if req.PaymentMethod != "" {
		params.PaymentMethod = stripe.String(req.PaymentMethod)
		params.Confirm = stripe.Bool(true)
	}

	params.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{
		Enabled: stripe.Bool(true),
	}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe payment intent creation failed: %w", err)
	}

	metadata := make(map[string]interface{})
	for k, v := range pi.Metadata {
		metadata[k] = v
	}

	status := models.PaymentStatusPending
	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		status = models.PaymentStatusSuccess
	} else if pi.Status == stripe.PaymentIntentStatusRequiresAction {
		status = models.PaymentStatusPending
	} else if pi.Status == stripe.PaymentIntentStatusCanceled {
		status = models.PaymentStatusFailed
	}

	paymentMethodID := ""
	if pi.PaymentMethod != nil {
		paymentMethodID = pi.PaymentMethod.ID
	}

	response := &models.ChargeResponse{
		ID:               pi.ID,
		CustomerID:       req.CustomerID,
		Amount:           pi.Amount,
		Currency:         string(pi.Currency),
		Status:           status,
		PaymentMethod:    paymentMethodID,
		Description:      req.Description,
		ProviderName:     "stripe",
		ProviderChargeID: pi.ID,
		Metadata:         metadata,
		CreatedAt:        time.Unix(pi.Created, 0),
	}

	if pi.NextAction != nil && pi.NextAction.Type == "redirect_to_url" {
		response.Metadata["requires_action"] = true
		response.Metadata["next_action_type"] = string(pi.NextAction.Type)
		if pi.NextAction.RedirectToURL != nil {
			response.Metadata["redirect_url"] = pi.NextAction.RedirectToURL.URL
		}
	}

	return response, nil
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
	if p.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	_, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

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
	params := &stripe.SubscriptionParams{
		// No need to set ID in params, it's used in the API call
	}

	if req.PlanID != nil && *req.PlanID != "" {
		itemsParams := &stripe.SubscriptionItemsParams{
			Plan: stripe.String(*req.PlanID),
		}

		if req.Quantity != nil && *req.Quantity > 0 {
			itemsParams.Quantity = stripe.Int64(int64(*req.Quantity))
		}

		params.Items = []*stripe.SubscriptionItemsParams{itemsParams}
	}

	if req.PaymentMethodID != nil && *req.PaymentMethodID != "" {
		params.DefaultPaymentMethod = stripe.String(*req.PaymentMethodID)
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

	sub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return nil, err
	}

	result := &models.Subscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             models.SubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.Created, 0),
		CurrentPeriodEnd:   time.Unix(sub.CanceledAt, 0),
		ProviderName:       "stripe",
		UpdatedAt:          time.Now(),
	}

	if req.Quantity != nil {
		result.Quantity = *req.Quantity
	}

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

func (p *StripeProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	params := &stripe.SubscriptionParams{}

	if req.CancelAtPeriodEnd {
		params.CancelAtPeriodEnd = stripe.Bool(true)
	}

	if req.Reason != "" {
		if params.Metadata == nil {
			params.Metadata = make(map[string]string)
		}
		params.Metadata["cancellation_reason"] = req.Reason
	}

	var sub *stripe.Subscription
	var err error

	if req.CancelAtPeriodEnd {
		// Update subscription to cancel at period end
		sub, err = subscription.Update(subscriptionID, params)
	} else {
		cancelParams := &stripe.SubscriptionCancelParams{
			Prorate: stripe.Bool(true),
		}

		// Metadata handling removed as it's deprecated in Stripe API

		sub, err = subscription.Cancel(subscriptionID, cancelParams)
	}

	if err != nil {
		return nil, err
	}

	canceledAt := time.Now()
	if sub.CancelAt > 0 {
		canceledAt = time.Unix(sub.CancelAt, 0)
	}

	result := &models.Subscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             models.SubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.Created, 0),
		CurrentPeriodEnd:   time.Unix(sub.CanceledAt, 0),
		CanceledAt:         &canceledAt,
		ProviderName:       "stripe",
		UpdatedAt:          time.Now(),
	}

	return result, nil
}

func (p *StripeProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	sub, err := subscription.Get(subscriptionID, params)
	if err != nil {
		return nil, err
	}

	result := &models.Subscription{
		ID:                 sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             models.SubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.Created, 0),
		CurrentPeriodEnd:   time.Unix(sub.CanceledAt, 0),
		CanceledAt:         nil,
		ProviderName:       "stripe",
	}

	if sub.CancelAt > 0 {
		canceledAt := time.Unix(sub.CancelAt, 0)
		result.CanceledAt = &canceledAt
	}

	return result, nil
}

func (p *StripeProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}

	i := subscription.List(params)
	var subscriptions []*models.Subscription

	for i.Next() {
		sub := i.Subscription()
		result := &models.Subscription{
			ID:                 sub.ID,
			CustomerID:         sub.Customer.ID,
			Status:             models.SubscriptionStatus(sub.Status),
			CurrentPeriodStart: time.Unix(sub.Created, 0),
			CurrentPeriodEnd:   time.Unix(sub.CanceledAt, 0),
			CanceledAt:         nil,
			ProviderName:       "stripe",
		}

		if sub.CancelAt > 0 {
			canceledAt := time.Unix(sub.CancelAt, 0)
			result.CanceledAt = &canceledAt
		}

		subscriptions = append(subscriptions, result)
	}

	return subscriptions, nil
}

func (p *StripeProvider) CreatePlan(ctx context.Context, planReq *models.Plan) (*models.Plan, error) {
	params := &stripe.PlanParams{
		Amount:   stripe.Int64(int64(planReq.Amount * 100)),
		Currency: stripe.String(planReq.Currency),
		Interval: stripe.String(string(planReq.BillingPeriod)),
		Product: &stripe.PlanProductParams{
			Name: stripe.String(planReq.Name),
		},
	}

	if planReq.TrialDays > 0 {
		params.TrialPeriodDays = stripe.Int64(int64(planReq.TrialDays))
	}

	if planReq.Metadata != nil {
		params.Metadata = make(map[string]string)
		if metadataMap, ok := planReq.Metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				if str, ok := v.(string); ok {
					params.Metadata[k] = str
				} else {
					params.Metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	stripePlan, err := plan.New(params)
	if err != nil {
		return nil, err
	}

	result := &models.Plan{
		ID:            stripePlan.ID,
		Name:          planReq.Name,
		Description:   planReq.Description,
		Amount:        float64(stripePlan.Amount) / 100,
		Currency:      string(stripePlan.Currency),
		BillingPeriod: models.BillingPeriod(stripePlan.Interval),
		PricingType:   models.PricingTypeFixed,
		TrialDays:     planReq.TrialDays,
		Features:      planReq.Features,
		Metadata:      planReq.Metadata,
		CreatedAt:     time.Unix(stripePlan.Created, 0),
		UpdatedAt:     time.Unix(stripePlan.Created, 0),
	}

	return result, nil
}

func (p *StripeProvider) UpdatePlan(ctx context.Context, planID string, planReq *models.Plan) (*models.Plan, error) {
	params := &stripe.PlanParams{}

	if planReq.Name != "" {
		params.Product = &stripe.PlanProductParams{
			Name: stripe.String(planReq.Name),
		}
	}

	if planReq.Metadata != nil {
		params.Metadata = make(map[string]string)
		if metadataMap, ok := planReq.Metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				if str, ok := v.(string); ok {
					params.Metadata[k] = str
				} else {
					params.Metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	stripePlan, err := plan.Update(planID, params)
	if err != nil {
		return nil, err
	}

	result := &models.Plan{
		ID:            stripePlan.ID,
		Name:          planReq.Name,
		Description:   planReq.Description,
		Amount:        float64(stripePlan.Amount) / 100,
		Currency:      string(stripePlan.Currency),
		BillingPeriod: models.BillingPeriod(stripePlan.Interval),
		PricingType:   models.PricingTypeFixed,
		TrialDays:     planReq.TrialDays,
		Features:      planReq.Features,
		Metadata:      planReq.Metadata,
		UpdatedAt:     time.Now(),
	}

	return result, nil
}

func (p *StripeProvider) DeletePlan(ctx context.Context, planID string) error {
	_, err := plan.Del(planID, nil)
	return err
}

func (p *StripeProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	stripePlan, err := plan.Get(planID, nil)
	if err != nil {
		return nil, err
	}

	result := &models.Plan{
		ID:            stripePlan.ID,
		Name:          stripePlan.Product.Name,
		Description:   stripePlan.Product.Description,
		Amount:        float64(stripePlan.Amount) / 100,
		Currency:      string(stripePlan.Currency),
		BillingPeriod: models.BillingPeriod(stripePlan.Interval),
		PricingType:   models.PricingTypeFixed,
		TrialDays:     int(stripePlan.TrialPeriodDays),
		Features:      []string{},
		Metadata:      nil,
		CreatedAt:     time.Unix(stripePlan.Created, 0),
		UpdatedAt:     time.Unix(stripePlan.Created, 0),
	}

	if stripePlan.Metadata != nil {
		metadata := make(map[string]interface{})
		for k, v := range stripePlan.Metadata {
			metadata[k] = v
		}
		result.Metadata = metadata
	}

	return result, nil
}

func (p *StripeProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	params := &stripe.PlanListParams{}
	i := plan.List(params)
	var plans []*models.Plan

	for i.Next() {
		stripePlan := i.Plan()
		result := &models.Plan{
			ID:            stripePlan.ID,
			Name:          stripePlan.Product.Name,
			Description:   stripePlan.Product.Description,
			Amount:        float64(stripePlan.Amount) / 100,
			Currency:      string(stripePlan.Currency),
			BillingPeriod: models.BillingPeriod(stripePlan.Interval),
			PricingType:   models.PricingTypeFixed,
			TrialDays:     int(stripePlan.TrialPeriodDays),
			Features:      []string{},
			Metadata:      nil,
			CreatedAt:     time.Unix(stripePlan.Created, 0),
			UpdatedAt:     time.Unix(stripePlan.Created, 0),
		}

		if stripePlan.Metadata != nil {
			metadata := make(map[string]interface{})
			for k, v := range stripePlan.Metadata {
				metadata[k] = v
			}
			result.Metadata = metadata
		}

		plans = append(plans, result)
	}

	return plans, nil
}

func (p *StripeProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("stripe: disputes are created automatically from charges, cannot create manually")
}

func (p *StripeProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	params := &stripe.DisputeParams{}

	if req.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			} else {
				params.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	stripeDispute, err := dispute.Update(disputeID, params)
	if err != nil {
		return nil, err
	}

	result := &models.Dispute{
		ID:        stripeDispute.ID,
		Status:    models.DisputeStatus(stripeDispute.Status),
		Metadata:  req.Metadata,
		UpdatedAt: time.Now(),
	}

	return result, nil
}

func (p *StripeProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	params := &stripe.DisputeEvidenceParams{}

	if req.Type != "" {
		switch req.Type {
		case "customer_email_address":
			params.CustomerEmailAddress = stripe.String(req.Description)
		case "customer_purchase_ip":
			params.CustomerPurchaseIP = stripe.String(req.Description)
		case "customer_signature":
			params.CustomerSignature = stripe.String(req.Description)
		case "billing_address":
			params.BillingAddress = stripe.String(req.Description)
		case "receipt":
			params.Receipt = stripe.String(req.Description)
		case "service_date":
			params.ServiceDate = stripe.String(req.Description)
		case "product_description":
			params.ProductDescription = stripe.String(req.Description)
		case "customer_name":
			params.CustomerName = stripe.String(req.Description)
		case "customer_communication":
			params.CustomerCommunication = stripe.String(req.Description)
		case "duplicate_charge_documentation":
			params.DuplicateChargeDocumentation = stripe.String(req.Description)
		case "duplicate_charge_explanation":
			params.DuplicateChargeExplanation = stripe.String(req.Description)
		case "duplicate_charge_id":
			params.DuplicateChargeID = stripe.String(req.Description)
		case "refund_policy":
			params.RefundPolicy = stripe.String(req.Description)
		case "refund_policy_disclosure":
			params.RefundPolicyDisclosure = stripe.String(req.Description)
		case "refund_refusal_explanation":
			params.RefundRefusalExplanation = stripe.String(req.Description)
		case "cancellation_policy":
			params.CancellationPolicy = stripe.String(req.Description)
		case "cancellation_policy_disclosure":
			params.CancellationPolicyDisclosure = stripe.String(req.Description)
		case "cancellation_rebuttal":
			params.CancellationRebuttal = stripe.String(req.Description)
		case "access_activity_log":
			params.AccessActivityLog = stripe.String(req.Description)
		case "shipping_address":
			params.ShippingAddress = stripe.String(req.Description)
		case "shipping_carrier":
			params.ShippingCarrier = stripe.String(req.Description)
		case "shipping_date":
			params.ShippingDate = stripe.String(req.Description)
		case "shipping_documentation":
			params.ShippingDocumentation = stripe.String(req.Description)
		case "shipping_tracking_number":
			params.ShippingTrackingNumber = stripe.String(req.Description)
		case "uncategorized_file":
			params.UncategorizedFile = stripe.String(req.Description)
		}
	}

	_, err := dispute.Update(disputeID, &stripe.DisputeParams{
		Evidence: params,
	})
	if err != nil {
		return nil, err
	}

	result := &models.Evidence{
		ID:          fmt.Sprintf("evid_%s", disputeID),
		DisputeID:   disputeID,
		Type:        req.Type,
		Description: req.Description,
		Files:       req.Files,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return result, nil
}

func (p *StripeProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	stripeDispute, err := dispute.Get(disputeID, nil)
	if err != nil {
		return nil, err
	}

	result := &models.Dispute{
		ID:            stripeDispute.ID,
		TransactionID: stripeDispute.Charge.ID,
		Amount:        stripeDispute.Amount,
		Currency:      string(stripeDispute.Currency),
		Reason:        string(stripeDispute.Reason),
		Status:        models.DisputeStatus(stripeDispute.Status),
		Evidence:      make(map[string]interface{}),
		DueBy:         time.Unix(stripeDispute.EvidenceDetails.DueBy, 0),
		CreatedAt:     time.Unix(stripeDispute.Created, 0),
		UpdatedAt:     time.Unix(stripeDispute.Created, 0),
	}

	if stripeDispute.Metadata != nil {
		metadata := make(map[string]interface{})
		for k, v := range stripeDispute.Metadata {
			metadata[k] = v
		}
		result.Metadata = metadata
	}

	return result, nil
}

func (p *StripeProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	params := &stripe.DisputeListParams{}
	i := dispute.List(params)
	var disputes []*models.Dispute

	for i.Next() {
		stripeDispute := i.Dispute()
		result := &models.Dispute{
			ID:            stripeDispute.ID,
			TransactionID: stripeDispute.Charge.ID,
			Amount:        stripeDispute.Amount,
			Currency:      string(stripeDispute.Currency),
			Reason:        string(stripeDispute.Reason),
			Status:        models.DisputeStatus(stripeDispute.Status),
			Evidence:      make(map[string]interface{}),
			DueBy:         time.Unix(stripeDispute.EvidenceDetails.DueBy, 0),
			CreatedAt:     time.Unix(stripeDispute.Created, 0),
			UpdatedAt:     time.Unix(stripeDispute.Created, 0),
		}

		if stripeDispute.Metadata != nil {
			metadata := make(map[string]interface{})
			for k, v := range stripeDispute.Metadata {
				metadata[k] = v
			}
			result.Metadata = metadata
		}

		disputes = append(disputes, result)
	}

	return disputes, nil
}

func (p *StripeProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	params := &stripe.DisputeListParams{}
	i := dispute.List(params)

	stats := &models.DisputeStats{}

	for i.Next() {
		stripeDispute := i.Dispute()
		stats.Total++

		switch stripeDispute.Status {
		case "needs_response":
			stats.Open++
		case "won":
			stats.Won++
		case "lost":
			stats.Lost++
		case "warning_needs_response":
			stats.Open++
		case "warning_under_review":
			stats.Open++
		case "under_review":
			stats.Open++
		case "charge_refunded":
			stats.Canceled++
		case "won_charge_refunded":
			stats.Won++
		case "lost_charge_refunded":
			stats.Lost++
		}
	}

	return stats, nil
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
