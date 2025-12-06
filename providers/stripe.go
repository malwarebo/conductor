package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/dispute"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/paymentmethod"
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

	if req.CaptureMethod == models.CaptureMethodManual || (req.Capture != nil && !*req.Capture) {
		params.CaptureMethod = stripe.String("manual")
	}

	if req.ReturnURL != "" {
		params.ReturnURL = stripe.String(req.ReturnURL)
	}

	params.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{
		Enabled:        stripe.Bool(true),
		AllowRedirects: stripe.String("always"),
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe payment intent creation failed: %w", err)
	}

	metadata := ConvertStringMapToMetadata(pi.Metadata)

	status := p.mapPaymentIntentStatus(pi.Status)
	captureMethod := models.CaptureMethodAutomatic
	if pi.CaptureMethod == stripe.PaymentIntentCaptureMethodManual {
		captureMethod = models.CaptureMethodManual
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
		CaptureMethod:    captureMethod,
		CapturedAmount:   pi.AmountReceived,
		ClientSecret:     pi.ClientSecret,
		Metadata:         metadata,
		CreatedAt:        time.Unix(pi.Created, 0),
	}

	if pi.NextAction != nil {
		response.RequiresAction = true
		response.NextActionType = string(pi.NextAction.Type)
		if pi.NextAction.RedirectToURL != nil {
			response.NextActionURL = pi.NextAction.RedirectToURL.URL
		}
		if pi.NextAction.UseStripeSDK != nil {
			response.NextActionType = "use_stripe_sdk"
		}
	}

	return response, nil
}

func (p *StripeProvider) mapPaymentIntentStatus(status stripe.PaymentIntentStatus) models.PaymentStatus {
	switch status {
	case stripe.PaymentIntentStatusSucceeded:
		return models.PaymentStatusSuccess
	case stripe.PaymentIntentStatusRequiresAction:
		return models.PaymentStatusRequiresAction
	case stripe.PaymentIntentStatusRequiresCapture:
		return models.PaymentStatusRequiresCapture
	case stripe.PaymentIntentStatusProcessing:
		return models.PaymentStatusProcessing
	case stripe.PaymentIntentStatusCanceled:
		return models.PaymentStatusCanceled
	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		return models.PaymentStatusFailed
	default:
		return models.PaymentStatusPending
	}
}

func (p *StripeProvider) CapturePayment(ctx context.Context, paymentID string, amount int64) error {
	params := &stripe.PaymentIntentCaptureParams{}
	if amount > 0 {
		params.AmountToCapture = stripe.Int64(amount)
	}

	_, err := paymentintent.Capture(paymentID, params)
	if err != nil {
		return fmt.Errorf("stripe capture failed: %w", err)
	}
	return nil
}

func (p *StripeProvider) VoidPayment(ctx context.Context, paymentID string) error {
	params := &stripe.PaymentIntentCancelParams{
		CancellationReason: stripe.String("requested_by_customer"),
	}

	_, err := paymentintent.Cancel(paymentID, params)
	if err != nil {
		return fmt.Errorf("stripe void/cancel failed: %w", err)
	}
	return nil
}

func (p *StripeProvider) Create3DSSession(ctx context.Context, paymentID string, returnURL string) (*ThreeDSecureSession, error) {
	pi, err := paymentintent.Get(paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get payment intent failed: %w", err)
	}

	session := &ThreeDSecureSession{
		PaymentID:    pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       string(pi.Status),
	}

	if pi.NextAction != nil && pi.NextAction.RedirectToURL != nil {
		session.RedirectURL = pi.NextAction.RedirectToURL.URL
	}

	return session, nil
}

func (p *StripeProvider) Confirm3DSPayment(ctx context.Context, paymentID string) (*models.ChargeResponse, error) {
	pi, err := paymentintent.Get(paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get payment intent failed: %w", err)
	}

	status := p.mapPaymentIntentStatus(pi.Status)
	captureMethod := models.CaptureMethodAutomatic
	if pi.CaptureMethod == stripe.PaymentIntentCaptureMethodManual {
		captureMethod = models.CaptureMethodManual
	}

	paymentMethodID := ""
	if pi.PaymentMethod != nil {
		paymentMethodID = pi.PaymentMethod.ID
	}

	return &models.ChargeResponse{
		ID:               pi.ID,
		CustomerID:       pi.Customer.ID,
		Amount:           pi.Amount,
		Currency:         string(pi.Currency),
		Status:           status,
		PaymentMethod:    paymentMethodID,
		ProviderName:     "stripe",
		ProviderChargeID: pi.ID,
		CaptureMethod:    captureMethod,
		CapturedAmount:   pi.AmountReceived,
		ClientSecret:     pi.ClientSecret,
		CreatedAt:        time.Unix(pi.Created, 0),
	}, nil
}

