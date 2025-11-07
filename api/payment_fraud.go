package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type PaymentWithFraudHandler struct {
	paymentService *services.PaymentService
	fraudService   services.FraudService
}

func CreatePaymentWithFraudHandler(paymentService *services.PaymentService, fraudService services.FraudService) *PaymentWithFraudHandler {
	return &PaymentWithFraudHandler{
		paymentService: paymentService,
		fraudService:   fraudService,
	}
}

type EnhancedChargeRequest struct {
	models.ChargeRequest
	BillingCountry      string `json:"billing_country"`
	ShippingCountry     string `json:"shipping_country"`
	IPAddress           string `json:"ip_address"`
	TransactionVelocity int    `json:"transaction_velocity"`
}

type FraudDeniedResponse struct {
	Error         string `json:"error"`
	FraudReason   string `json:"fraud_reason"`
	TransactionID string `json:"transaction_id"`
}

func (h *PaymentWithFraudHandler) HandleChargeWithFraudCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EnhancedChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.IPAddress == "" {
		req.IPAddress = getClientIP(r)
	}

	transactionID := generateTransactionID()

	// Perform fraud analysis
	fraudRequest := &models.FraudAnalysisRequest{
		TransactionID:       transactionID,
		UserID:              req.CustomerID,
		TransactionAmount:   float64(req.Amount) / 100, // Convert from cents
		BillingCountry:      req.BillingCountry,
		ShippingCountry:     req.ShippingCountry,
		IPAddress:           req.IPAddress,
		TransactionVelocity: req.TransactionVelocity,
	}

	fraudResponse, err := h.fraudService.AnalyzeTransaction(r.Context(), fraudRequest)
	if err != nil {
		log.Printf("Fraud analysis failed: %v", err)
		// In case of fraud service failure, you might want to allow the transaction
		// or implement a more sophisticated fallback strategy
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Fraud analysis failed"})
		return
	}

	// Check fraud decision
	if !fraudResponse.Allow {
		log.Printf("Transaction %s denied due to fraud: %s", transactionID, fraudResponse.Reason)
		writeJSON(w, http.StatusForbidden, FraudDeniedResponse{
			Error:         "Transaction denied due to fraud risk",
			FraudReason:   fraudResponse.Reason,
			TransactionID: transactionID,
		})
		return
	}

	// Proceed with payment if fraud check passes
	log.Printf("Transaction %s passed fraud check, proceeding with payment", transactionID)

	resp, err := h.paymentService.CreateCharge(r.Context(), &req.ChargeRequest)
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

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		return xff
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

func generateTransactionID() string {
	timestamp := fmt.Sprintf("%d", getCurrentTimestamp())
	return "txn_" + timestamp
}

func getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// Note: writeJSON function is already defined in payment_handler.go
// Remove the duplicate definition if importing from a shared utils package

// Example of how to use this in your main.go:
//
// enhancedPaymentHandler := api.NewPaymentWithFraudHandler(paymentService, fraudService)
// apiRouter.HandleFunc("/charges/enhanced", enhancedPaymentHandler.HandleChargeWithFraudCheck).Methods("POST")
