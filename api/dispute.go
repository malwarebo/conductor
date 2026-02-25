package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type DisputeHandler struct {
	disputeService *services.DisputeService
}

func CreateDisputeHandler(disputeService *services.DisputeService) *DisputeHandler {
	return &DisputeHandler{
		disputeService: disputeService,
	}
}

func (h *DisputeHandler) HandleDisputes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch r.Method {
	case http.MethodPost:
		if strings.HasSuffix(path, "/accept") {
			h.handleAcceptDispute(w, r)
		} else if strings.HasSuffix(path, "/contest") {
			h.handleContestDispute(w, r)
		} else if strings.HasSuffix(path, "/evidence") {
			h.handleSubmitEvidence(w, r)
		} else {
			h.handleCreateDispute(w, r)
		}
	case http.MethodGet:
		if strings.HasSuffix(path, "/stats") {
			h.handleGetStats(w, r)
		} else if id := extractDisputeID(path); id != "" {
			h.handleGetDispute(w, r, id)
		} else {
			h.handleListDisputes(w, r)
		}
	case http.MethodPut:
		if id := extractDisputeID(path); id != "" {
			h.handleUpdateDispute(w, r, id)
		} else {
			http.Error(w, "Dispute ID required", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func extractDisputeID(path string) string {
	path = strings.TrimPrefix(path, "/v1/disputes/")
	path = strings.TrimPrefix(path, "/disputes/")
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

func (h *DisputeHandler) handleCreateDispute(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	dispute, err := h.disputeService.CreateDispute(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, dispute)
}

func (h *DisputeHandler) handleUpdateDispute(w http.ResponseWriter, r *http.Request, disputeID string) {
	var req models.UpdateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	dispute, err := h.disputeService.UpdateDispute(r.Context(), disputeID, &req)
	if err != nil {
		if err == services.ErrDisputeNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Dispute not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dispute)
}

func (h *DisputeHandler) handleAcceptDispute(w http.ResponseWriter, r *http.Request) {
	disputeID := extractDisputeID(r.URL.Path)
	if disputeID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Dispute ID required"})
		return
	}

	dispute, err := h.disputeService.AcceptDispute(r.Context(), disputeID)
	if err != nil {
		if err == services.ErrDisputeNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Dispute not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dispute)
}

func (h *DisputeHandler) handleContestDispute(w http.ResponseWriter, r *http.Request) {
	disputeID := extractDisputeID(r.URL.Path)
	if disputeID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Dispute ID required"})
		return
	}

	var evidence map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&evidence); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	dispute, err := h.disputeService.ContestDispute(r.Context(), disputeID, evidence)
	if err != nil {
		if err == services.ErrDisputeNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Dispute not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dispute)
}

func (h *DisputeHandler) handleSubmitEvidence(w http.ResponseWriter, r *http.Request) {
	disputeID := extractDisputeID(r.URL.Path)
	if disputeID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Dispute ID required"})
		return
	}

	var req models.SubmitEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	evidence, err := h.disputeService.SubmitEvidence(r.Context(), disputeID, &req)
	if err != nil {
		if err == services.ErrDisputeNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Dispute not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, evidence)
}

func (h *DisputeHandler) handleGetDispute(w http.ResponseWriter, r *http.Request, disputeID string) {
	dispute, err := h.disputeService.GetDispute(r.Context(), disputeID)
	if err != nil {
		if err == services.ErrDisputeNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Dispute not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dispute)
}

func (h *DisputeHandler) handleListDisputes(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "customer_id query parameter is required"})
		return
	}

	disputes, err := h.disputeService.ListDisputes(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  disputes,
		"total": len(disputes),
	})
}

func (h *DisputeHandler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.disputeService.GetStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
