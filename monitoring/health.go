package monitoring

import (
	"context"
	"sync"
	"time"
)

type HealthStatus string

const (
	Healthy   HealthStatus = "healthy"
	Unhealthy HealthStatus = "unhealthy"
	Degraded  HealthStatus = "degraded"
)

type HealthCheck struct {
	Name        string        `json:"name"`
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message"`
	Duration    time.Duration `json:"duration"`
	LastChecked time.Time     `json:"last_checked"`
	Error       string        `json:"error,omitempty"`
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

func (hc *HealthChecker) AddCheck(name string, check func(context.Context) error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

func (hc *HealthChecker) RunCheck(ctx context.Context, name string) HealthCheck {
	hc.mu.RLock()
	check, exists := hc.checks[name]
	hc.mu.RUnlock()

	if !exists {
		return HealthCheck{
			Name:        name,
			Status:      Unhealthy,
			Message:     "Check not found",
			LastChecked: time.Now(),
			Error:       "Check not found",
		}
	}

	start := time.Now()
	err := check(ctx)
	duration := time.Since(start)

	status := Healthy
	message := "OK"
	errorMsg := ""

	if err != nil {
		status = Unhealthy
		message = "Failed"
		errorMsg = err.Error()
	}

	return HealthCheck{
		Name:        name,
		Status:      status,
		Message:     message,
		Duration:    duration,
		LastChecked: time.Now(),
		Error:       errorMsg,
	}
}

func (hc *HealthChecker) RunAllChecks(ctx context.Context) map[string]HealthCheck {
	hc.mu.RLock()
	checkNames := make([]string, 0, len(hc.checks))
	for name := range hc.checks {
		checkNames = append(checkNames, name)
	}
	hc.mu.RUnlock()

	results := make(map[string]HealthCheck)
	for _, name := range checkNames {
		results[name] = hc.RunCheck(ctx, name)
	}

	return results
}

func (hc *HealthChecker) GetOverallStatus(checks map[string]HealthCheck) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	for _, check := range checks {
		switch check.Status {
		case Unhealthy:
			hasUnhealthy = true
		case Degraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return Unhealthy
	}
	if hasDegraded {
		return Degraded
	}
	return Healthy
}

type SystemHealth struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
	Uptime    time.Duration          `json:"uptime"`
	Version   string                 `json:"version"`
}

type HealthService struct {
	checker   *HealthChecker
	startTime time.Time
	version   string
}

func CreateHealthService(version string) *HealthService {
	return &HealthService{
		checker:   CreateHealthChecker(),
		startTime: time.Now(),
		version:   version,
	}
}

func (hs *HealthService) AddCheck(name string, check func(context.Context) error) {
	hs.checker.AddCheck(name, check)
}

func (hs *HealthService) GetHealth(ctx context.Context) SystemHealth {
	checks := hs.checker.RunAllChecks(ctx)
	status := hs.checker.GetOverallStatus(checks)

	return SystemHealth{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
		Uptime:    time.Since(hs.startTime),
		Version:   hs.version,
	}
}

func (hs *HealthService) IsHealthy(ctx context.Context) bool {
	health := hs.GetHealth(ctx)
	return health.Status == Healthy
}

func (hs *HealthService) GetStatus() HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := hs.GetHealth(ctx)
	return health.Status
}
