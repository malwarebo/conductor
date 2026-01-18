package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/malwarebo/conductor/models"
)

const (
	airwallexProdURL = "https://api.airwallex.com"
	airwallexDemoURL = "https://api-demo.airwallex.com"
)

type AirwallexProvider struct {
	clientID      string
	apiKey        string
	webhookSecret string
	baseURL       string
	httpClient    *http.Client
	accessToken   string
	tokenExpiry   time.Time
	tokenMu       sync.RWMutex
}

func CreateAirwallexProvider(clientID, apiKey string, useSandbox bool) *AirwallexProvider {
	baseURL := airwallexProdURL
	if useSandbox {
		baseURL = airwallexDemoURL
	}
	return &AirwallexProvider{
		clientID:   clientID,
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func CreateAirwallexProviderWithWebhook(clientID, apiKey, webhookSecret string, useSandbox bool) *AirwallexProvider {
	p := CreateAirwallexProvider(clientID, apiKey, useSandbox)
	p.webhookSecret = webhookSecret
	return p
}

type awxAuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type awxPaymentIntentRequest struct {
	RequestID       string                 `json:"request_id"`
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	MerchantOrderID string                 `json:"merchant_order_id,omitempty"`
	CustomerID      string                 `json:"customer_id,omitempty"`
	Descriptor      string                 `json:"descriptor,omitempty"`
	ReturnURL       string                 `json:"return_url,omitempty"`
	CaptureMethod   string                 `json:"capture_method,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type awxPaymentIntentResponse struct {
	ID                  string                 `json:"id"`
	RequestID           string                 `json:"request_id"`
	Amount              float64                `json:"amount"`
	Currency            string                 `json:"currency"`
	MerchantOrderID     string                 `json:"merchant_order_id"`
	CustomerID          string                 `json:"customer_id"`
	Status              string                 `json:"status"`
	CapturedAmount      float64                `json:"captured_amount"`
	ClientSecret        string                 `json:"client_secret"`
	Descriptor          string                 `json:"descriptor"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
	NextAction          *awxNextAction         `json:"next_action,omitempty"`
	LatestPaymentAttempt *awxPaymentAttempt    `json:"latest_payment_attempt,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

type awxNextAction struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
}

type awxPaymentAttempt struct {
	ID              string `json:"id"`
	PaymentMethodID string `json:"payment_method_id"`
	Status          string `json:"status"`
}

type awxRefundRequest struct {
	RequestID       string                 `json:"request_id"`
	PaymentIntentID string                 `json:"payment_intent_id"`
	Amount          float64                `json:"amount,omitempty"`
	Reason          string                 `json:"reason,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type awxRefundResponse struct {
	ID              string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	PaymentIntentID string                 `json:"payment_intent_id"`
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	Status          string                 `json:"status"`
	Reason          string                 `json:"reason"`
	CreatedAt       string                 `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type awxCustomerRequest struct {
	RequestID   string                 `json:"request_id"`
	MerchantCustomerID string          `json:"merchant_customer_id"`
	Email       string                 `json:"email,omitempty"`
	FirstName   string                 `json:"first_name,omitempty"`
	LastName    string                 `json:"last_name,omitempty"`
	PhoneNumber string                 `json:"phone_number,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type awxCustomerResponse struct {
	ID                 string                 `json:"id"`
	RequestID          string                 `json:"request_id"`
	MerchantCustomerID string                 `json:"merchant_customer_id"`
	Email              string                 `json:"email"`
	FirstName          string                 `json:"first_name"`
	LastName           string                 `json:"last_name"`
	PhoneNumber        string                 `json:"phone_number"`
	CreatedAt          string                 `json:"created_at"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type awxPayoutRequest struct {
	RequestID      string                 `json:"request_id"`
	SourceID       string                 `json:"source_id,omitempty"`
	BeneficiaryID  string                 `json:"beneficiary_id"`
	PayoutAmount   float64                `json:"payout_amount"`
	PayoutCurrency string                 `json:"payout_currency"`
	PayoutMethod   string                 `json:"payout_method"`
	Reason         string                 `json:"reason,omitempty"`
	Reference      string                 `json:"reference,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type awxPayoutResponse struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	Status         string                 `json:"status"`
	BeneficiaryID  string                 `json:"beneficiary_id"`
	Reference      string                 `json:"reference"`
	CreatedAt      string                 `json:"created_at"`
	FailureReason  string                 `json:"failure_reason,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type awxSubscriptionRequest struct {
	RequestID          string                 `json:"request_id"`
	CustomerID         string                 `json:"customer_id"`
	Currency           string                 `json:"currency"`
	RecurringAmount    float64                `json:"recurring_amount"`
	Period             string                 `json:"period"`
	PeriodCount        int                    `json:"period_count"`
	StartDate          string                 `json:"start_date,omitempty"`
	TrialPeriodDays    int                    `json:"trial_period_days,omitempty"`
	PaymentMethodID    string                 `json:"payment_method_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type awxSubscriptionResponse struct {
	ID              string                 `json:"id"`
	CustomerID      string                 `json:"customer_id"`
	Status          string                 `json:"status"`
	Currency        string                 `json:"currency"`
	RecurringAmount float64                `json:"recurring_amount"`
	Period          string                 `json:"period"`
	PeriodCount     int                    `json:"period_count"`
	CurrentPeriodStart string              `json:"current_period_start"`
	CurrentPeriodEnd   string              `json:"current_period_end"`
	TrialEnd        string                 `json:"trial_end,omitempty"`
	CanceledAt      string                 `json:"canceled_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type awxInvoiceRequest struct {
	RequestID   string                 `json:"request_id"`
	CustomerID  string                 `json:"customer_id"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	Description string                 `json:"description,omitempty"`
	DueDate     string                 `json:"due_date,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type awxInvoiceResponse struct {
	ID          string                 `json:"id"`
	CustomerID  string                 `json:"customer_id"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	Status      string                 `json:"status"`
	Description string                 `json:"description"`
	DueDate     string                 `json:"due_date"`
	PaidAt      string                 `json:"paid_at,omitempty"`
	InvoiceURL  string                 `json:"hosted_invoice_url"`
	CreatedAt   string                 `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type awxBalanceResponse struct {
	AvailableAmount float64 `json:"available_amount"`
	PendingAmount   float64 `json:"pending_amount"`
	TotalAmount     float64 `json:"total_amount"`
	Currency        string  `json:"currency"`
}

type awxListResponse struct {
	Items   json.RawMessage `json:"items"`
	HasMore bool            `json:"has_more"`
}

func (p *AirwallexProvider) authenticate(ctx context.Context) error {
	p.tokenMu.RLock()
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		p.tokenMu.RUnlock()
		return nil
	}
	p.tokenMu.RUnlock()

	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/v1/authentication/login", nil)
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-id", p.clientID)
	req.Header.Set("x-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp awxAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	p.accessToken = authResp.Token
	expiry, err := time.Parse(time.RFC3339, authResp.ExpiresAt)
	if err != nil {
		p.tokenExpiry = time.Now().Add(25 * time.Minute)
	} else {
		p.tokenExpiry = expiry.Add(-1 * time.Minute)
	}

	return nil
}

func (p *AirwallexProvider) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if err := p.authenticate(ctx); err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.tokenMu.RLock()
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	p.tokenMu.RUnlock()
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("airwallex API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (p *AirwallexProvider) Name() string {
	return "airwallex"
}

func (p *AirwallexProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsInvoices:        true,
		SupportsPayouts:         true,
		SupportsPaymentSessions: true,
		Supports3DS:             true,
		SupportsManualCapture:   true,
		SupportsBalance:         true,
		SupportedCurrencies:     []string{"USD", "EUR", "GBP", "AUD", "NZD", "HKD", "SGD", "CNY", "JPY", "CAD", "CHF"},
		SupportedPaymentMethods: []models.PaymentMethodType{models.PMTypeCard, models.PMTypeBankAccount, models.PMTypeEWallet},
	}
}

func (p *AirwallexProvider) Charge(ctx context.Context, req *models.ChargeRequest) (*models.ChargeResponse, error) {
	piReq := awxPaymentIntentRequest{
		RequestID:       fmt.Sprintf("pi_%d", time.Now().UnixNano()),
		Amount:          float64(req.Amount) / 100,
		Currency:        req.Currency,
		MerchantOrderID: req.CustomerID,
		CustomerID:      req.CustomerID,
		Descriptor:      req.Description,
	}

	if req.ReturnURL != "" {
		piReq.ReturnURL = req.ReturnURL
	}

	if req.CaptureMethod == models.CaptureMethodManual || (req.Capture != nil && !*req.Capture) {
		piReq.CaptureMethod = "manual"
	} else {
		piReq.CaptureMethod = "automatic"
	}

	if req.Metadata != nil {
		piReq.Metadata = ConvertStringMapToMetadata(ConvertInterfaceMetadataToStringMap(req.Metadata))
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/create", piReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex charge failed: %w", err)
	}

	var piResp awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &piResp); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent response: %w", err)
	}

	return p.mapPaymentIntentToChargeResponse(&piResp, req), nil
}

func (p *AirwallexProvider) mapPaymentIntentToChargeResponse(pi *awxPaymentIntentResponse, req *models.ChargeRequest) *models.ChargeResponse {
	status := p.mapPaymentIntentStatus(pi.Status)
	created, _ := time.Parse(time.RFC3339, pi.CreatedAt)

	captureMethod := models.CaptureMethodAutomatic
	if req != nil && (req.CaptureMethod == models.CaptureMethodManual || (req.Capture != nil && !*req.Capture)) {
		captureMethod = models.CaptureMethodManual
	}

	resp := &models.ChargeResponse{
		ID:               pi.ID,
		CustomerID:       pi.CustomerID,
		Amount:           int64(pi.Amount * 100),
		Currency:         pi.Currency,
		Status:           status,
		Description:      pi.Descriptor,
		ProviderName:     "airwallex",
		ProviderChargeID: pi.ID,
		CaptureMethod:    captureMethod,
		CapturedAmount:   int64(pi.CapturedAmount * 100),
		ClientSecret:     pi.ClientSecret,
		Metadata:         pi.Metadata,
		CreatedAt:        created,
	}

	if pi.NextAction != nil {
		resp.RequiresAction = true
		resp.NextActionType = pi.NextAction.Type
		resp.NextActionURL = pi.NextAction.URL
	}

	return resp
}

func (p *AirwallexProvider) mapPaymentIntentStatus(status string) models.PaymentStatus {
	switch status {
	case "SUCCEEDED":
		return models.PaymentStatusSuccess
	case "REQUIRES_PAYMENT_METHOD", "REQUIRES_CUSTOMER_ACTION":
		return models.PaymentStatusRequiresAction
	case "REQUIRES_CAPTURE":
		return models.PaymentStatusRequiresCapture
	case "PENDING", "PROCESSING":
		return models.PaymentStatusProcessing
	case "CANCELLED":
		return models.PaymentStatusCanceled
	case "FAILED":
		return models.PaymentStatusFailed
	default:
		return models.PaymentStatusPending
	}
}

func (p *AirwallexProvider) CapturePayment(ctx context.Context, paymentID string, amount int64) error {
	reqBody := map[string]interface{}{
		"request_id": fmt.Sprintf("cap_%d", time.Now().UnixNano()),
	}
	if amount > 0 {
		reqBody["amount"] = float64(amount) / 100
	}

	_, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/"+paymentID+"/capture", reqBody)
	if err != nil {
		return fmt.Errorf("airwallex capture failed: %w", err)
	}
	return nil
}

func (p *AirwallexProvider) VoidPayment(ctx context.Context, paymentID string) error {
	reqBody := map[string]interface{}{
		"request_id":        fmt.Sprintf("void_%d", time.Now().UnixNano()),
		"cancellation_reason": "requested_by_customer",
	}

	_, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/"+paymentID+"/cancel", reqBody)
	if err != nil {
		return fmt.Errorf("airwallex void/cancel failed: %w", err)
	}
	return nil
}

func (p *AirwallexProvider) Create3DSSession(ctx context.Context, paymentID string, returnURL string) (*ThreeDSecureSession, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/pa/payment_intents/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get payment intent failed: %w", err)
	}

	var pi awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &pi); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	session := &ThreeDSecureSession{
		PaymentID:    pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       pi.Status,
	}

	if pi.NextAction != nil {
		session.RedirectURL = pi.NextAction.URL
	}

	return session, nil
}

func (p *AirwallexProvider) Confirm3DSPayment(ctx context.Context, paymentID string) (*models.ChargeResponse, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/pa/payment_intents/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get payment intent failed: %w", err)
	}

	var pi awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &pi); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	return p.mapPaymentIntentToChargeResponse(&pi, nil), nil
}

func (p *AirwallexProvider) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	refundReq := awxRefundRequest{
		RequestID:       fmt.Sprintf("ref_%d", time.Now().UnixNano()),
		PaymentIntentID: req.PaymentID,
		Amount:          float64(req.Amount) / 100,
		Reason:          req.Reason,
	}

	if req.Metadata != nil {
		refundReq.Metadata = ConvertStringMapToMetadata(ConvertInterfaceMetadataToStringMap(req.Metadata))
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/pa/refunds/create", refundReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex refund failed: %w", err)
	}

	var refundResp awxRefundResponse
	if err := json.Unmarshal(respBody, &refundResp); err != nil {
		return nil, fmt.Errorf("failed to parse refund response: %w", err)
	}

	created, _ := time.Parse(time.RFC3339, refundResp.CreatedAt)

	return &models.RefundResponse{
		ID:               refundResp.ID,
		PaymentID:        req.PaymentID,
		Amount:           int64(refundResp.Amount * 100),
		Currency:         refundResp.Currency,
		Status:           refundResp.Status,
		Reason:           refundResp.Reason,
		ProviderName:     "airwallex",
		ProviderRefundID: refundResp.ID,
		Metadata:         refundResp.Metadata,
		CreatedAt:        created,
	}, nil
}

func (p *AirwallexProvider) CreatePaymentSession(ctx context.Context, req *models.CreatePaymentSessionRequest) (*models.PaymentSession, error) {
	piReq := awxPaymentIntentRequest{
		RequestID: fmt.Sprintf("ps_%d", time.Now().UnixNano()),
		Amount:    float64(req.Amount) / 100,
		Currency:  req.Currency,
	}

	if req.CustomerID != "" {
		piReq.CustomerID = req.CustomerID
	}

	if req.Description != "" {
		piReq.Descriptor = req.Description
	}

	if req.CaptureMethod == models.CaptureMethodManual {
		piReq.CaptureMethod = "manual"
	} else {
		piReq.CaptureMethod = "automatic"
	}

	if req.ReturnURL != "" {
		piReq.ReturnURL = req.ReturnURL
	}

	if req.Metadata != nil {
		piReq.Metadata = req.Metadata
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/create", piReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex create payment session failed: %w", err)
	}

	var piResp awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &piResp); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent response: %w", err)
	}

	return p.mapPaymentIntentToSession(&piResp), nil
}

func (p *AirwallexProvider) GetPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/pa/payment_intents/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get payment session failed: %w", err)
	}

	var piResp awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &piResp); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent response: %w", err)
	}

	return p.mapPaymentIntentToSession(&piResp), nil
}

func (p *AirwallexProvider) UpdatePaymentSession(ctx context.Context, sessionID string, req *models.UpdatePaymentSessionRequest) (*models.PaymentSession, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) ConfirmPaymentSession(ctx context.Context, sessionID string, req *models.ConfirmPaymentSessionRequest) (*models.PaymentSession, error) {
	confirmReq := map[string]interface{}{
		"request_id": fmt.Sprintf("confirm_%d", time.Now().UnixNano()),
	}

	if req.PaymentMethodID != "" {
		confirmReq["payment_method_id"] = req.PaymentMethodID
	}

	if req.ReturnURL != "" {
		confirmReq["return_url"] = req.ReturnURL
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/"+sessionID+"/confirm", confirmReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex confirm payment session failed: %w", err)
	}

	var piResp awxPaymentIntentResponse
	if err := json.Unmarshal(respBody, &piResp); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent response: %w", err)
	}

	return p.mapPaymentIntentToSession(&piResp), nil
}

func (p *AirwallexProvider) CapturePaymentSession(ctx context.Context, sessionID string, amount *int64) (*models.PaymentSession, error) {
	captureReq := map[string]interface{}{
		"request_id": fmt.Sprintf("cap_%d", time.Now().UnixNano()),
	}
	if amount != nil {
		captureReq["amount"] = float64(*amount) / 100
	}

	_, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/"+sessionID+"/capture", captureReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex capture payment session failed: %w", err)
	}

	return p.GetPaymentSession(ctx, sessionID)
}

func (p *AirwallexProvider) CancelPaymentSession(ctx context.Context, sessionID string) (*models.PaymentSession, error) {
	cancelReq := map[string]interface{}{
		"request_id":        fmt.Sprintf("cancel_%d", time.Now().UnixNano()),
		"cancellation_reason": "requested_by_customer",
	}

	_, err := p.doRequest(ctx, "POST", "/api/v1/pa/payment_intents/"+sessionID+"/cancel", cancelReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex cancel payment session failed: %w", err)
	}

	return p.GetPaymentSession(ctx, sessionID)
}

func (p *AirwallexProvider) ListPaymentSessions(ctx context.Context, req *models.ListPaymentSessionsRequest) ([]*models.PaymentSession, error) {
	path := "/api/v1/pa/payment_intents"
	if req.CustomerID != "" {
		path += "?customer_id=" + req.CustomerID
	}

	respBody, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex list payment sessions failed: %w", err)
	}

	var listResp awxListResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	var items []awxPaymentIntentResponse
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return nil, fmt.Errorf("failed to parse payment intents: %w", err)
	}

	var sessions []*models.PaymentSession
	for _, pi := range items {
		sessions = append(sessions, p.mapPaymentIntentToSession(&pi))
	}

	return sessions, nil
}

func (p *AirwallexProvider) mapPaymentIntentToSession(pi *awxPaymentIntentResponse) *models.PaymentSession {
	created, _ := time.Parse(time.RFC3339, pi.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, pi.UpdatedAt)

	session := &models.PaymentSession{
		ProviderID:     pi.ID,
		ProviderName:   "airwallex",
		ExternalID:     pi.MerchantOrderID,
		Amount:         int64(pi.Amount * 100),
		Currency:       pi.Currency,
		Status:         p.mapPaymentIntentStatus(pi.Status),
		CustomerID:     pi.CustomerID,
		Description:    pi.Descriptor,
		ClientSecret:   pi.ClientSecret,
		CapturedAmount: int64(pi.CapturedAmount * 100),
		Metadata:       pi.Metadata,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}

	if pi.NextAction != nil {
		session.RequiresAction = true
		session.NextActionType = pi.NextAction.Type
		session.NextActionURL = pi.NextAction.URL
		session.NextAction = &models.NextAction{
			Type:        pi.NextAction.Type,
			RedirectURL: pi.NextAction.URL,
		}
	}

	if pi.LatestPaymentAttempt != nil {
		session.PaymentMethodID = pi.LatestPaymentAttempt.PaymentMethodID
	}

	return session
}

func (p *AirwallexProvider) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (string, error) {
	custReq := awxCustomerRequest{
		RequestID:          fmt.Sprintf("cust_%d", time.Now().UnixNano()),
		MerchantCustomerID: req.ExternalID,
		Email:              req.Email,
	}

	if req.Name != "" {
		custReq.FirstName = req.Name
	}

	if req.Phone != "" {
		custReq.PhoneNumber = req.Phone
	}

	if req.Metadata != nil {
		custReq.Metadata = req.Metadata
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/pa/customers/create", custReq)
	if err != nil {
		return "", fmt.Errorf("airwallex create customer failed: %w", err)
	}

	var custResp awxCustomerResponse
	if err := json.Unmarshal(respBody, &custResp); err != nil {
		return "", fmt.Errorf("failed to parse customer response: %w", err)
	}

	return custResp.ID, nil
}

func (p *AirwallexProvider) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	updateReq := map[string]interface{}{}

	if req.Email != "" {
		updateReq["email"] = req.Email
	}

	if req.Name != "" {
		updateReq["first_name"] = req.Name
	}

	if req.Phone != "" {
		updateReq["phone_number"] = req.Phone
	}

	if req.Metadata != nil {
		updateReq["metadata"] = req.Metadata
	}

	_, err := p.doRequest(ctx, "POST", "/api/v1/pa/customers/"+customerID+"/update", updateReq)
	return err
}

func (p *AirwallexProvider) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/pa/customers/"+customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get customer failed: %w", err)
	}

	var custResp awxCustomerResponse
	if err := json.Unmarshal(respBody, &custResp); err != nil {
		return nil, fmt.Errorf("failed to parse customer response: %w", err)
	}

	created, _ := time.Parse(time.RFC3339, custResp.CreatedAt)

	return &models.Customer{
		ExternalID: custResp.MerchantCustomerID,
		Email:      custResp.Email,
		Name:       custResp.FirstName + " " + custResp.LastName,
		Phone:      custResp.PhoneNumber,
		Metadata:   custResp.Metadata,
		CreatedAt:  created,
	}, nil
}

func (p *AirwallexProvider) DeleteCustomer(ctx context.Context, customerID string) error {
	return ErrNotSupported
}

func (p *AirwallexProvider) CreatePaymentMethod(ctx context.Context, req *models.CreatePaymentMethodRequest) (*models.PaymentMethod, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) ListPaymentMethods(ctx context.Context, customerID string, pmType *models.PaymentMethodType) ([]*models.PaymentMethod, error) {
	return []*models.PaymentMethod{}, nil
}

func (p *AirwallexProvider) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	return ErrNotSupported
}

func (p *AirwallexProvider) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	return ErrNotSupported
}

func (p *AirwallexProvider) ExpirePaymentMethod(ctx context.Context, paymentMethodID string) (*models.PaymentMethod, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	period := "month"
	periodCount := 1

	subReq := awxSubscriptionRequest{
		RequestID:       fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		CustomerID:      req.CustomerID,
		Currency:        "USD",
		RecurringAmount: 0,
		Period:          period,
		PeriodCount:     periodCount,
	}

	if req.TrialDays != nil && *req.TrialDays > 0 {
		subReq.TrialPeriodDays = *req.TrialDays
	}

	if req.Metadata != nil {
		if metadata, ok := req.Metadata.(map[string]interface{}); ok {
			subReq.Metadata = metadata
			if amount, ok := metadata["amount"].(float64); ok {
				subReq.RecurringAmount = amount
			}
			if currency, ok := metadata["currency"].(string); ok {
				subReq.Currency = currency
			}
		}
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/billing/subscriptions/create", subReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex create subscription failed: %w", err)
	}

	var subResp awxSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	return p.mapSubscriptionResponse(&subResp, req.PlanID), nil
}

func (p *AirwallexProvider) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) (*models.Subscription, error) {
	updateReq := map[string]interface{}{}

	if req.Metadata != nil {
		updateReq["metadata"] = req.Metadata
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/billing/subscriptions/"+subscriptionID+"/update", updateReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex update subscription failed: %w", err)
	}

	var subResp awxSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	planID := ""
	if req.PlanID != nil {
		planID = *req.PlanID
	}

	return p.mapSubscriptionResponse(&subResp, planID), nil
}

func (p *AirwallexProvider) CancelSubscription(ctx context.Context, subscriptionID string, req *models.CancelSubscriptionRequest) (*models.Subscription, error) {
	cancelReq := map[string]interface{}{
		"request_id": fmt.Sprintf("cancel_%d", time.Now().UnixNano()),
	}

	if req.CancelAtPeriodEnd {
		cancelReq["cancel_at_period_end"] = true
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/billing/subscriptions/"+subscriptionID+"/cancel", cancelReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex cancel subscription failed: %w", err)
	}

	var subResp awxSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	sub := p.mapSubscriptionResponse(&subResp, "")
	now := time.Now()
	sub.CanceledAt = &now

	return sub, nil
}

func (p *AirwallexProvider) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/billing/subscriptions/"+subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get subscription failed: %w", err)
	}

	var subResp awxSubscriptionResponse
	if err := json.Unmarshal(respBody, &subResp); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	return p.mapSubscriptionResponse(&subResp, ""), nil
}

func (p *AirwallexProvider) ListSubscriptions(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	path := "/api/v1/billing/subscriptions"
	if customerID != "" {
		path += "?customer_id=" + customerID
	}

	respBody, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex list subscriptions failed: %w", err)
	}

	var listResp awxListResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	var items []awxSubscriptionResponse
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions: %w", err)
	}

	var subscriptions []*models.Subscription
	for _, sub := range items {
		subscriptions = append(subscriptions, p.mapSubscriptionResponse(&sub, ""))
	}

	return subscriptions, nil
}

func (p *AirwallexProvider) mapSubscriptionResponse(sub *awxSubscriptionResponse, planID string) *models.Subscription {
	created, _ := time.Parse(time.RFC3339, sub.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, sub.UpdatedAt)
	periodStart, _ := time.Parse(time.RFC3339, sub.CurrentPeriodStart)
	periodEnd, _ := time.Parse(time.RFC3339, sub.CurrentPeriodEnd)

	status := p.mapSubscriptionStatus(sub.Status)

	result := &models.Subscription{
		ID:                 sub.ID,
		CustomerID:         sub.CustomerID,
		PlanID:             planID,
		Status:             status,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Quantity:           1,
		ProviderName:       "airwallex",
		Metadata:           sub.Metadata,
		CreatedAt:          created,
		UpdatedAt:          updated,
	}

	if sub.TrialEnd != "" {
		trialEnd, _ := time.Parse(time.RFC3339, sub.TrialEnd)
		result.TrialEnd = &trialEnd
	}

	if sub.CanceledAt != "" {
		canceledAt, _ := time.Parse(time.RFC3339, sub.CanceledAt)
		result.CanceledAt = &canceledAt
	}

	return result
}

func (p *AirwallexProvider) mapSubscriptionStatus(status string) models.SubscriptionStatus {
	switch status {
	case "ACTIVE":
		return models.SubscriptionStatusActive
	case "CANCELED", "CANCELLED":
		return models.SubscriptionStatusCanceled
	case "TRIALING":
		return models.SubscriptionStatusTrialing
	case "PAST_DUE":
		return models.SubscriptionStatusPastDue
	case "PAUSED":
		return models.SubscriptionStatusPaused
	default:
		return models.SubscriptionStatus("pending")
	}
}

func (p *AirwallexProvider) CreatePlan(ctx context.Context, planReq *models.Plan) (*models.Plan, error) {
	planID := fmt.Sprintf("awx_plan_%d", time.Now().UnixNano())

	result := &models.Plan{
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
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return result, nil
}

func (p *AirwallexProvider) UpdatePlan(ctx context.Context, planID string, planReq *models.Plan) (*models.Plan, error) {
	result := &models.Plan{
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
	}

	return result, nil
}

func (p *AirwallexProvider) DeletePlan(ctx context.Context, planID string) error {
	return nil
}

func (p *AirwallexProvider) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	return nil, fmt.Errorf("airwallex plans are stored locally, use the plan store to retrieve")
}

func (p *AirwallexProvider) ListPlans(ctx context.Context) ([]*models.Plan, error) {
	return nil, fmt.Errorf("airwallex plans are stored locally, use the plan store to list")
}

func (p *AirwallexProvider) CreateInvoice(ctx context.Context, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	invReq := awxInvoiceRequest{
		RequestID:   fmt.Sprintf("inv_%d", time.Now().UnixNano()),
		CustomerID:  req.CustomerID,
		Amount:      float64(req.Amount) / 100,
		Currency:    req.Currency,
		Description: req.Description,
	}

	if req.DueDate != nil {
		invReq.DueDate = req.DueDate.Format(time.RFC3339)
	}

	if req.Metadata != nil {
		invReq.Metadata = req.Metadata
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/billing/invoices/create", invReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex create invoice failed: %w", err)
	}

	var invResp awxInvoiceResponse
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return nil, fmt.Errorf("failed to parse invoice response: %w", err)
	}

	return p.mapInvoiceResponse(&invResp), nil
}

func (p *AirwallexProvider) GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/billing/invoices/"+invoiceID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get invoice failed: %w", err)
	}

	var invResp awxInvoiceResponse
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return nil, fmt.Errorf("failed to parse invoice response: %w", err)
	}

	return p.mapInvoiceResponse(&invResp), nil
}

func (p *AirwallexProvider) ListInvoices(ctx context.Context, req *models.ListInvoicesRequest) ([]*models.Invoice, error) {
	path := "/api/v1/billing/invoices"
	if req.CustomerID != "" {
		path += "?customer_id=" + req.CustomerID
	}

	respBody, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex list invoices failed: %w", err)
	}

	var listResp awxListResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	var items []awxInvoiceResponse
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return nil, fmt.Errorf("failed to parse invoices: %w", err)
	}

	var invoices []*models.Invoice
	for _, inv := range items {
		invoices = append(invoices, p.mapInvoiceResponse(&inv))
	}

	return invoices, nil
}

func (p *AirwallexProvider) CancelInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	cancelReq := map[string]interface{}{
		"request_id": fmt.Sprintf("cancel_%d", time.Now().UnixNano()),
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/billing/invoices/"+invoiceID+"/void", cancelReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex cancel invoice failed: %w", err)
	}

	var invResp awxInvoiceResponse
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return nil, fmt.Errorf("failed to parse invoice response: %w", err)
	}

	return p.mapInvoiceResponse(&invResp), nil
}

func (p *AirwallexProvider) mapInvoiceResponse(inv *awxInvoiceResponse) *models.Invoice {
	created, _ := time.Parse(time.RFC3339, inv.CreatedAt)

	result := &models.Invoice{
		ProviderID:   inv.ID,
		ProviderName: "airwallex",
		CustomerID:   inv.CustomerID,
		Amount:       int64(inv.Amount * 100),
		Currency:     inv.Currency,
		Status:       p.mapInvoiceStatus(inv.Status),
		Description:  inv.Description,
		InvoiceURL:   inv.InvoiceURL,
		Metadata:     inv.Metadata,
		CreatedAt:    created,
		UpdatedAt:    created,
	}

	if inv.DueDate != "" {
		dueDate, _ := time.Parse(time.RFC3339, inv.DueDate)
		result.DueDate = &dueDate
	}

	if inv.PaidAt != "" {
		paidAt, _ := time.Parse(time.RFC3339, inv.PaidAt)
		result.PaidAt = &paidAt
	}

	return result
}

func (p *AirwallexProvider) mapInvoiceStatus(status string) models.InvoiceStatus {
	switch status {
	case "DRAFT":
		return models.InvoiceStatusDraft
	case "PENDING", "OPEN":
		return models.InvoiceStatusPending
	case "PAID":
		return models.InvoiceStatusPaid
	case "VOID", "VOIDED":
		return models.InvoiceStatusVoid
	case "UNCOLLECTIBLE":
		return models.InvoiceStatusCanceled
	default:
		return models.InvoiceStatusPending
	}
}

func (p *AirwallexProvider) CreatePayout(ctx context.Context, req *models.CreatePayoutRequest) (*models.Payout, error) {
	payoutReq := awxPayoutRequest{
		RequestID:      fmt.Sprintf("payout_%d", time.Now().UnixNano()),
		BeneficiaryID:  req.DestinationAccount,
		PayoutAmount:   float64(req.Amount) / 100,
		PayoutCurrency: req.Currency,
		PayoutMethod:   "LOCAL",
		Reference:      req.ReferenceID,
		Reason:         req.Description,
	}

	if req.SourceAccount != "" {
		payoutReq.SourceID = req.SourceAccount
	}

	if req.Metadata != nil {
		payoutReq.Metadata = req.Metadata
	}

	respBody, err := p.doRequest(ctx, "POST", "/api/v1/payouts/create", payoutReq)
	if err != nil {
		return nil, fmt.Errorf("airwallex create payout failed: %w", err)
	}

	var payoutResp awxPayoutResponse
	if err := json.Unmarshal(respBody, &payoutResp); err != nil {
		return nil, fmt.Errorf("failed to parse payout response: %w", err)
	}

	return p.mapPayoutResponse(&payoutResp, req), nil
}

func (p *AirwallexProvider) GetPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	respBody, err := p.doRequest(ctx, "GET", "/api/v1/payouts/"+payoutID, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get payout failed: %w", err)
	}

	var payoutResp awxPayoutResponse
	if err := json.Unmarshal(respBody, &payoutResp); err != nil {
		return nil, fmt.Errorf("failed to parse payout response: %w", err)
	}

	return p.mapPayoutResponse(&payoutResp, nil), nil
}

func (p *AirwallexProvider) ListPayouts(ctx context.Context, req *models.ListPayoutsRequest) ([]*models.Payout, error) {
	path := "/api/v1/payouts"

	respBody, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex list payouts failed: %w", err)
	}

	var listResp awxListResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	var items []awxPayoutResponse
	if err := json.Unmarshal(listResp.Items, &items); err != nil {
		return nil, fmt.Errorf("failed to parse payouts: %w", err)
	}

	var payouts []*models.Payout
	for _, po := range items {
		payouts = append(payouts, p.mapPayoutResponse(&po, nil))
	}

	return payouts, nil
}

func (p *AirwallexProvider) CancelPayout(ctx context.Context, payoutID string) (*models.Payout, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) GetPayoutChannels(ctx context.Context, currency string) ([]*models.PayoutChannel, error) {
	return []*models.PayoutChannel{
		{Code: "LOCAL", Name: "Local Bank Transfer", Category: "bank", Currency: currency},
		{Code: "SWIFT", Name: "SWIFT Transfer", Category: "bank", Currency: currency},
	}, nil
}

func (p *AirwallexProvider) mapPayoutResponse(po *awxPayoutResponse, req *models.CreatePayoutRequest) *models.Payout {
	created, _ := time.Parse(time.RFC3339, po.CreatedAt)

	status := p.mapPayoutStatus(po.Status)

	result := &models.Payout{
		ProviderID:         po.ID,
		ProviderName:       "airwallex",
		ReferenceID:        po.Reference,
		Amount:             int64(po.Amount * 100),
		Currency:           po.Currency,
		Status:             status,
		DestinationType:    models.DestinationBankAccount,
		DestinationAccount: po.BeneficiaryID,
		FailureReason:      po.FailureReason,
		Metadata:           po.Metadata,
		CreatedAt:          created,
		UpdatedAt:          created,
	}

	if req != nil {
		result.Description = req.Description
	}

	return result
}

func (p *AirwallexProvider) mapPayoutStatus(status string) models.PayoutStatus {
	switch status {
	case "SUCCEEDED", "COMPLETED":
		return models.PayoutStatusSucceeded
	case "FAILED":
		return models.PayoutStatusFailed
	case "CANCELLED":
		return models.PayoutStatusCanceled
	case "PENDING", "IN_PROGRESS":
		return models.PayoutStatusProcessing
	default:
		return models.PayoutStatusPending
	}
}

func (p *AirwallexProvider) GetBalance(ctx context.Context, currency string) (*models.Balance, error) {
	path := "/api/v1/balances/current"
	if currency != "" {
		path += "?currency=" + currency
	}

	respBody, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("airwallex get balance failed: %w", err)
	}

	var balResp awxBalanceResponse
	if err := json.Unmarshal(respBody, &balResp); err != nil {
		return nil, fmt.Errorf("failed to parse balance response: %w", err)
	}

	return &models.Balance{
		Available:    int64(balResp.AvailableAmount * 100),
		Pending:      int64(balResp.PendingAmount * 100),
		ProviderName: "airwallex",
		Currency:     balResp.Currency,
	}, nil
}

func (p *AirwallexProvider) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	return nil, fmt.Errorf("airwallex: disputes are initiated by card networks, cannot create via API")
}

func (p *AirwallexProvider) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) AcceptDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) ContestDispute(ctx context.Context, disputeID string, evidence map[string]interface{}) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) SubmitDisputeEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	return nil, ErrNotSupported
}

func (p *AirwallexProvider) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	return []*models.Dispute{}, nil
}

func (p *AirwallexProvider) GetDisputeStats(ctx context.Context) (*models.DisputeStats, error) {
	return &models.DisputeStats{}, nil
}

func (p *AirwallexProvider) ValidateWebhookSignature(payload []byte, signature string) error {
	if p.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("webhook signature verification failed")
	}

	return nil
}

func (p *AirwallexProvider) IsAvailable(ctx context.Context) bool {
	if p.clientID == "" || p.apiKey == "" {
		return false
	}

	err := p.authenticate(ctx)
	return err == nil
}