func (p *StripeProvider) CreatePaymentIntent(ctx context.Context, req *models.CreatePaymentIntentRequest) (*models.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(req.Currency),
	}

	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}

	if req.Description != "" {
		params.Description = stripe.String(req.Description)
	}

	if req.PaymentMethod != "" {
		params.PaymentMethod = stripe.String(req.PaymentMethod)
	}

	if req.CaptureMethod == models.CaptureMethodManual {
		params.CaptureMethod = stripe.String("manual")
	}

	if req.SetupFutureUsage != "" {
		params.SetupFutureUsage = stripe.String(req.SetupFutureUsage)
	}

	if req.ReturnURL != "" {
		params.ReturnURL = stripe.String(req.ReturnURL)
	}

	params.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{
		Enabled: stripe.Bool(true),
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe create payment intent failed: %w", err)
	}

	return p.mapPaymentIntent(pi), nil
}

func (p *StripeProvider) GetPaymentIntent(ctx context.Context, paymentIntentID string) (*models.PaymentIntent, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get payment intent failed: %w", err)
	}

	return p.mapPaymentIntent(pi), nil
}

func (p *StripeProvider) UpdatePaymentIntent(ctx context.Context, paymentIntentID string, req *models.UpdatePaymentIntentRequest) (*models.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{}

	if req.Amount != nil {
		params.Amount = stripe.Int64(*req.Amount)
	}

	if req.Currency != nil {
		params.Currency = stripe.String(*req.Currency)
	}

	if req.Description != nil {
		params.Description = stripe.String(*req.Description)
	}

	if req.PaymentMethod != nil {
		params.PaymentMethod = stripe.String(*req.PaymentMethod)
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	pi, err := paymentintent.Update(paymentIntentID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe update payment intent failed: %w", err)
	}

	return p.mapPaymentIntent(pi), nil
}

func (p *StripeProvider) ConfirmPaymentIntent(ctx context.Context, paymentIntentID string, req *models.ConfirmPaymentIntentRequest) (*models.PaymentIntent, error) {
	params := &stripe.PaymentIntentConfirmParams{}

	if req.PaymentMethod != "" {
		params.PaymentMethod = stripe.String(req.PaymentMethod)
	}

	if req.ReturnURL != "" {
		params.ReturnURL = stripe.String(req.ReturnURL)
	}

	pi, err := paymentintent.Confirm(paymentIntentID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe confirm payment intent failed: %w", err)
	}

	return p.mapPaymentIntent(pi), nil
}

func (p *StripeProvider) ListPaymentIntents(ctx context.Context, req *models.ListPaymentIntentsRequest) ([]*models.PaymentIntent, error) {
	params := &stripe.PaymentIntentListParams{}

	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}

	if req.Limit > 0 {
		params.Limit = stripe.Int64(int64(req.Limit))
	}

	i := paymentintent.List(params)
	var intents []*models.PaymentIntent

	for i.Next() {
		intents = append(intents, p.mapPaymentIntent(i.PaymentIntent()))
	}

	return intents, nil
}

func (p *StripeProvider) mapPaymentIntent(pi *stripe.PaymentIntent) *models.PaymentIntent {
	intent := &models.PaymentIntent{
		ID:             pi.ID,
		Amount:         pi.Amount,
		Currency:       string(pi.Currency),
		Status:         string(pi.Status),
		ClientSecret:   pi.ClientSecret,
		CaptureMethod:  string(pi.CaptureMethod),
		CapturedAmount: pi.AmountReceived,
		ProviderName:   "stripe",
		CreatedAt:      time.Unix(pi.Created, 0),
	}

	if pi.Customer != nil {
		intent.CustomerID = pi.Customer.ID
	}

	if pi.PaymentMethod != nil {
		intent.PaymentMethod = pi.PaymentMethod.ID
	}

	if pi.Description != "" {
		intent.Description = pi.Description
	}

	if pi.NextAction != nil {
		intent.RequiresAction = true
		intent.NextActionType = string(pi.NextAction.Type)
		if pi.NextAction.RedirectToURL != nil {
			intent.NextActionURL = pi.NextAction.RedirectToURL.URL
		}
	}

	if pi.Metadata != nil {
		intent.Metadata = ConvertStringMapToMetadata(pi.Metadata)
	}

	return intent
}

