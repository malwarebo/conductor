package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/malwarebo/conductor/utils"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

type MetricsResponse struct {
	GoRoutines int                    `json:"goroutines"`
	Memory     Memory                 `json:"memory"`
	Uptime     string                 `json:"uptime"`
	Providers  map[string]interface{} `json:"providers,omitempty"`
	Business   map[string]interface{} `json:"business_metrics,omitempty"`
}

type Memory struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

var startTime = time.Now()

func CreateHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func CreateMetricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)

	businessMetrics := make(map[string]interface{})
	allMetrics := utils.CreateGetAllMetrics()

	paymentCounts := make(map[string]int)
	fraudCounts := make(map[string]int)
	providerCounts := make(map[string]int)

	for _, metric := range allMetrics {
		switch metric.Name {
		case "payments_total":
			key := metric.Labels["status"] + "_" + metric.Labels["currency"]
			paymentCounts[key] = int(metric.Value)
		case "fraud_analysis_total":
			key := metric.Labels["status"]
			fraudCounts[key] = int(metric.Value)
		case "provider_operations_total":
			key := metric.Labels["provider"] + "_" + metric.Labels["status"]
			providerCounts[key] = int(metric.Value)
		}
	}

	businessMetrics["payments"] = paymentCounts
	businessMetrics["fraud_analysis"] = fraudCounts
	businessMetrics["provider_operations"] = providerCounts

	response := MetricsResponse{
		GoRoutines: runtime.NumGoroutine(),
		Memory: Memory{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
		Uptime:   uptime.String(),
		Business: businessMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
