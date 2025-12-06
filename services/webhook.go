package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
)

type WebhookService struct {
	webhookStore   *stores.WebhookStore
	paymentStore   *stores.PaymentRepository
	tenantStore    *stores.TenantStore
	auditStore     *stores.AuditStore
	httpClient     *http.Client
}

func CreateWebhookService(
	webhookStore *stores.WebhookStore,
	paymentStore *stores.PaymentRepository,
	tenantStore *stores.TenantStore,
	auditStore *stores.AuditStore,
) *WebhookService {
	return &WebhookService{
		webhookStore: webhookStore,
		paymentStore: paymentStore,
		tenantStore:  tenantStore,
		auditStore:   auditStore,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *WebhookService) ProcessInboundWebhook(ctx context.Context, provider, eventID, eventType string, payload []byte) error {
	existing, _ := s.webhookStore.GetByEventID(ctx, provider, eventID)
	if existing != nil && existing.Status == models.WebhookEventStatusCompleted {
		return nil
	}

	var payloadJSON models.JSON
	json.Unmarshal(payload, &payloadJSON)

	event := &models.WebhookEvent{
		Provider:  provider,
		EventType: eventType,
		EventID:   eventID,
		Payload:   payloadJSON,
		Status:    models.WebhookEventStatusPending,
	}

	if err := s.webhookStore.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to create webhook event: %w", err)
	}

	return s.processEvent(ctx, event)
}

func (s *WebhookService) processEvent(ctx context.Context, event *models.WebhookEvent) error {
	if err := s.webhookStore.MarkProcessing(ctx, event.ID); err != nil {
		return err
	}

	var err error
	switch event.Provider {
	case "stripe":
		err = s.processStripeEvent(ctx, event)
	case "xendit":
		err = s.processXenditEvent(ctx, event)
	default:
		err = fmt.Errorf("unknown provider: %s", event.Provider)
	}

	if err != nil {
		shouldRetry := event.Attempts < event.MaxAttempts
		s.webhookStore.MarkFailed(ctx, event.ID, err.Error(), shouldRetry)
		return err
	}

	return s.webhookStore.MarkCompleted(ctx, event.ID)
}

func (s *WebhookService) processStripeEvent(ctx context.Context, event *models.WebhookEvent) error {
	payload := map[string]interface{}(event.Payload)

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload structure")
	}

	object, ok := data["object"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid object in payload")
	}

	switch event.EventType {
	case "payment_intent.succeeded":
		return s.handlePaymentSucceeded(ctx, object)
	case "payment_intent.payment_failed":
		return s.handlePaymentFailed(ctx, object)
	case "payment_intent.requires_action":
		return s.handlePaymentRequiresAction(ctx, object)
	case "payment_intent.canceled":
		return s.handlePaymentCanceled(ctx, object)
	case "payment_intent.amount_capturable_updated":
		return s.handlePaymentCapturable(ctx, object)
	case "charge.refunded":
		return s.handleChargeRefunded(ctx, object)
	case "charge.dispute.created":
		return s.handleDisputeCreated(ctx, object)
	}

	return nil
}

func (s *WebhookService) processXenditEvent(ctx context.Context, event *models.WebhookEvent) error {
	payload := map[string]interface{}(event.Payload)

	switch event.EventType {
	case "payment.succeeded", "capture.succeeded":
		return s.handleXenditPaymentSucceeded(ctx, payload)
	case "payment.failed":
		return s.handleXenditPaymentFailed(ctx, payload)
	case "payment.pending":
		return s.handleXenditPaymentPending(ctx, payload)
	case "refund.succeeded":
		return s.handleXenditRefundSucceeded(ctx, payload)
	}

	return nil
}

func (s *WebhookService) handlePaymentSucceeded(ctx context.Context, object map[string]interface{}) error {
	paymentID, ok := object["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusSuccess
	if amount, ok := object["amount_received"].(float64); ok {
		payment.CapturedAmount = int64(amount)
	}
	payment.RequiresAction = false

	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handlePaymentFailed(ctx context.Context, object map[string]interface{}) error {
	paymentID, ok := object["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusFailed
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handlePaymentRequiresAction(ctx context.Context, object map[string]interface{}) error {
	paymentID, ok := object["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusRequiresAction
	payment.RequiresAction = true

	if nextAction, ok := object["next_action"].(map[string]interface{}); ok {
		if actionType, ok := nextAction["type"].(string); ok {
			payment.NextActionType = actionType
		}
		if redirectToURL, ok := nextAction["redirect_to_url"].(map[string]interface{}); ok {
			if url, ok := redirectToURL["url"].(string); ok {
				payment.NextActionURL = url
			}
		}
	}

	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handlePaymentCanceled(ctx context.Context, object map[string]interface{}) error {
	paymentID, ok := object["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusCanceled
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handlePaymentCapturable(ctx context.Context, object map[string]interface{}) error {
	paymentID, ok := object["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusRequiresCapture
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleChargeRefunded(ctx context.Context, object map[string]interface{}) error {
	paymentIntentID, ok := object["payment_intent"].(string)
	if !ok {
		return nil
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentIntentID)
	if err != nil {
		return nil
	}

	amountRefunded, _ := object["amount_refunded"].(float64)
	if int64(amountRefunded) >= payment.Amount {
		payment.Status = models.PaymentStatusRefunded
	} else {
		payment.Status = models.PaymentStatusPartiallyRefunded
	}

	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleDisputeCreated(ctx context.Context, object map[string]interface{}) error {
	paymentIntentID, ok := object["payment_intent"].(string)
	if !ok {
		return nil
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentIntentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusDisputed
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleXenditPaymentSucceeded(ctx context.Context, payload map[string]interface{}) error {
	paymentID, ok := payload["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusSuccess
	if amount, ok := payload["capture_amount"].(float64); ok {
		payment.CapturedAmount = int64(amount)
	}

	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleXenditPaymentFailed(ctx context.Context, payload map[string]interface{}) error {
	paymentID, ok := payload["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusFailed
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleXenditPaymentPending(ctx context.Context, payload map[string]interface{}) error {
	paymentID, ok := payload["id"].(string)
	if !ok {
		return fmt.Errorf("missing payment id")
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusProcessing
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) handleXenditRefundSucceeded(ctx context.Context, payload map[string]interface{}) error {
	paymentID, ok := payload["payment_id"].(string)
	if !ok {
		return nil
	}

	payment, err := s.paymentStore.GetByProviderChargeID(ctx, paymentID)
	if err != nil {
		return nil
	}

	payment.Status = models.PaymentStatusRefunded
	return s.paymentStore.Update(ctx, payment)
}

func (s *WebhookService) SendOutboundWebhook(ctx context.Context, tenantID, eventType string, data map[string]interface{}) error {
	tenant, err := s.tenantStore.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.WebhookURL == "" {
		return nil
	}

	payload := &models.OutboundWebhook{
		ID:        generateID(),
		TenantID:  tenantID,
		EventType: eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	signature := s.signPayload(payloadBytes, tenant.WebhookSecret)
	payload.Signature = signature

	payloadBytes, _ = json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", tenant.WebhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-ID", payload.ID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook delivery failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *WebhookService) ProcessPendingWebhooks(ctx context.Context, batchSize int) error {
	events, err := s.webhookStore.GetPendingEvents(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, event := range events {
		s.processEvent(ctx, event)
	}

	return nil
}

func (s *WebhookService) signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = hex.DecodeString(fmt.Sprintf("%032x", time.Now().UnixNano()))
	return fmt.Sprintf("evt_%x", bytes)
}