func (p *StripeProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.PaymentID),
		Amount:        stripe.Int64(req.Amount), // Amount is already in cents
		Reason:        stripe.String(req.Reason),
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	ref, err := refund.New(params)
	if err != nil {
		return nil, err
	}

	metadata := ConvertStringMapToMetadata(ref.Metadata)

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
		params.Metadata = ConvertInterfaceMetadataToStringMap(req.Metadata)
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
		params.Metadata = ConvertInterfaceMetadataToStringMap(req.Metadata)
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
		params.Metadata = ConvertInterfaceMetadataToStringMap(planReq.Metadata)
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
		params.Metadata = ConvertInterfaceMetadataToStringMap(planReq.Metadata)
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
		result.Metadata = ConvertStringMapToMetadata(stripePlan.Metadata)
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
			result.Metadata = ConvertStringMapToMetadata(stripePlan.Metadata)
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
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
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
		result.Metadata = ConvertStringMapToMetadata(stripeDispute.Metadata)
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
			result.Metadata = ConvertStringMapToMetadata(stripeDispute.Metadata)
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

func (p *StripeProvider) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	if req.Name != "" {
		params.Name = stripe.String(req.Name)
	}

	if req.Phone != "" {
		params.Phone = stripe.String(req.Phone)
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe customer creation failed: %w", err)
	}

	return cust.ID, nil
}

func (p *StripeProvider) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	params := &stripe.CustomerParams{}

	if req.Email != "" {
		params.Email = stripe.String(req.Email)
	}

	if req.Name != "" {
		params.Name = stripe.String(req.Name)
	}

	if req.Phone != "" {
		params.Phone = stripe.String(req.Phone)
	}

	if req.Metadata != nil {
		params.Metadata = ConvertMetadataToStringMap(req.Metadata)
	}

	_, err := customer.Update(customerID, params)
	return err
}

func (p *StripeProvider) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, err
	}

	return &models.Customer{
		ExternalID: cust.ID,
		Email:      cust.Email,
		Name:       cust.Name,
		Phone:      cust.Phone,
		Metadata:   ConvertStringMapToMetadata(cust.Metadata),
		CreatedAt:  time.Unix(cust.Created, 0),
	}, nil
}

func (p *StripeProvider) DeleteCustomer(ctx context.Context, customerID string) error {
	_, err := customer.Del(customerID, nil)
	return err
}

func (p *StripeProvider) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	pm, err := paymentmethod.Get(req.PaymentMethodID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe payment method get failed: %w", err)
	}

	result := &models.PaymentMethod{
		CustomerID:              req.CustomerID,
		ProviderName:            "stripe",
		ProviderPaymentMethodID: pm.ID,
		Type:                    string(pm.Type),
		IsDefault:               req.IsDefault,
		Metadata:                req.Metadata,
		CreatedAt:               time.Unix(pm.Created, 0),
	}

	if pm.Card != nil {
		result.Last4 = pm.Card.Last4
		result.Brand = string(pm.Card.Brand)
		result.ExpMonth = int(pm.Card.ExpMonth)
		result.ExpYear = int(pm.Card.ExpYear)
	}

	return result, nil
}

func (p *StripeProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	pm, err := paymentmethod.Get(paymentMethodID, nil)
	if err != nil {
		return nil, err
	}

	customerID := ""
	if pm.Customer != nil {
		customerID = pm.Customer.ID
	}

	result := &models.PaymentMethod{
		CustomerID:              customerID,
		ProviderName:            "stripe",
		ProviderPaymentMethodID: pm.ID,
		Type:                    string(pm.Type),
		CreatedAt:               time.Unix(pm.Created, 0),
	}

	if pm.Card != nil {
		result.Last4 = pm.Card.Last4
		result.Brand = string(pm.Card.Brand)
		result.ExpMonth = int(pm.Card.ExpMonth)
		result.ExpYear = int(pm.Card.ExpYear)
	}

	return result, nil
}

func (p *StripeProvider) ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String("card"),
	}

	i := paymentmethod.List(params)
	var paymentMethods []*models.PaymentMethod

	for i.Next() {
		pm := i.PaymentMethod()
		result := &models.PaymentMethod{
			CustomerID:              customerID,
			ProviderName:            "stripe",
			ProviderPaymentMethodID: pm.ID,
			Type:                    string(pm.Type),
			CreatedAt:               time.Unix(pm.Created, 0),
		}

		if pm.Card != nil {
			result.Last4 = pm.Card.Last4
			result.Brand = string(pm.Card.Brand)
			result.ExpMonth = int(pm.Card.ExpMonth)
			result.ExpYear = int(pm.Card.ExpYear)
		}

		paymentMethods = append(paymentMethods, result)
	}

	return paymentMethods, nil
}

func (p *StripeProvider) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	_, err := paymentmethod.Attach(paymentMethodID, params)
	return err
}

func (p *StripeProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := paymentmethod.Detach(paymentMethodID, nil)
	return err
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
