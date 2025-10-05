package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/services"
)

type SubscriptionHandler struct {
	subscriptionService *services.SubscriptionService
}

func CreateSubscriptionHandler(subscriptionService *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

func (h *SubscriptionHandler) HandlePlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreatePlan(w, r)
	case http.MethodGet:
		if id := strings.TrimPrefix(r.URL.Path, "/plans/"); id != "" {
			h.handleGetPlan(w, r, id)
		} else {
			h.handleListPlans(w, r)
		}
	case http.MethodPut:
		if id := strings.TrimPrefix(r.URL.Path, "/plans/"); id != "" {
			h.handleUpdatePlan(w, r, id)
		} else {
			http.Error(w, "Plan ID required", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if id := strings.TrimPrefix(r.URL.Path, "/plans/"); id != "" {
			h.handleDeletePlan(w, r, id)
		} else {
			http.Error(w, "Plan ID required", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SubscriptionHandler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateSubscription(w, r)
	case http.MethodGet:
		if id := strings.TrimPrefix(r.URL.Path, "/subscriptions/"); id != "" {
			h.handleGetSubscription(w, r, id)
		} else {
			h.handleListSubscriptions(w, r)
		}
	case http.MethodPut:
		if id := strings.TrimPrefix(r.URL.Path, "/subscriptions/"); id != "" {
			h.handleUpdateSubscription(w, r, id)
		} else {
			http.Error(w, "Subscription ID required", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if id := strings.TrimPrefix(r.URL.Path, "/subscriptions/"); id != "" {
			h.handleCancelSubscription(w, r, id)
		} else {
			http.Error(w, "Subscription ID required", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Plan handlers
func (h *SubscriptionHandler) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var plan models.Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	createdPlan, err := h.subscriptionService.CreatePlan(r.Context(), &plan)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, createdPlan)
}

func (h *SubscriptionHandler) handleUpdatePlan(w http.ResponseWriter, r *http.Request, planID string) {
	var plan models.Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	updatedPlan, err := h.subscriptionService.UpdatePlan(r.Context(), planID, &plan)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, updatedPlan)
}

func (h *SubscriptionHandler) handleDeletePlan(w http.ResponseWriter, r *http.Request, planID string) {
	if err := h.subscriptionService.DeletePlan(r.Context(), planID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SubscriptionHandler) handleGetPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, err := h.subscriptionService.GetPlan(r.Context(), planID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, plan)
}

func (h *SubscriptionHandler) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionService.ListPlans(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, plans)
}

// Subscription handlers
func (h *SubscriptionHandler) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	subscription, err := h.subscriptionService.CreateSubscription(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, subscription)
}

func (h *SubscriptionHandler) handleUpdateSubscription(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	var req models.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	subscription, err := h.subscriptionService.UpdateSubscription(r.Context(), subscriptionID, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) handleCancelSubscription(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	var req models.CancelSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	subscription, err := h.subscriptionService.CancelSubscription(r.Context(), subscriptionID, &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) handleGetSubscription(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	subscription, err := h.subscriptionService.GetSubscription(r.Context(), subscriptionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func (h *SubscriptionHandler) handleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "customer_id query parameter is required"})
		return
	}

	subscriptions, err := h.subscriptionService.ListSubscriptions(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, subscriptions)
}
