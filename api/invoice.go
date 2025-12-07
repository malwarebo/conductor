package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type InvoiceHandler struct {
	invoiceService *services.InvoiceService
}

func CreateInvoiceHandler(invoiceService *services.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
	}
}

func (h *InvoiceHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	invoice, err := h.invoiceService.CreateInvoice(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, models.InvoiceResponse{Invoice: invoice})
}

func (h *InvoiceHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["id"]

	invoice, err := h.invoiceService.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
		return
	}

	writeJSON(w, http.StatusOK, models.InvoiceResponse{Invoice: invoice})
}

func (h *InvoiceHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	req := &models.ListInvoicesRequest{
		CustomerID: r.URL.Query().Get("customer_id"),
		Status:     r.URL.Query().Get("status"),
		Limit:      20,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			req.Offset = o
		}
	}

	invoices, err := h.invoiceService.ListInvoices(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.InvoiceListResponse{
		Invoices: invoices,
		Total:    len(invoices),
	})
}

func (h *InvoiceHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["id"]

	invoice, err := h.invoiceService.CancelInvoice(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.InvoiceResponse{Invoice: invoice})
}

