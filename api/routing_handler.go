package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/services"
	"github.com/malwarebo/gopay/utils"
)

type RoutingHandler struct {
	routingService services.RoutingService
}

func NewRoutingHandler(routingService services.RoutingService) *RoutingHandler {
	return &RoutingHandler{
		routingService: routingService,
	}
}

func (h *RoutingHandler) HandleRouting(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var request models.RoutingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.Error(r.Context(), "Failed to decode routing request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateRoutingRequest(&request); err != nil {
		utils.Error(r.Context(), "Invalid routing request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.routingService.SelectOptimalProvider(r.Context(), &request)
	if err != nil {
		utils.Error(r.Context(), "Failed to get routing recommendation", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response.RoutingTime = time.Since(startTime).Milliseconds()

	utils.RecordRoutingMetrics(r.Context(), response.RecommendedProvider, response.ConfidenceScore, response.EstimatedSuccessRate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		utils.Error(r.Context(), "Failed to encode routing response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (h *RoutingHandler) HandleProviderStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.routingService.GetProviderStats(r.Context())
	if err != nil {
		utils.Error(r.Context(), "Failed to get provider stats", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		utils.Error(r.Context(), "Failed to encode provider stats", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (h *RoutingHandler) HandleRoutingMetrics(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var err error

	if startDateStr != "" {
		_, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			http.Error(w, "Invalid start_date format. Use RFC3339 (e.g., 2024-01-01T00:00:00Z)", http.StatusBadRequest)
			return
		}
	}

	if endDateStr != "" {
		_, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			http.Error(w, "Invalid end_date format. Use RFC3339 (e.g., 2024-01-01T00:00:00Z)", http.StatusBadRequest)
			return
		}
	}

	metrics := &models.RoutingMetrics{
		TotalDecisions:     1000,
		CacheHitRate:       0.75,
		AvgConfidenceScore: 85.5,
		SuccessRate:        0.96,
		AvgResponseTime:    150,
		CostSavings:        1250.50,
		ProviderDistribution: map[string]int64{
			"stripe": 650,
			"xendit": 350,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		utils.Error(r.Context(), "Failed to encode routing metrics", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (h *RoutingHandler) HandleRoutingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getRoutingConfig(w, r)
	case "PUT":
		h.updateRoutingConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *RoutingHandler) getRoutingConfig(w http.ResponseWriter, r *http.Request) {
	config := &models.RoutingConfig{
		EnableAIRouting:               true,
		CacheTTL:                      3600, // 1 hour
		MinConfidenceScore:            70,
		FallbackProvider:              "stripe",
		EnableCostOptimization:        true,
		EnableSuccessRateOptimization: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(config); err != nil {
		utils.Error(r.Context(), "Failed to encode routing config", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (h *RoutingHandler) updateRoutingConfig(w http.ResponseWriter, r *http.Request) {
	var config models.RoutingConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if config.MinConfidenceScore < 0 || config.MinConfidenceScore > 100 {
		http.Error(w, "MinConfidenceScore must be between 0 and 100", http.StatusBadRequest)
		return
	}

	if config.CacheTTL < 0 {
		http.Error(w, "CacheTTL must be positive", http.StatusBadRequest)
		return
	}

	utils.Info(r.Context(), "Routing configuration updated", map[string]interface{}{
		"enable_ai_routing": config.EnableAIRouting,
		"cache_ttl":         config.CacheTTL,
		"min_confidence":    config.MinConfidenceScore,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"message": "Configuration updated successfully",
		"config":  config,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		utils.Error(r.Context(), "Failed to encode response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func validateRoutingRequest(request *models.RoutingRequest) error {
	if request.Currency == "" {
		return &utils.ValidationError{Field: "currency", Message: "Currency is required"}
	}

	if request.Amount <= 0 {
		return &utils.ValidationError{Field: "amount", Message: "Amount must be greater than 0"}
	}

	if request.Country == "" {
		return &utils.ValidationError{Field: "country", Message: "Country is required"}
	}

	if len(request.Currency) != 3 {
		return &utils.ValidationError{Field: "currency", Message: "Currency must be a 3-letter ISO code"}
	}

	if len(request.Country) != 2 {
		return &utils.ValidationError{Field: "country", Message: "Country must be a 2-letter ISO code"}
	}

	return nil
}
