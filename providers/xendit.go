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
	"github.com/xendit/xendit-go/v6/invoice"
	"github.com/xendit/xendit-go/v6/payment_method"
	paymentrequest "github.com/xendit/xendit-go/v6/payment_request"
	"github.com/xendit/xendit-go/v6/payout"
	"github.com/xendit/xendit-go/v6/refund"
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

func (p *XenditProvider) Name() string {
	return "xendit"
}

func (p *XenditProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsInvoices:        true,
		SupportsPayouts:         true,
		SupportsPaymentSessions: true,
		Supports3DS:             true,
		SupportsManualCapture:   true,
		SupportsBalance:         true,
		SupportedCurrencies:     []string{"IDR", "PHP", "VND", "THB", "MYR", "SGD"},
		SupportedPaymentMethods: []models.PaymentMethodType{models.PMTypeCard, models.PMTypeEWallet, models.PMTypeVirtualAccount, models.PMTypeQRCode, models.PMTypeDirectDebit, models.PMTypeRetail},
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

	captureMethod := models.CaptureMethodAutomatic
	if req.CaptureMethod == models.CaptureMethodManual || (req.Capture != nil && !*req.Capture) {
		captureMethod = models.CaptureMethodManual
		paymentReq.SetCaptureMethod(paymentrequest.PAYMENTREQUESTCAPTUREMETHOD_MANUAL)
	}

	if req.Metadata != nil {
		paymentReq.SetMetadata(req.Metadata)
	}

	pr, _, err := p.client.PaymentRequestApi.CreatePaymentRequest(ctx).PaymentRequestParameters(*paymentReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit payment request creation failed: %w", err)
	}

	status := p.mapPaymentStatus(string(pr.GetStatus()))

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
		CaptureMethod:    captureMethod,
		Metadata:         req.Metadata,
		CreatedAt:        time.Now(),
	}

	if actions := pr.GetActions(); len(actions) > 0 {
		response.RequiresAction = true
		for _, action := range actions {
			if action.GetAction() == "AUTH" {
				response.NextActionType = "redirect_to_url"
				response.NextActionURL = action.GetUrl()
			}
		}
	}

	return response, nil
}

func (p *XenditProvider) mapPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "SUCCEEDED":
		return models.PaymentStatusSuccess
	case "FAILED":
		return models.PaymentStatusFailed
	case "PENDING":
		return models.PaymentStatusPending
	case "AWAITING_CAPTURE":
		return models.PaymentStatusRequiresCapture
	case "REQUIRES_ACTION":
		return models.PaymentStatusRequiresAction
	default:
		return models.PaymentStatusPending
	}
}

func (p *XenditProvider) CapturePayment(ctx context.Context, paymentID string, amount int64) error {
	captureParams := paymentrequest.NewCaptureParameters(float64(amount))
	_, _, err := p.client.PaymentRequestApi.CapturePaymentRequest(ctx, paymentID).CaptureParameters(*captureParams).Execute()
	if err != nil {
		return fmt.Errorf("xendit capture failed: %w", err)
	}
	return nil
}

func (p *XenditProvider) VoidPayment(ctx context.Context, paymentID string) error {
	return fmt.Errorf("xendit does not support voiding payments directly, use refund instead")
}

func (p *XenditProvider) getCurrency(currency string) (paymentrequest.PaymentRequestCurrency, error) {
	switch currency {
	case "IDR":
		return paymentrequest.PAYMENTREQUESTCURRENCY_IDR, nil
	case "PHP":
		return paymentrequest.PAYMENTREQUESTCURRENCY_PHP, nil
	default:
		return paymentrequest.PAYMENTREQUESTCURRENCY_IDR, fmt.Errorf("unsupported currency: %s", currency)
	}
}

