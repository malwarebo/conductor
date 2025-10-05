package observability

import (
	"context"
	"sync"
	"time"
)

type DashboardMetrics struct {
	Timestamp      time.Time              `json:"timestamp"`
	SystemHealth   string                 `json:"system_health"`
	Performance    map[string]interface{} `json:"performance"`
	Business       map[string]interface{} `json:"business"`
	Infrastructure map[string]interface{} `json:"infrastructure"`
}

type Dashboard struct {
	metrics       *MetricsCollector
	healthChecker *HealthChecker
	tracer        *Tracer
	mu            sync.RWMutex
	lastUpdate    time.Time
}

func CreateDashboard(metrics *MetricsCollector, healthChecker *HealthChecker, tracer *Tracer) *Dashboard {
	return &Dashboard{
		metrics:       metrics,
		healthChecker: healthChecker,
		tracer:        tracer,
		lastUpdate:    time.Now(),
	}
}

func (d *Dashboard) GetMetrics(ctx context.Context) *DashboardMetrics {
	d.mu.Lock()
	defer d.mu.Unlock()

	healthChecks := d.healthChecker.RunChecks(ctx)
	overallHealth := d.healthChecker.GetOverallStatus(healthChecks)

	performance := map[string]interface{}{
		"response_time": d.getResponseTimeMetrics(),
		"throughput":    d.getThroughputMetrics(),
		"error_rate":    d.getErrorRateMetrics(),
		"cache_hits":    d.getCacheHitMetrics(),
	}

	business := map[string]interface{}{
		"payments":      d.getPaymentMetrics(),
		"fraud_checks":  d.getFraudMetrics(),
		"routing":       d.getRoutingMetrics(),
		"subscriptions": d.getSubscriptionMetrics(),
	}

	infrastructure := map[string]interface{}{
		"database":   d.getDatabaseMetrics(),
		"redis":      d.getRedisMetrics(),
		"memory":     d.getMemoryMetrics(),
		"goroutines": d.getGoroutineMetrics(),
	}

	return &DashboardMetrics{
		Timestamp:      time.Now(),
		SystemHealth:   string(overallHealth),
		Performance:    performance,
		Business:       business,
		Infrastructure: infrastructure,
	}
}

func (d *Dashboard) getResponseTimeMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Histogram)

	avgResponseTime := 0.0
	count := 0

	for _, metric := range metrics {
		if metric.Name == "response_time" {
			avgResponseTime += metric.Value
			count++
		}
	}

	if count > 0 {
		avgResponseTime = avgResponseTime / float64(count)
	}

	return map[string]interface{}{
		"average": avgResponseTime,
		"count":   count,
	}
}

func (d *Dashboard) getThroughputMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	totalRequests := 0.0
	for _, metric := range metrics {
		if metric.Name == "requests_total" {
			totalRequests += metric.Value
		}
	}

	return map[string]interface{}{
		"total_requests":      totalRequests,
		"requests_per_second": totalRequests / time.Since(d.lastUpdate).Seconds(),
	}
}

func (d *Dashboard) getErrorRateMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	totalRequests := 0.0
	totalErrors := 0.0

	for _, metric := range metrics {
		if metric.Name == "requests_total" {
			totalRequests += metric.Value
		}
		if metric.Name == "errors_total" {
			totalErrors += metric.Value
		}
	}

	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = totalErrors / totalRequests
	}

	return map[string]interface{}{
		"error_rate":     errorRate,
		"total_errors":   totalErrors,
		"total_requests": totalRequests,
	}
}

func (d *Dashboard) getCacheHitMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Gauge)

	cacheHits := 0.0
	cacheMisses := 0.0

	for _, metric := range metrics {
		if metric.Name == "cache_hits" {
			cacheHits = metric.Value
		}
		if metric.Name == "cache_misses" {
			cacheMisses = metric.Value
		}
	}

	hitRate := 0.0
	totalCache := cacheHits + cacheMisses
	if totalCache > 0 {
		hitRate = cacheHits / totalCache
	}

	return map[string]interface{}{
		"hit_rate":     hitRate,
		"cache_hits":   cacheHits,
		"cache_misses": cacheMisses,
	}
}

