package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/services"
)

type AuditHandler struct {
	auditService *services.AuditService
}

func CreateAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	filter := models.AuditLogFilter{
		Limit:  100,
		Offset: 0,
	}

	if tenantID := r.Context().Value("tenant_id"); tenantID != nil {
		filter.TenantID = tenantID.(string)
	}

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		filter.UserID = userID
	}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}
	if resourceID := r.URL.Query().Get("resource_id"); resourceID != "" {
		filter.ResourceID = resourceID
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			filter.Limit = parsed
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			filter.Offset = parsed
		}
	}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = &parsed
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = &parsed
		}
	}

	logs, total, err := h.auditService.GetAuditLogs(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"audit_logs": logs,
		"total":      total,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	})
}

func (h *AuditHandler) HandleGetResourceHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceType := vars["resource_type"]
	resourceID := vars["resource_id"]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	logs, err := h.auditService.GetResourceHistory(r.Context(), resourceType, resourceID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"audit_logs":    logs,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