func (p *XenditProvider) CreatePaymentSession(ctx context.Context, req *models.CreatePaymentSessionRequest) (*models.PaymentSession, error) {
	currency, err := p.getCurrency(req.Currency)
	if err != nil {
		return nil, err
	}

	paymentReq := paymentrequest.NewPaymentRequestParameters(currency)
	paymentReq.SetAmount(float64(req.Amount))

	if req.ExternalID != "" {
		paymentReq.SetReferenceId(req.ExternalID)
	}

	if req.Description != "" {
		paymentReq.SetDescription(req.Description)
	}

	if req.PaymentMethodID != "" {
		paymentReq.SetPaymentMethodId(req.PaymentMethodID)
	}

	if req.CaptureMethod == models.CaptureMethodManual {
		paymentReq.SetCaptureMethod(paymentrequest.PAYMENTREQUESTCAPTUREMETHOD_MANUAL)
	}

	if req.Metadata != nil {
		paymentReq.SetMetadata(req.Metadata)
	}

	pr, _, err := p.client.PaymentRequestApi.CreatePaymentRequest(ctx).PaymentRequestParameters(*paymentReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit create payment session failed: %w", err)
	}

	return p.mapPaymentSession(pr), nil
}

func (p *XenditProvider) GetPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	pr, _, err := p.client.PaymentRequestApi.GetPaymentRequestByID(ctx, sessionID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get payment session failed: %w", err)
	}

	return p.mapPaymentSession(pr), nil
}

func (p *XenditProvider) UpdatePaymentSession(ctx context.Context, sessionID string, req *models.UpdatePaymentSessionRequest) (*models.PaymentSession, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) ConfirmPaymentSession(ctx context.Context, sessionID string, req *models.ConfirmPaymentSessionRequest) (*models.PaymentSession, error) {
	pr, _, err := p.client.PaymentRequestApi.GetPaymentRequestByID(ctx, sessionID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit confirm payment session failed: %w", err)
	}

	return p.mapPaymentSession(pr), nil
}

func (p *XenditProvider) CapturePaymentSession(ctx context.Context, sessionID string, amount *int64) (*models.PaymentSession, error) {
	captureAmount := float64(0)
	if amount != nil {
		captureAmount = float64(*amount)
	}

	captureParams := paymentrequest.NewCaptureParameters(captureAmount)
	_, _, err := p.client.PaymentRequestApi.CapturePaymentRequest(ctx, sessionID).CaptureParameters(*captureParams).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit capture payment session failed: %w", err)
	}

	return p.GetPaymentSession(ctx, sessionID)
}

func (p *XenditProvider) CancelPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) ListPaymentSessions(ctx context.Context, req *models.ListPaymentSessionsRequest) ([]*models.PaymentSession, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) mapPaymentSession(pr *paymentrequest.PaymentRequest) *models.PaymentSession {
	desc := ""
	if pr.Description.IsSet() && pr.Description.Get() != nil {
		desc = *pr.Description.Get()
	}

	session := &models.PaymentSession{
		ProviderID:    pr.GetId(),
		ProviderName:  "xendit",
		ExternalID:    pr.GetReferenceId(),
		Amount:        int64(pr.GetAmount()),
		Currency:      string(pr.GetCurrency()),
		Status:        p.mapPaymentStatus(string(pr.GetStatus())),
		CaptureMethod: models.CaptureMethod(pr.GetCaptureMethod()),
		Description:   desc,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	pm := pr.GetPaymentMethod()
	if pm.Id != "" {
		session.PaymentMethodID = pm.Id
	}

	if actions := pr.GetActions(); len(actions) > 0 {
		session.RequiresAction = true
		for _, action := range actions {
			if action.GetAction() == "AUTH" {
				session.NextActionType = "redirect_to_url"
				session.NextActionURL = action.GetUrl()
				session.NextAction = &models.NextAction{
					Type:        "redirect",
					RedirectURL: action.GetUrl(),
				}
			}
		}
	}

	return session
}

func (p *XenditProvider) CreateInvoice(ctx context.Context, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	invoiceReq := invoice.NewCreateInvoiceRequest(req.ExternalID, float64(req.Amount))

	if req.Currency != "" {
		invoiceReq.SetCurrency(req.Currency)
	}
	if req.CustomerEmail != "" {
		invoiceReq.SetPayerEmail(req.CustomerEmail)
	}
	if req.Description != "" {
		invoiceReq.SetDescription(req.Description)
	}
	if req.DurationSeconds > 0 {
		invoiceReq.SetInvoiceDuration(fmt.Sprintf("%d", req.DurationSeconds))
	}
	if req.SuccessRedirectURL != "" {
		invoiceReq.SetSuccessRedirectUrl(req.SuccessRedirectURL)
	}
	if req.FailureRedirectURL != "" {
		invoiceReq.SetFailureRedirectUrl(req.FailureRedirectURL)
	}
	invoiceReq.SetShouldSendEmail(req.SendEmail)

	inv, _, err := p.client.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(*invoiceReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit create invoice failed: %w", err)
	}

	return p.mapInvoice(inv), nil
}

func (p *XenditProvider) GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	inv, _, err := p.client.InvoiceApi.GetInvoiceById(ctx, invoiceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get invoice failed: %w", err)
	}

	return p.mapInvoice(inv), nil
}

