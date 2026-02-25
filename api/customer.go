package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type CustomerHandler struct {
	customerService *services.CustomerService
}

func CreateCustomerHandler(customerService *services.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
	}
}

func (h *CustomerHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	customer, err := h.customerService.CreateCustomer(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, customer)
}

func (h *CustomerHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	customerID := vars["id"]

	customer, err := h.customerService.GetCustomer(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Customer not found"})
		return
	}

	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	customerID := vars["id"]

	var req models.UpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if err := h.customerService.UpdateCustomer(r.Context(), customerID, &req); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	customer, err := h.customerService.GetCustomer(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Customer updated but failed to retrieve"})
		return
	}

	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	customerID := vars["id"]

	if err := h.customerService.DeleteCustomer(r.Context(), customerID); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true,
		"id":      customerID,
	})
}
