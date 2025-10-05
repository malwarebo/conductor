package observability

import (
	"context"
	"sync"
	"time"
)

type HealthStatus string

const (
	Healthy   HealthStatus = "healthy"
	Degraded  HealthStatus = "degraded"
	Unhealthy HealthStatus = "unhealthy"
	Unknown   HealthStatus = "unknown"
)

type HealthCheck struct {
	Name        string
	Status      HealthStatus
	Message     string
	LastChecked time.Time
	Duration    time.Duration
	Metadata    map[string]interface{}
}

type HealthChecker struct {
	checks map[string]func(context.Context) error
	mu     sync.RWMutex
}

func CreateHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]func(context.Context) error),
	}
}

func (hc *HealthChecker) AddCheck(name string, checkFunc func(context.Context) error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = checkFunc
}

func (hc *HealthChecker) RemoveCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.checks, name)
}

func (hc *HealthChecker) RunChecks(ctx context.Context) map[string]*HealthCheck {
	hc.mu.RLock()
	checks := make(map[string]func(context.Context) error)
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()

	results := make(map[string]*HealthCheck)

	for name, checkFunc := range checks {
		start := time.Now()
		err := checkFunc(ctx)
		duration := time.Since(start)

		status := Healthy
		message := "OK"

		if err != nil {
			status = Unhealthy
			message = err.Error()
		}

		results[name] = &HealthCheck{
			Name:        name,
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    duration,
			Metadata:    make(map[string]interface{}),
		}
	}

	return results
}

func (hc *HealthChecker) GetOverallStatus(checks map[string]*HealthCheck) HealthStatus {
	if len(checks) == 0 {
		return Unknown
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, check := range checks {
		switch check.Status {
		case Unhealthy:
			unhealthyCount++
		case Degraded:
			degradedCount++
		}
	}

	if unhealthyCount > 0 {
		return Unhealthy
	}

	if degradedCount > 0 {
		return Degraded
	}

	return Healthy
}

func (hc *HealthChecker) GetHealthSummary(checks map[string]*HealthCheck) map[string]interface{} {
	overallStatus := hc.GetOverallStatus(checks)

	summary := map[string]interface{}{
		"overall_status": overallStatus,
		"total_checks":   len(checks),
		"healthy":        0,
		"degraded":       0,
		"unhealthy":      0,
		"unknown":        0,
	}

	for _, check := range checks {
		switch check.Status {
		case Healthy:
			summary["healthy"] = summary["healthy"].(int) + 1
		case Degraded:
			summary["degraded"] = summary["degraded"].(int) + 1
		case Unhealthy:
			summary["unhealthy"] = summary["unhealthy"].(int) + 1
		case Unknown:
			summary["unknown"] = summary["unknown"].(int) + 1
		}
	}

	return summary
}
