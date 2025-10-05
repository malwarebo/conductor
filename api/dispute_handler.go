package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/services"
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
	switch r.Method {
	case http.MethodPost:
		if strings.HasSuffix(r.URL.Path, "/evidence") {
			h.handleSubmitEvidence(w, r)
		} else {
			h.handleCreateDispute(w, r)
		}
	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "/stats") {
			h.handleGetStats(w, r)
		} else if id := strings.TrimPrefix(r.URL.Path, "/disputes/"); id != "" {
			h.handleGetDispute(w, r, id)
		} else {
			h.handleListDisputes(w, r)
		}
	case http.MethodPut:
		if id := strings.TrimPrefix(r.URL.Path, "/disputes/"); id != "" {
			h.handleUpdateDispute(w, r, id)
		} else {
			http.Error(w, "Dispute ID required", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

func (h *DisputeHandler) handleSubmitEvidence(w http.ResponseWriter, r *http.Request) {
	disputeID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/disputes/"), "/evidence")
	if disputeID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Dispute ID required"})
		return
	}

	var req models.SubmitEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata := map[string]interface{}{
		"evidence_type":        req.Type,
		"evidence_description": req.Description,
		"evidence_files":       req.Files,
	}

	updateReq := &models.UpdateDisputeRequest{
		Metadata: metadata,
	}
	dispute, err := h.disputeService.UpdateDispute(r.Context(), disputeID, updateReq)
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

	writeJSON(w, http.StatusOK, disputes)
}

func (h *DisputeHandler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.disputeService.GetStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