func (p *XenditProvider) ListInvoices(ctx context.Context, req *models.ListInvoicesRequest) ([]*models.Invoice, error) {
	apiReq := p.client.InvoiceApi.GetInvoices(ctx)
	if req.Limit > 0 {
		apiReq = apiReq.Limit(float32(req.Limit))
	}

	invoices, _, err := apiReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit list invoices failed: %w", err)
	}

	result := make([]*models.Invoice, len(invoices))
	for i, inv := range invoices {
		result[i] = p.mapInvoice(&inv)
	}

	return result, nil
}

func (p *XenditProvider) CancelInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	inv, _, err := p.client.InvoiceApi.ExpireInvoice(ctx, invoiceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit cancel invoice failed: %w", err)
	}

	return p.mapInvoice(inv), nil
}

func (p *XenditProvider) mapInvoice(inv *invoice.Invoice) *models.Invoice {
	result := &models.Invoice{
		ProviderID:         inv.GetId(),
		ProviderName:       "xendit",
		ExternalID:         inv.GetExternalId(),
		Amount:             int64(inv.GetAmount()),
		Currency:           string(inv.GetCurrency()),
		CustomerEmail:      inv.GetPayerEmail(),
		Description:        inv.GetDescription(),
		Status:             p.mapInvoiceStatus(inv.GetStatus()),
		InvoiceURL:         inv.GetInvoiceUrl(),
		SuccessRedirectURL: "",
		FailureRedirectURL: "",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if inv.SuccessRedirectUrl != nil {
		result.SuccessRedirectURL = *inv.SuccessRedirectUrl
	}

	dueDate := inv.GetExpiryDate()
	result.DueDate = &dueDate

	return result
}

func (p *XenditProvider) mapInvoiceStatus(status invoice.InvoiceStatus) models.InvoiceStatus {
	switch status {
	case invoice.INVOICESTATUS_PENDING:
		return models.InvoiceStatusPending
	case invoice.INVOICESTATUS_PAID:
		return models.InvoiceStatusPaid
	case invoice.INVOICESTATUS_EXPIRED:
		return models.InvoiceStatusExpired
	case invoice.INVOICESTATUS_SETTLED:
		return models.InvoiceStatusPaid
	default:
		return models.InvoiceStatusPending
	}
}

func (p *XenditProvider) CreatePayout(ctx context.Context, req *models.CreatePayoutRequest) (*models.Payout, error) {
	payoutReq := payout.NewCreatePayoutRequest(
		req.ReferenceID,
		req.DestinationChannel,
		payout.DigitalPayoutChannelProperties{},
		float32(req.Amount),
		req.Currency,
	)

	if req.Description != "" {
		payoutReq.SetDescription(req.Description)
	}
	if req.Metadata != nil {
		payoutReq.SetMetadata(req.Metadata)
	}

	po, _, err := p.client.PayoutApi.CreatePayout(ctx).CreatePayoutRequest(*payoutReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit create payout failed: %w", err)
	}

	return p.mapPayout(po), nil
}

func (p *XenditProvider) GetPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	po, _, err := p.client.PayoutApi.GetPayoutById(ctx, payoutID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get payout failed: %w", err)
	}

	return p.mapPayout(po), nil
}

