package providers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/malwarebo/conductor/models"
	xendit "github.com/xendit/xendit-go/v6"
	"github.com/xendit/xendit-go/v6/customer"
	paymentrequest "github.com/xendit/xendit-go/v6/payment_request"
	refund "github.com/xendit/xendit-go/v6/refund"
)

type XenditProvider struct {
	apiKey        string
	webhookSecret string
	client        *xendit.APIClient
}

func CreateXenditProvider(apiKey string) *XenditProvider {
	client := xendit.NewClient(apiKey)

	return &XenditProvider{
		apiKey: apiKey,
		client: client,
	}
}

func CreateXenditProviderWithWebhook(apiKey, webhookSecret string) *XenditProvider {
	client := xendit.NewClient(apiKey)

	return &XenditProvider{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		client:        client,
	}
}

func (p *XenditProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	currency, err := p.getCurrency(req.Currency)
	if err != nil {
		return nil, fmt.Errorf("unsupported currency: %w", err)
	}

	paymentReq := paymentrequest.NewPaymentRequestParameters(currency)
	paymentReq.SetAmount(float64(req.Amount))
	paymentReq.SetReferenceId(req.CustomerID)
	paymentReq.SetDescription(req.Description)

	if req.PaymentMethod != "" {
		paymentReq.SetPaymentMethodId(req.PaymentMethod)
	}

	if req.Metadata != nil {
		metadata := make(map[string]interface{})
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		paymentReq.SetMetadata(metadata)
	}

	pr, _, err := p.client.PaymentRequestApi.CreatePaymentRequest(ctx).PaymentRequestParameters(*paymentReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit payment request creation failed: %w", err)
	}

	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		metadata = req.Metadata
	}

	status := models.PaymentStatusPending
	if pr.GetStatus() == "SUCCEEDED" {
		status = models.PaymentStatusSuccess
	} else if pr.GetStatus() == "FAILED" {
		status = models.PaymentStatusFailed
	} else if pr.GetStatus() == "PENDING" {
		status = models.PaymentStatusPending
	}

	response := &models.ChargeResponse{
		ID:               pr.GetId(),
		CustomerID:       req.CustomerID,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           status,
		PaymentMethod:    req.PaymentMethod,
		Description:      req.Description,
		ProviderName:     "xendit",
		ProviderChargeID: pr.GetId(),
		Metadata:         metadata,
		CreatedAt:        time.Now(),
	}

	if actions := pr.GetActions(); len(actions) > 0 {
		response.Metadata["requires_action"] = true
		for _, action := range actions {
			if action.GetAction() == "AUTH" {
				response.Metadata["auth_url"] = action.GetUrl()
				response.Metadata["auth_method"] = action.GetMethod()
			}
		}
	}

	return response, nil
}

func (p *XenditProvider) getCurrency(currency string) (paymentrequest.PaymentRequestCurrency, error) {
	switch currency {
	case "IDR":
		return paymentrequest.PAYMENTREQUESTCURRENCY_IDR, nil
	case "PHP":
		return paymentrequest.PAYMENTREQUESTCURRENCY_PHP, nil
	default:
		return paymentrequest.PAYMENTREQUESTCURRENCY_IDR, fmt.Errorf("unsupported currency: %s (Xendit Go SDK limited to IDR and PHP)", currency)
	}
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
	paymentReq := paymentrequest.NewPaymentRequestParameters(paymentrequest.PAYMENTREQUESTCURRENCY_IDR)
	amount := float64(req.Quantity * 1000)
	paymentReq.SetAmount(amount)
	paymentReq.SetReferenceId(req.CustomerID)

	if req.Metadata != nil {
		metadata := make(map[string]interface{})
		if metadataMap, ok := req.Metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				metadata[k] = v
			}
		}
		paymentReq.SetMetadata(metadata)
	}

	pr, _, err := p.client.PaymentRequestApi.CreatePaymentRequest(ctx).PaymentRequestParameters(*paymentReq).Execute()
	if err != nil {
		return nil, err
	}

	return &models.Subscription{
		ID:                 pr.GetId(),
		CustomerID:         req.CustomerID,
		PlanID:             req.PlanID,
		Status:             models.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		Quantity:           req.Quantity,
		ProviderName:       "xendit",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

func (p *XenditProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	paymentReq := paymentrequest.NewPaymentRequestParameters(paymentrequest.PAYMENTREQUESTCURRENCY_IDR)

	if req.Quantity != nil {
		amount := float64(*req.Quantity * 1000)
		paymentReq.SetAmount(amount)
	}

	if req.Metadata != nil {
		metadata := make(map[string]interface{})
		if metadataMap, ok := req.Metadata.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				metadata[k] = v
			}
		}
		paymentReq.SetMetadata(metadata)
	}

	pr, _, err := p.client.PaymentRequestApi.CreatePaymentRequest(ctx).PaymentRequestParameters(*paymentReq).Execute()
	if err != nil {
		return nil, err
	}

	return &models.Subscription{
		ID:                 pr.GetId(),
		Status:             models.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		ProviderName:       "xendit",
		UpdatedAt:          time.Now(),
	}, nil
}

