package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type TenantHandler struct {
	tenantService *services.TenantService
}

func CreateTenantHandler(tenantService *services.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

func (h *TenantHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	tenant, err := h.tenantService.Create(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, models.TenantResponse{Tenant: tenant})
}

func (h *TenantHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tenant, err := h.tenantService.GetByID(r.Context(), id)
	if err != nil {
		if err == services.ErrTenantNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Tenant not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.TenantResponse{Tenant: tenant})
}

func (h *TenantHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req models.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	tenant, err := h.tenantService.Update(r.Context(), id, &req)
	if err != nil {
		if err == services.ErrTenantNotFound {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "Tenant not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.TenantResponse{Tenant: tenant})
}

func (h *TenantHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	activeOnly := true

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}
	if a := r.URL.Query().Get("active_only"); a == "false" {
		activeOnly = false
	}

	tenants, total, err := h.tenantService.List(r.Context(), activeOnly, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tenants": tenants,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *TenantHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.tenantService.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *TenantHandler) HandleDeactivate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.tenantService.Deactivate(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *TenantHandler) HandleRegenerateSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	newSecret, err := h.tenantService.RegenerateAPISecret(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"api_secret": newSecret})
}