func (p *XenditProvider) ListPayouts(ctx context.Context, req *models.ListPayoutsRequest) ([]*models.Payout, error) {
	apiReq := p.client.PayoutApi.GetPayouts(ctx)
	if req.ReferenceID != "" {
		apiReq = apiReq.ReferenceId(req.ReferenceID)
	}
	if req.Limit > 0 {
		apiReq = apiReq.Limit(float32(req.Limit))
	}

	resp, _, err := apiReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit list payouts failed: %w", err)
	}

	data := resp.GetData()
	result := make([]*models.Payout, 0, len(data))
	for i := range data {
		if mapped := p.mapPayout(&data[i]); mapped != nil {
			result = append(result, mapped)
		}
	}

	return result, nil
}

func (p *XenditProvider) CancelPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	po, _, err := p.client.PayoutApi.CancelPayout(ctx, payoutID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit cancel payout failed: %w", err)
	}

	return p.mapPayout(po), nil
}

func (p *XenditProvider) GetPayoutChannels(ctx context.Context, currency string) ([]*models.PayoutChannel, error) {
	apiReq := p.client.PayoutApi.GetPayoutChannels(ctx)
	if currency != "" {
		apiReq = apiReq.Currency(currency)
	}

	channels, _, err := apiReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get payout channels failed: %w", err)
	}

	result := make([]*models.PayoutChannel, len(channels))
	for i, ch := range channels {
		result[i] = &models.PayoutChannel{
			Code:     ch.GetChannelCode(),
			Category: string(ch.GetChannelCategory()),
			Currency: ch.GetCurrency(),
		}
	}

	return result, nil
}

func (p *XenditProvider) mapPayout(po *payout.GetPayouts200ResponseDataInner) *models.Payout {
	if po.Payout == nil {
		return nil
	}
	pyt := po.Payout

	status := models.PayoutStatusPending
	switch pyt.Status {
	case "SUCCEEDED":
		status = models.PayoutStatusSucceeded
	case "FAILED":
		status = models.PayoutStatusFailed
	case "CANCELLED":
		status = models.PayoutStatusCanceled
	case "PENDING", "ACCEPTED":
		status = models.PayoutStatusPending
	}

	result := &models.Payout{
		ProviderID:         pyt.Id,
		ProviderName:       "xendit",
		ReferenceID:        pyt.ReferenceId,
		Amount:             int64(pyt.Amount),
		Currency:           pyt.Currency,
		Status:             status,
		DestinationType:    models.DestinationBankAccount,
		DestinationChannel: pyt.ChannelCode,
		CreatedAt:          pyt.Created,
		UpdatedAt:          pyt.Updated,
	}

	if pyt.Description != nil {
		result.Description = *pyt.Description
	}
	if pyt.FailureCode != nil {
		result.FailureReason = *pyt.FailureCode
	}

	return result
}

func (p *XenditProvider) GetBalance(ctx context.Context, currency string) (*models.Balance, error) {
	apiReq := p.client.BalanceApi.GetBalance(ctx)
	if currency != "" {
		apiReq = apiReq.AccountType(currency)
	}

	bal, _, err := apiReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get balance failed: %w", err)
	}

	return &models.Balance{
		Available:    int64(bal.GetBalance()),
		ProviderName: "xendit",
		Currency:     currency,
	}, nil
}

