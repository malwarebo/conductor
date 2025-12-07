package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
	webhookService *services.WebhookService
}

func CreatePaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func CreatePaymentHandlerWithWebhook(paymentService *services.PaymentService, webhookService *services.WebhookService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		webhookService: webhookService,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *PaymentHandler) HandleCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if idempotencyKey := r.Header.Get("Idempotency-Key"); idempotencyKey != "" {
		req.IdempotencyKey = idempotencyKey
	}

	resp, err := h.paymentService.CreateCharge(r.Context(), &req)
	if err != nil {
		if err == services.ErrNoAvailableProvider {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "No payment provider available"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if idempotencyKey := r.Header.Get("Idempotency-Key"); idempotencyKey != "" {
		req.IdempotencyKey = idempotencyKey
	}

	resp, err := h.paymentService.Authorize(r.Context(), &req)
	if err != nil {
		if err == services.ErrNoAvailableProvider {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "No payment provider available"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	paymentID := vars["id"]

	var req models.CaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	req.PaymentID = paymentID

	resp, err := h.paymentService.Capture(r.Context(), &req)
	if err != nil {
		switch err {
		case services.ErrPaymentNotFound:
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment not found"})
		case services.ErrPaymentNotCapturable:
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Payment is not in capturable state"})
		case services.ErrPaymentAlreadyCaptured:
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Payment already captured"})
		case services.ErrInvalidCaptureAmount:
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid capture amount"})
		default:
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleVoid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	paymentID := vars["id"]

	var req models.VoidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}
	req.PaymentID = paymentID

	resp, err := h.paymentService.Void(r.Context(), &req)
	if err != nil {
		if err == services.ErrPaymentNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleConfirm3DS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	paymentID := vars["id"]

	req := &models.Confirm3DSRequest{PaymentID: paymentID}

	resp, err := h.paymentService.Confirm3DS(r.Context(), req)
	if err != nil {
		if err == services.ErrPaymentNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleGetPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["id"]

	payment, err := h.paymentService.GetPayment(r.Context(), paymentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment not found"})
		return
	}

	writeJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) HandleCreatePaymentSession(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePaymentSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	session, err := h.paymentService.CreatePaymentSession(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleGetPaymentSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := h.paymentService.GetPaymentSession(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment session not found"})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleUpdatePaymentSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req models.UpdatePaymentSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	session, err := h.paymentService.UpdatePaymentSession(r.Context(), sessionID, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleConfirmPaymentSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req models.ConfirmPaymentSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	session, err := h.paymentService.ConfirmPaymentSession(r.Context(), sessionID, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleCapturePaymentSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	var req struct {
		Amount *int64 `json:"amount,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	session, err := h.paymentService.CapturePaymentSession(r.Context(), sessionID, req.Amount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleCancelPaymentSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := h.paymentService.CancelPaymentSession(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *PaymentHandler) HandleListPaymentSessions(w http.ResponseWriter, r *http.Request) {
	req := &models.ListPaymentSessionsRequest{
		CustomerID: r.URL.Query().Get("customer_id"),
		Limit:      20,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = l
		}
	}

	sessions, err := h.paymentService.ListPaymentSessions(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_sessions": sessions,
	})
}

func (h *PaymentHandler) HandleRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	resp, err := h.paymentService.CreateRefund(r.Context(), &req)
	if err != nil {
		if err == services.ErrNoAvailableProvider {
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "No payment provider available"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PaymentHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Failed to read request body"})
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	eventID, _ := event["id"].(string)
	eventType, _ := event["type"].(string)

	if h.webhookService != nil {
		if err := h.webhookService.ProcessInboundWebhook(r.Context(), "stripe", eventID, eventType, payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to process webhook"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"received":   true,
		"event_id":   eventID,
		"event_type": eventType,
	})
}

func (h *PaymentHandler) HandleXenditWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Failed to read request body"})
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	eventID, _ := event["id"].(string)
	eventType, _ := event["event"].(string)

	if h.webhookService != nil {
		if err := h.webhookService.ProcessInboundWebhook(r.Context(), "xendit", eventID, eventType, payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to process webhook"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"received":   true,
		"event_id":   eventID,
		"event_type": eventType,
	})
}

func (h *PaymentHandler) HandleRazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Failed to read request body"})
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	eventType, _ := event["event"].(string)
	eventID := ""
	if payloadData, ok := event["payload"].(map[string]interface{}); ok {
		if payment, ok := payloadData["payment"].(map[string]interface{}); ok {
			if entity, ok := payment["entity"].(map[string]interface{}); ok {
				eventID, _ = entity["id"].(string)
			}
		} else if order, ok := payloadData["order"].(map[string]interface{}); ok {
			if entity, ok := order["entity"].(map[string]interface{}); ok {
				eventID, _ = entity["id"].(string)
			}
		} else if subscription, ok := payloadData["subscription"].(map[string]interface{}); ok {
			if entity, ok := subscription["entity"].(map[string]interface{}); ok {
				eventID, _ = entity["id"].(string)
			}
		}
	}

	if h.webhookService != nil {
		if err := h.webhookService.ProcessInboundWebhook(r.Context(), "razorpay", eventID, eventType, payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to process webhook"})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"received":   true,
		"event_id":   eventID,
		"event_type": eventType,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
