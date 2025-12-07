package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type PaymentMethodHandler struct {
	paymentMethodService *services.PaymentMethodService
}

func CreatePaymentMethodHandler(paymentMethodService *services.PaymentMethodService) *PaymentMethodHandler {
	return &PaymentMethodHandler{
		paymentMethodService: paymentMethodService,
	}
}

func (h *PaymentMethodHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	pm, err := h.paymentMethodService.CreatePaymentMethod(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, models.PaymentMethodResponse{PaymentMethod: pm})
}

func (h *PaymentMethodHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentMethodID := vars["id"]

	pm, err := h.paymentMethodService.GetPaymentMethod(r.Context(), paymentMethodID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payment method not found"})
		return
	}

	writeJSON(w, http.StatusOK, models.PaymentMethodResponse{PaymentMethod: pm})
}

func (h *PaymentMethodHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")

	var pmType *models.PaymentMethodType
	if t := r.URL.Query().Get("type"); t != "" {
		pt := models.PaymentMethodType(t)
		pmType = &pt
	}

	methods, err := h.paymentMethodService.ListPaymentMethods(r.Context(), customerID, pmType)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.PaymentMethodListResponse{
		PaymentMethods: methods,
		Total:          len(methods),
	})
}

func (h *PaymentMethodHandler) HandleAttach(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentMethodID := vars["id"]

	var req struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if err := h.paymentMethodService.AttachPaymentMethod(r.Context(), paymentMethodID, req.CustomerID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	pm, _ := h.paymentMethodService.GetPaymentMethod(r.Context(), paymentMethodID)
	writeJSON(w, http.StatusOK, models.PaymentMethodResponse{PaymentMethod: pm})
}

func (h *PaymentMethodHandler) HandleDetach(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentMethodID := vars["id"]

	if err := h.paymentMethodService.DetachPaymentMethod(r.Context(), paymentMethodID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"detached": true,
		"id":       paymentMethodID,
	})
}

func (h *PaymentMethodHandler) HandleExpire(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentMethodID := vars["id"]

	_, _ = io.ReadAll(r.Body)

	pm, err := h.paymentMethodService.ExpirePaymentMethod(r.Context(), paymentMethodID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.PaymentMethodResponse{PaymentMethod: pm})
}