func (d *Dashboard) getPaymentMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	payments := 0.0
	refunds := 0.0

	for _, metric := range metrics {
		if metric.Name == "payments_total" {
			payments = metric.Value
		}
		if metric.Name == "refunds_total" {
			refunds = metric.Value
		}
	}

	return map[string]interface{}{
		"total_payments": payments,
		"total_refunds":  refunds,
		"refund_rate":    refunds / payments,
	}
}

func (d *Dashboard) getFraudMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	fraudChecks := 0.0
	fraudDetected := 0.0

	for _, metric := range metrics {
		if metric.Name == "fraud_checks_total" {
			fraudChecks = metric.Value
		}
		if metric.Name == "fraud_detected_total" {
			fraudDetected = metric.Value
		}
	}

	detectionRate := 0.0
	if fraudChecks > 0 {
		detectionRate = fraudDetected / fraudChecks
	}

	return map[string]interface{}{
		"total_checks":   fraudChecks,
		"fraud_detected": fraudDetected,
		"detection_rate": detectionRate,
	}
}

func (d *Dashboard) getRoutingMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	routingDecisions := 0.0
	stripeSelected := 0.0
	xenditSelected := 0.0

	for _, metric := range metrics {
		if metric.Name == "routing_decisions_total" {
			routingDecisions = metric.Value
		}
		if metric.Name == "stripe_selected_total" {
			stripeSelected = metric.Value
		}
		if metric.Name == "xendit_selected_total" {
			xenditSelected = metric.Value
		}
	}

	stripeRate := 0.0
	xenditRate := 0.0
	if routingDecisions > 0 {
		stripeRate = stripeSelected / routingDecisions
		xenditRate = xenditSelected / routingDecisions
	}

	return map[string]interface{}{
		"total_decisions": routingDecisions,
		"stripe_rate":     stripeRate,
		"xendit_rate":     xenditRate,
	}
}

func (d *Dashboard) getSubscriptionMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Counter)

	subscriptions := 0.0
	activeSubscriptions := 0.0

	for _, metric := range metrics {
		if metric.Name == "subscriptions_total" {
			subscriptions = metric.Value
		}
		if metric.Name == "active_subscriptions_total" {
			activeSubscriptions = metric.Value
		}
	}

	return map[string]interface{}{
		"total_subscriptions":  subscriptions,
		"active_subscriptions": activeSubscriptions,
		"activation_rate":      activeSubscriptions / subscriptions,
	}
}

func (d *Dashboard) getDatabaseMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Gauge)

	connections := 0.0
	queryTime := 0.0

	for _, metric := range metrics {
		if metric.Name == "db_connections" {
			connections = metric.Value
		}
		if metric.Name == "db_query_time" {
			queryTime = metric.Value
		}
	}

	return map[string]interface{}{
		"active_connections": connections,
		"avg_query_time":     queryTime,
	}
}

func (d *Dashboard) getRedisMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Gauge)

	memoryUsage := 0.0
	operations := 0.0

	for _, metric := range metrics {
		if metric.Name == "redis_memory" {
			memoryUsage = metric.Value
		}
		if metric.Name == "redis_operations" {
			operations = metric.Value
		}
	}

	return map[string]interface{}{
		"memory_usage": memoryUsage,
		"operations":   operations,
	}
}

func (d *Dashboard) getMemoryMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Gauge)

	alloc := 0.0
	sys := 0.0

	for _, metric := range metrics {
		if metric.Name == "memory_alloc" {
			alloc = metric.Value
		}
		if metric.Name == "memory_sys" {
			sys = metric.Value
		}
	}

	return map[string]interface{}{
		"allocated": alloc,
		"system":    sys,
	}
}

func (d *Dashboard) getGoroutineMetrics() map[string]interface{} {
	metrics := d.metrics.GetMetricsByType(Gauge)

	goroutines := 0.0

	for _, metric := range metrics {
		if metric.Name == "goroutines" {
			goroutines = metric.Value
		}
	}

	return map[string]interface{}{
		"count": goroutines,
	}
}
