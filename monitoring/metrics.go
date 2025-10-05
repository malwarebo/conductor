package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"
)

type MetricType int

const (
	Counter MetricType = iota
	Gauge
	Histogram
	Summary
)

type Metric struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type MetricsCollector struct {
	metrics map[string]*Metric
	mu      sync.RWMutex
}

type SystemMetrics struct {
	CPUUsage     float64
	MemoryUsage  float64
	Goroutines   int
	HeapSize     uint64
	GCs          uint32
	Uptime       time.Duration
	RequestRate  float64
	ErrorRate    float64
	ResponseTime float64
}

var globalMetrics = &MetricsCollector{
	metrics: make(map[string]*Metric),
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
	}
}

func (mc *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value++
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:      name,
			Type:      Counter,
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      Gauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value = (metric.Value + value) / 2
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:      name,
			Type:      Histogram,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) *Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.getKey(name, labels)
	return mc.metrics[key]
}

func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

func (mc *MetricsCollector) getKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += "_" + k + ":" + v
	}
	return key
}

func IncrementCounter(name string, labels map[string]string) {
	globalMetrics.IncrementCounter(name, labels)
}

func SetGauge(name string, value float64, labels map[string]string) {
	globalMetrics.SetGauge(name, value, labels)
}

func RecordHistogram(name string, value float64, labels map[string]string) {
	globalMetrics.RecordHistogram(name, value, labels)
}

func GetMetric(name string, labels map[string]string) *Metric {
	return globalMetrics.GetMetric(name, labels)
}

func GetAllMetrics() map[string]*Metric {
	return globalMetrics.GetAllMetrics()
}

func GetSystemMetrics() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := make(map[string]float64)
	metrics["cpu_usage"] = getCPUUsage()
	metrics["memory_usage"] = float64(m.Alloc)
	metrics["goroutines"] = float64(runtime.NumGoroutine())
	metrics["heap_size"] = float64(m.HeapSys)
	metrics["gc_count"] = float64(m.NumGC)
	metrics["gc_pause"] = float64(m.PauseTotalNs)
	metrics["uptime"] = float64(time.Since(startTime).Seconds())

	return metrics
}

func getCPUUsage() float64 {
	return 0.0
}

var startTime = time.Now()

func RecordPaymentMetrics(ctx context.Context, amount int64, currency, provider, status string) {
	IncrementCounter("payments_total", map[string]string{
		"currency": currency,
		"provider": provider,
		"status":   status,
	})

	SetGauge("payment_amount", float64(amount), map[string]string{
		"currency": currency,
		"provider": provider,
	})

	RecordHistogram("payment_processing_time", float64(time.Since(time.Now()).Milliseconds()), map[string]string{
		"currency": currency,
		"provider": provider,
	})
}

func RecordFraudMetrics(ctx context.Context, isFraudulent bool, fraudScore int, reason string) {
	status := "allowed"
	if isFraudulent {
		status = "blocked"
	}

	IncrementCounter("fraud_analysis_total", map[string]string{
		"status": status,
		"reason": reason,
	})

	SetGauge("fraud_score", float64(fraudScore), map[string]string{
		"status": status,
	})
}

func RecordProviderMetrics(ctx context.Context, provider, operation, status string, duration time.Duration) {
	IncrementCounter("provider_operations_total", map[string]string{
		"provider":  provider,
		"operation": operation,
		"status":    status,
	})

	RecordHistogram("provider_response_time", float64(duration.Milliseconds()), map[string]string{
		"provider":  provider,
		"operation": operation,
	})
}

func RecordDatabaseMetrics(ctx context.Context, operation, status string, duration time.Duration) {
	IncrementCounter("database_operations_total", map[string]string{
		"operation": operation,
		"status":    status,
	})

	RecordHistogram("database_response_time", float64(duration.Milliseconds()), map[string]string{
		"operation": operation,
	})
}

func RecordCacheMetrics(ctx context.Context, operation, status string, duration time.Duration) {
	IncrementCounter("cache_operations_total", map[string]string{
		"operation": operation,
		"status":    status,
	})

	RecordHistogram("cache_response_time", float64(duration.Milliseconds()), map[string]string{
		"operation": operation,
	})
}

func GetSystemHealth() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		CPUUsage:    getCPUUsage(),
		MemoryUsage: float64(m.Alloc),
		Goroutines:  runtime.NumGoroutine(),
		HeapSize:    m.HeapSys,
		GCs:         m.NumGC,
		Uptime:      time.Since(startTime),
	}
}
