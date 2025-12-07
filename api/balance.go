package api

import (
	"net/http"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type BalanceHandler struct {
	balanceService *services.BalanceService
}

func CreateBalanceHandler(balanceService *services.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
	}
}

func (h *BalanceHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	currency := r.URL.Query().Get("currency")

	if currency != "" {
		balance, err := h.balanceService.GetBalance(r.Context(), currency)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"balance": balance,
		})
		return
	}

	balances, err := h.balanceService.GetAllBalances(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.BalanceResponse{Balances: balances})
}