func (p *XenditProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	now := time.Now()
	return &models.Subscription{
		ID:           subscriptionID,
		Status:       models.SubscriptionStatusCanceled,
		CanceledAt:   &now,
		ProviderName: "xendit",
		UpdatedAt:    time.Now(),
	}, nil
}

func (p *XenditProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	pr, _, err := p.client.PaymentRequestApi.GetPaymentRequestByID(ctx, subscriptionID).Execute()
	if err != nil {
		return nil, err
	}

	status := models.SubscriptionStatusActive
	if pr.GetStatus() == "EXPIRED" {
		status = models.SubscriptionStatusCanceled
	}

	return &models.Subscription{
		ID:                 pr.GetId(),
		Status:             status,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		ProviderName:       "xendit",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

func (p *XenditProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	prs, _, err := p.client.PaymentRequestApi.GetPaymentRequestByID(ctx, customerID).Execute()
	if err != nil {
		return nil, err
	}

	var subscriptions []*models.Subscription
	subscription := &models.Subscription{
		ID:                 prs.GetId(),
		CustomerID:         customerID,
		Status:             models.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		ProviderName:       "xendit",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	subscriptions = append(subscriptions, subscription)

	return subscriptions, nil
}

func (p *XenditProvider) CreatePlan(ctx context.Context, planReq *models.Plan) (*models.Plan, error) {
	return &models.Plan{
		ID:            "plan_" + fmt.Sprintf("%d", time.Now().Unix()),
		Name:          planReq.Name,
		Description:   planReq.Description,
		Amount:        planReq.Amount,
		Currency:      planReq.Currency,
		BillingPeriod: planReq.BillingPeriod,
		PricingType:   planReq.PricingType,
		TrialDays:     planReq.TrialDays,
		Features:      planReq.Features,
		Metadata:      planReq.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *XenditProvider) UpdatePlan(ctx context.Context, planID string, planReq *models.Plan) (*models.Plan, error) {
	return &models.Plan{
		ID:            planID,
		Name:          planReq.Name,
		Description:   planReq.Description,
		Amount:        planReq.Amount,
		Currency:      planReq.Currency,
		BillingPeriod: planReq.BillingPeriod,
		PricingType:   planReq.PricingType,
		TrialDays:     planReq.TrialDays,
		Features:      planReq.Features,
		Metadata:      planReq.Metadata,
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *XenditProvider) DeletePlan(ctx context.Context, planID string) error {
	return nil
}

func (p *XenditProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return &models.Plan{
		ID:            planID,
		Name:          "Default Plan",
		Description:   "Default plan for Xendit",
		Amount:        1000,
		Currency:      "IDR",
		BillingPeriod: models.BillingPeriodMonthly,
		PricingType:   models.PricingTypeFixed,
		TrialDays:     0,
		Features:      []string{},
		Metadata:      nil,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *XenditProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return []*models.Plan{
		{
			ID:            "plan_default",
			Name:          "Default Plan",
			Description:   "Default plan for Xendit",
			Amount:        1000,
			Currency:      "IDR",
			BillingPeriod: models.BillingPeriodMonthly,
			PricingType:   models.PricingTypeFixed,
			TrialDays:     0,
			Features:      []string{},
			Metadata:      nil,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}, nil
}

func (p *XenditProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return &models.Dispute{
		ID:            "disp_" + fmt.Sprintf("%d", time.Now().Unix()),
		CustomerID:    req.CustomerID,
		TransactionID: req.TransactionID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Reason:        req.Reason,
		Status:        models.DisputeStatusOpen,
		Evidence:      req.Evidence,
		DueBy:         req.DueBy,
		Metadata:      req.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *XenditProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return &models.Dispute{
		ID:        disputeID,
		Status:    req.Status,
		Metadata:  req.Metadata,
		UpdatedAt: time.Now(),
	}, nil
}

func (p *XenditProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return &models.Evidence{
		ID:          "evid_" + fmt.Sprintf("%d", time.Now().Unix()),
		DisputeID:   disputeID,
		Type:        req.Type,
		Description: req.Description,
		Files:       req.Files,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (p *XenditProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return &models.Dispute{
		ID:            disputeID,
		CustomerID:    "customer_123",
		TransactionID: "txn_123",
		Amount:        1000,
		Currency:      "IDR",
		Reason:        "fraudulent",
		Status:        models.DisputeStatusOpen,
		Evidence:      make(map[string]interface{}),
		DueBy:         time.Now().AddDate(0, 0, 30),
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (p *XenditProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return []*models.Dispute{
		{
			ID:            "disp_1",
			CustomerID:    customerID,
			TransactionID: "txn_123",
			Amount:        1000,
			Currency:      "IDR",
			Reason:        "fraudulent",
			Status:        models.DisputeStatusOpen,
			Evidence:      make(map[string]interface{}),
			DueBy:         time.Now().AddDate(0, 0, 30),
			Metadata:      make(map[string]interface{}),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}, nil
}

func (p *XenditProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return &models.DisputeStats{
		Total:    1,
		Open:     1,
		Won:      0,
		Lost:     0,
		Canceled: 0,
	}, nil
}

func (p *XenditProvider) ValidateWebhookSignature(payload []byte, signature string) error {
	if p.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if signature != expectedSignature {
		return fmt.Errorf("webhook signature verification failed")
	}

	return nil
}

func (p *XenditProvider) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error) {
	customerReq := *customer.NewCustomerRequest(req.ExternalID)
	customerReq.SetEmail(req.Email)
	
	if req.Phone != "" {
		customerReq.SetMobileNumber(req.Phone)
	}
	
	if req.Metadata != nil {
		customerReq.SetMetadata(req.Metadata)
	}
	
	cust, _, err := p.client.CustomerApi.CreateCustomer(ctx).CustomerRequest(customerReq).Execute()
	if err != nil {
		return "", fmt.Errorf("xendit customer creation failed: %w", err)
	}
	
	return cust.GetId(), nil
}

func (p *XenditProvider) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	updateReq := customer.NewPatchCustomer()
	
	if req.Email != "" {
		updateReq.SetEmail(req.Email)
	}
	
	if req.Phone != "" {
		updateReq.SetMobileNumber(req.Phone)
	}
	
	if req.Metadata != nil {
		updateReq.SetMetadata(req.Metadata)
	}
	
	_, _, err := p.client.CustomerApi.UpdateCustomer(ctx, customerID).PatchCustomer(*updateReq).Execute()
	return err
}

func (p *XenditProvider) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	cust, _, err := p.client.CustomerApi.GetCustomer(ctx, customerID).Execute()
	if err != nil {
		return nil, err
	}
	
	metadata := make(map[string]interface{})
	if custMetadata := cust.GetMetadata(); custMetadata != nil {
		metadata = custMetadata
	}
	
	result := &models.Customer{
		ExternalID: cust.GetReferenceId(),
		Email:      cust.GetEmail(),
		Metadata:   metadata,
		CreatedAt:  cust.GetCreated(),
	}
	
	if cust.MobileNumber.IsSet() {
		if phonePtr := cust.MobileNumber.Get(); phonePtr != nil {
			result.Phone = *phonePtr
		}
	}
	
	return result, nil
}

func (p *XenditProvider) DeleteCustomer(ctx context.Context, customerID string) error {
	return fmt.Errorf("xendit does not support customer deletion via API")
}

func (p *XenditProvider) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}
	
	result := &models.PaymentMethod{
		CustomerID:              req.CustomerID,
		ProviderName:            "xendit",
		ProviderPaymentMethodID: req.PaymentMethodID,
		Type:                    req.Type,
		IsDefault:               req.IsDefault,
		Metadata:                metadata,
		CreatedAt:               time.Now(),
	}
	
	return result, nil
}

func (p *XenditProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	return &models.PaymentMethod{
		ProviderName:            "xendit",
		ProviderPaymentMethodID: paymentMethodID,
		Type:                    "card",
		CreatedAt:               time.Now(),
	}, nil
}

func (p *XenditProvider) ListPaymentMethods(ctx context.Context, customerID string) ([]*models.PaymentMethod, error) {
	return []*models.PaymentMethod{}, nil
}

func (p *XenditProvider) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	return fmt.Errorf("xendit payment methods are automatically attached on creation")
}

func (p *XenditProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return fmt.Errorf("xendit does not support payment method expiration via standard API")
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
