package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type PayoutHandler struct {
	payoutService *services.PayoutService
}

func CreatePayoutHandler(payoutService *services.PayoutService) *PayoutHandler {
	return &PayoutHandler{
		payoutService: payoutService,
	}
}

func (h *PayoutHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	payout, err := h.payoutService.CreatePayout(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, models.PayoutResponse{Payout: payout})
}

func (h *PayoutHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	payoutID := vars["id"]

	payout, err := h.payoutService.GetPayout(r.Context(), payoutID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Payout not found"})
		return
	}

	writeJSON(w, http.StatusOK, models.PayoutResponse{Payout: payout})
}

func (h *PayoutHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	req := &models.ListPayoutsRequest{
		ReferenceID: r.URL.Query().Get("reference_id"),
		Status:      r.URL.Query().Get("status"),
		Limit:       20,
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

	payouts, err := h.payoutService.ListPayouts(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.PayoutListResponse{
		Payouts: payouts,
		Total:   len(payouts),
	})
}

func (h *PayoutHandler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	payoutID := vars["id"]

	payout, err := h.payoutService.CancelPayout(r.Context(), payoutID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.PayoutResponse{Payout: payout})
}

func (h *PayoutHandler) HandleGetChannels(w http.ResponseWriter, r *http.Request) {
	currency := r.URL.Query().Get("currency")

	channels, err := h.payoutService.GetPayoutChannels(r.Context(), currency)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
	})
}

