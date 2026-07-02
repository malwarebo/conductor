package api

import (
	"encoding/json"
	"net/http"
)

type MetricsHandler struct {
	snapshot func() map[string]interface{}
}

func CreateMetricsHandler(snapshot func() map[string]interface{}) *MetricsHandler {
	return &MetricsHandler{snapshot: snapshot}
}

func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(h.snapshot())
}
