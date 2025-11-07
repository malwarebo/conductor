package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/malwarebo/gopay/providers"
)

func TestWebhookHandler_HandleStripeWebhook_MissingSignature(t *testing.T) {
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", "webhook_secret")
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`{"type": "payment_intent.succeeded"}`)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()

	handler.HandleStripeWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleStripeWebhook() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	body := w.Body.String()
	if body != "Missing Stripe signature\n" {
		t.Errorf("HandleStripeWebhook() body = %q, want %q", body, "Missing Stripe signature\n")
	}
}

func TestWebhookHandler_HandleStripeWebhook_InvalidSignature(t *testing.T) {
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", "webhook_secret")
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`{"type": "payment_intent.succeeded"}`)

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewBuffer(payload))
	req.Header.Set("Stripe-Signature", "invalid_signature")
	w := httptest.NewRecorder()

	handler.HandleStripeWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleStripeWebhook() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWebhookHandler_HandleXenditWebhook_Success(t *testing.T) {
	webhookSecret := "test_webhook_secret"
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", webhookSecret)
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`{
		"event": "payment.succeeded",
		"data": {
			"id": "pr_123",
			"status": "SUCCEEDED"
		}
	}`)

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhooks/xendit", bytes.NewBuffer(payload))
	req.Header.Set("x-callback-token", signature)
	w := httptest.NewRecorder()

	handler.HandleXenditWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleXenditWebhook() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if received, ok := response["received"].(bool); !ok || !received {
		t.Error("HandleXenditWebhook() response[received] should be true")
	}
	if eventType, ok := response["event_type"].(string); !ok || eventType != "payment.succeeded" {
		t.Errorf("HandleXenditWebhook() event_type = %v, want payment.succeeded", eventType)
	}
}

func TestWebhookHandler_HandleXenditWebhook_MissingSignature(t *testing.T) {
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", "webhook_secret")
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`{"event": "payment.succeeded"}`)

	req := httptest.NewRequest("POST", "/webhooks/xendit", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()

	handler.HandleXenditWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleXenditWebhook() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	body := w.Body.String()
	if body != "Missing Xendit signature\n" {
		t.Errorf("HandleXenditWebhook() body = %q, want %q", body, "Missing Xendit signature\n")
	}
}

func TestWebhookHandler_HandleXenditWebhook_InvalidSignature(t *testing.T) {
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", "webhook_secret")
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`{"event": "payment.succeeded"}`)

	req := httptest.NewRequest("POST", "/webhooks/xendit", bytes.NewBuffer(payload))
	req.Header.Set("x-callback-token", "invalid_signature")
	w := httptest.NewRecorder()

	handler.HandleXenditWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleXenditWebhook() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWebhookHandler_HandleXenditWebhook_InvalidPayload(t *testing.T) {
	webhookSecret := "test_webhook_secret"
	stripeProvider := providers.CreateStripeProviderWithWebhook("sk_test_123", "whsec_test123")
	xenditProvider := providers.CreateXenditProviderWithWebhook("xnd_test_123", webhookSecret)
	handler := CreateWebhookHandler(stripeProvider, xenditProvider)

	payload := []byte(`invalid json`)

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhooks/xendit", bytes.NewBuffer(payload))
	req.Header.Set("x-callback-token", signature)
	w := httptest.NewRecorder()

	handler.HandleXenditWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleXenditWebhook() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