func (p *XenditProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	refundData := refund.NewCreateRefund()
	refundData.SetInvoiceId(req.PaymentID)
	refundData.SetAmount(float64(req.Amount))
	refundData.SetReason(req.Reason)

	if req.Metadata != nil {
		refundData.SetMetadata(req.Metadata)
	}

	ref, _, err := p.client.RefundApi.CreateRefund(ctx).CreateRefund(*refundData).Execute()
	if err != nil {
		return nil, err
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
		Metadata:         req.Metadata,
		CreatedAt:        time.Now(),
	}, nil
}

func (p *XenditProvider) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	reusability := payment_method.PAYMENTMETHODREUSABILITY_ONE_TIME_USE
	if req.Reusable {
		reusability = payment_method.PAYMENTMETHODREUSABILITY_MULTIPLE_USE
	}

	pmReq := payment_method.NewPaymentMethodParameters(
		payment_method.PaymentMethodType(req.Type),
		reusability,
	)

	if req.CustomerID != "" {
		pmReq.SetCustomerId(req.CustomerID)
	}
	if req.Metadata != nil {
		pmReq.SetMetadata(req.Metadata)
	}

	pm, _, err := p.client.PaymentMethodApi.CreatePaymentMethod(ctx).PaymentMethodParameters(*pmReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit create payment method failed: %w", err)
	}

	return p.mapPaymentMethod(pm), nil
}

func (p *XenditProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	pm, _, err := p.client.PaymentMethodApi.GetPaymentMethodByID(ctx, paymentMethodID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit get payment method failed: %w", err)
	}

	return p.mapPaymentMethod(pm), nil
}

func (p *XenditProvider) ListPaymentMethods(ctx context.Context, customerID string, pmType *models.PaymentMethodType) ([]*models.PaymentMethod, error) {
	apiReq := p.client.PaymentMethodApi.GetAllPaymentMethods(ctx)

	resp, _, err := apiReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit list payment methods failed: %w", err)
	}

	data := resp.GetData()
	var result []*models.PaymentMethod
	for i := range data {
		pm := p.mapPaymentMethod(&data[i])
		if customerID != "" && pm.CustomerID != customerID {
			continue
		}
		if pmType != nil && pm.Type != *pmType {
			continue
		}
		result = append(result, pm)
	}

	return result, nil
}

func (p *XenditProvider) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	return ErrNotSupported
}

func (p *XenditProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return ErrNotSupported
}

func (p *XenditProvider) ExpirePaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	pm, _, err := p.client.PaymentMethodApi.ExpirePaymentMethod(ctx, paymentMethodID).Execute()
	if err != nil {
		return nil, fmt.Errorf("xendit expire payment method failed: %w", err)
	}

	return p.mapPaymentMethod(pm), nil
}

func (p *XenditProvider) mapPaymentMethod(pm *payment_method.PaymentMethod) *models.PaymentMethod {
	result := &models.PaymentMethod{
		ProviderPaymentMethodID: pm.GetId(),
		ProviderName:            "xendit",
		Type:                    models.PaymentMethodType(pm.GetType()),
		Reusable:                pm.GetReusability() == payment_method.PAYMENTMETHODREUSABILITY_MULTIPLE_USE,
		Status:                  string(pm.GetStatus()),
		ChannelCode:             "",
		CreatedAt:               pm.GetCreated(),
		UpdatedAt:               pm.GetUpdated(),
	}

	if pm.CustomerId.IsSet() && pm.CustomerId.Get() != nil {
		result.CustomerID = *pm.CustomerId.Get()
	}

	if pm.Metadata != nil {
		result.Metadata = pm.Metadata
	}

	return result
}

func (p *XenditProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) CreatePlan(ctx context.Context, planReq *models.Plan) (*models.Plan, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) UpdatePlan(ctx context.Context, planID string, planReq *models.Plan) (*models.Plan, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) DeletePlan(ctx context.Context, planID string) error {
	return ErrNotSupported
}

func (p *XenditProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *XenditProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return nil, ErrNotSupported
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
	return ErrNotSupported
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
