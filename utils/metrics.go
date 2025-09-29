package utils

import (
	"context"
	"sync"
	"time"
)

type MetricType int

const (
	Counter MetricType = iota
	Gauge
	Histogram
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
	mutex   sync.RWMutex
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
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

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
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

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
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

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
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	key := mc.getKey(name, labels)
	return mc.metrics[key]
}

func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

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

	Info(ctx, "Payment metric recorded", map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"provider": provider,
		"status":   status,
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

	Info(ctx, "Fraud metric recorded", map[string]interface{}{
		"is_fraudulent": isFraudulent,
		"fraud_score":   fraudScore,
		"reason":        reason,
	})
}

func RecordProviderMetrics(ctx context.Context, provider string, operation string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	IncrementCounter("provider_operations_total", map[string]string{
		"provider":  provider,
		"operation": operation,
		"status":    status,
	})

	Info(ctx, "Provider metric recorded", map[string]interface{}{
		"provider":  provider,
		"operation": operation,
		"success":   success,
	})
}

func RecordRoutingMetrics(ctx context.Context, provider string, confidenceScore int, successRate float64) {
	IncrementCounter("routing_decisions_total", map[string]string{
		"provider":         provider,
		"confidence_level": getConfidenceLevel(confidenceScore),
	})

	SetGauge("routing_confidence_score", float64(confidenceScore), map[string]string{
		"provider": provider,
	})

	SetGauge("routing_success_rate", successRate, map[string]string{
		"provider": provider,
	})

	Info(ctx, "Routing metric recorded", map[string]interface{}{
		"provider":         provider,
		"confidence_score": confidenceScore,
		"success_rate":     successRate,
	})
}

func getConfidenceLevel(score int) string {
	if score >= 90 {
		return "high"
	} else if score >= 70 {
		return "medium"
	} else {
		return "low"
	}
}
