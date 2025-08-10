package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/services"
)

type FraudHandler struct {
	service services.FraudService
}

func NewFraudHandler(service services.FraudService) *FraudHandler {
	return &FraudHandler{
		service: service,
	}
}

func (h *FraudHandler) AnalyzeTransaction(w http.ResponseWriter, r *http.Request) {
	var request models.FraudAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.TransactionID == "" {
		http.Error(w, "transaction_id is required", http.StatusBadRequest)
		return
	}
	if request.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}
	if request.TransactionAmount <= 0 {
		http.Error(w, "transaction_amount must be greater than 0", http.StatusBadRequest)
		return
	}
	if request.BillingCountry == "" {
		http.Error(w, "billing_country is required", http.StatusBadRequest)
		return
	}
	if request.ShippingCountry == "" {
		http.Error(w, "shipping_country is required", http.StatusBadRequest)
		return
	}
	if request.IPAddress == "" {
		http.Error(w, "ip_address is required", http.StatusBadRequest)
		return
	}

	response, err := h.service.AnalyzeTransaction(r.Context(), &request)
	if err != nil {
		log.Printf("Error analyzing transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *FraudHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	if startDateStr == "" || endDateStr == "" {
		http.Error(w, "start_date and end_date query parameters are required", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		http.Error(w, "Invalid start_date format. Use ISO 8601 format (e.g., 2025-08-01T00:00:00Z)", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		http.Error(w, "Invalid end_date format. Use ISO 8601 format (e.g., 2025-08-10T23:59:59Z)", http.StatusBadRequest)
		return
	}

	if endDate.Before(startDate) {
		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
		return
	}

	stats, err := h.service.GetStatsByDateRange(startDate, endDate)
	if err != nil {
		log.Printf("Error getting fraud stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
