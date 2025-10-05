package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/malwarebo/gopay/observability"
	"github.com/malwarebo/gopay/performance"
)

type PerformanceHandler struct {
	benchmarkSuite *performance.BenchmarkSuite
	optimizer      *performance.PerformanceOptimizer
	loadTester     *performance.LoadTester
	metrics        *observability.MetricsCollector
	tracer         *observability.Tracer
	healthChecker  *observability.HealthChecker
}

func CreatePerformanceHandler(
	benchmarkSuite *performance.BenchmarkSuite,
	optimizer *performance.PerformanceOptimizer,
	loadTester *performance.LoadTester,
	metrics *observability.MetricsCollector,
	tracer *observability.Tracer,
	healthChecker *observability.HealthChecker,
) *PerformanceHandler {
	return &PerformanceHandler{
		benchmarkSuite: benchmarkSuite,
		optimizer:      optimizer,
		loadTester:     loadTester,
		metrics:        metrics,
		tracer:         tracer,
		healthChecker:  healthChecker,
	}
}

func (h *PerformanceHandler) HandleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string        `json:"name"`
		Duration    time.Duration `json:"duration"`
		Concurrency int           `json:"concurrency"`
		Operation   string        `json:"operation"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	operation := h.getOperation(req.Operation)
	if operation == nil {
		http.Error(w, "Invalid operation", http.StatusBadRequest)
		return
	}

	var result performance.BenchmarkResult
	if req.Concurrency > 1 {
		result = h.benchmarkSuite.RunConcurrentBenchmark(req.Name, req.Duration, req.Concurrency, operation)
	} else {
		result = h.benchmarkSuite.RunBenchmark(req.Name, req.Duration, operation)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PerformanceHandler) HandleLoadTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Concurrency    int           `json:"concurrency"`
		Duration       time.Duration `json:"duration"`
		RampUpDuration time.Duration `json:"ramp_up_duration"`
		TargetRPS      float64       `json:"target_rps"`
		Endpoint       string        `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config := performance.LoadTestConfig{
		Concurrency:    req.Concurrency,
		Duration:       req.Duration,
		RampUpDuration: req.RampUpDuration,
		TargetRPS:      req.TargetRPS,
	}

	requestFunc := func() (*http.Request, error) {
		return http.NewRequest("GET", req.Endpoint, nil)
	}

	result := h.loadTester.RunLoadTest(r.Context(), config, requestFunc)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *PerformanceHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.metrics.GetAllMetrics()
	summary := h.metrics.GetSummary()

	response := map[string]interface{}{
		"metrics": metrics,
		"summary": summary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PerformanceHandler) HandleTraces(w http.ResponseWriter, r *http.Request) {
	spans := h.tracer.GetSpans()
	summary := h.tracer.GetTraceSummary()

	response := map[string]interface{}{
		"spans":   spans,
		"summary": summary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PerformanceHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	checks := h.healthChecker.RunChecks(r.Context())
	overallStatus := h.healthChecker.GetOverallStatus(checks)
	summary := h.healthChecker.GetHealthSummary(checks)

	response := map[string]interface{}{
		"status":  overallStatus,
		"checks":  checks,
		"summary": summary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PerformanceHandler) HandleOptimization(w http.ResponseWriter, r *http.Request) {
	metrics := h.optimizer.GetMetrics()
	recommendations := h.optimizer.GetRecommendations()
	optimizations := h.optimizer.OptimizeCache(r.Context(), 1000, time.Hour)

	response := map[string]interface{}{
		"metrics":         metrics,
		"recommendations": recommendations,
		"optimizations":   optimizations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PerformanceHandler) getOperation(operation string) func() error {
	switch operation {
	case "payment":
		return func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}
	case "fraud_check":
		return func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}
	case "routing":
		return func() error {
			time.Sleep(20 * time.Millisecond)
			return nil
		}
	default:
		return nil
	}
}
