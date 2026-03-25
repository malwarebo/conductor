package metrics

import (
	"context"
	"sync"
	"time"
)

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type Metric struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type ProviderMetrics struct {
	mu              sync.RWMutex
	totalRequests   int64
	successCount    int64
	failureCount    int64
	totalLatency    int64
	totalCost       float64
	lastHourData    []requestData
	volumeProcessed float64
}

type requestData struct {
	timestamp   time.Time
	success     bool
	latencyMs   int64
	amount      float64
	cost        float64
}

type Collector struct {
	mu        sync.RWMutex
	providers map[string]*ProviderMetrics
	decisions []decisionRecord
	maxDecisions int
}

type decisionRecord struct {
	timestamp       time.Time
	provider        string
	success         bool
	latencyMs       int64
	amount          float64
	currency        string
	merchantID      string
	rulesApplied    []string
	decisionTimeMs  int64
}

func NewCollector() *Collector {
	return &Collector{
		providers:    make(map[string]*ProviderMetrics),
		decisions:    make([]decisionRecord, 0),
		maxDecisions: 10000,
	}
}

func (c *Collector) getOrCreateProvider(name string) *ProviderMetrics {
	c.mu.RLock()
	pm, exists := c.providers[name]
	c.mu.RUnlock()

	if exists {
		return pm
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if pm, exists = c.providers[name]; exists {
		return pm
	}

	pm = &ProviderMetrics{
		lastHourData: make([]requestData, 0),
	}
	c.providers[name] = pm
	return pm
}

func (c *Collector) RecordRequest(provider string, success bool, latencyMs int64, amount, cost float64) {
	pm := c.getOrCreateProvider(provider)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalRequests++
	pm.totalLatency += latencyMs
	pm.totalCost += cost
	pm.volumeProcessed += amount

	if success {
		pm.successCount++
	} else {
		pm.failureCount++
	}

	now := time.Now()
	pm.lastHourData = append(pm.lastHourData, requestData{
		timestamp: now,
		success:   success,
		latencyMs: latencyMs,
		amount:    amount,
		cost:      cost,
	})

	pm.pruneOldData(now)
}

func (pm *ProviderMetrics) pruneOldData(now time.Time) {
	cutoff := now.Add(-time.Hour)
	idx := 0
	for i, r := range pm.lastHourData {
		if r.timestamp.After(cutoff) {
			idx = i
			break
		}
	}
	if idx > 0 {
		pm.lastHourData = pm.lastHourData[idx:]
	}
}

func (c *Collector) RecordDecision(provider string, success bool, latencyMs int64, amount float64, currency, merchantID string, rulesApplied []string, decisionTimeMs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.decisions = append(c.decisions, decisionRecord{
		timestamp:      time.Now(),
		provider:       provider,
		success:        success,
		latencyMs:      latencyMs,
		amount:         amount,
		currency:       currency,
		merchantID:     merchantID,
		rulesApplied:   rulesApplied,
		decisionTimeMs: decisionTimeMs,
	})

	if len(c.decisions) > c.maxDecisions {
		c.decisions = c.decisions[len(c.decisions)-c.maxDecisions:]
	}
}

func (c *Collector) GetProviderStats(provider string) map[string]interface{} {
	pm := c.getOrCreateProvider(provider)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.totalRequests == 0 {
		return map[string]interface{}{
			"success_rate":     1.0,
			"avg_latency_ms":   0,
			"total_requests":   0,
			"volume_processed": 0.0,
		}
	}

	return map[string]interface{}{
		"success_rate":     float64(pm.successCount) / float64(pm.totalRequests),
		"avg_latency_ms":   pm.totalLatency / pm.totalRequests,
		"total_requests":   pm.totalRequests,
		"success_count":    pm.successCount,
		"failure_count":    pm.failureCount,
		"total_cost":       pm.totalCost,
		"volume_processed": pm.volumeProcessed,
	}
}

func (c *Collector) GetRecentSuccessRate(provider string, duration time.Duration) float64 {
	pm := c.getOrCreateProvider(provider)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var total, success int64

	for _, r := range pm.lastHourData {
		if r.timestamp.After(cutoff) {
			total++
			if r.success {
				success++
			}
		}
	}

	if total == 0 {
		return 1.0
	}
	return float64(success) / float64(total)
}

func (c *Collector) GetRecentLatency(provider string, duration time.Duration) int64 {
	pm := c.getOrCreateProvider(provider)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var total, latency int64

	for _, r := range pm.lastHourData {
		if r.timestamp.After(cutoff) {
			total++
			latency += r.latencyMs
		}
	}

	if total == 0 {
		return 0
	}
	return latency / total
}

func (c *Collector) GetVolumeDistribution(duration time.Duration) map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	distribution := make(map[string]float64)
	var total float64

	for _, d := range c.decisions {
		if d.timestamp.After(cutoff) {
			distribution[d.provider] += d.amount
			total += d.amount
		}
	}

	if total == 0 {
		return distribution
	}

	for provider := range distribution {
		distribution[provider] = distribution[provider] / total
	}

	return distribution
}

func (c *Collector) GetAllProviderStats() map[string]map[string]interface{} {
	c.mu.RLock()
	providers := make([]string, 0, len(c.providers))
	for name := range c.providers {
		providers = append(providers, name)
	}
	c.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for _, name := range providers {
		stats[name] = c.GetProviderStats(name)
	}
	return stats
}

func (c *Collector) GetDecisionStats(duration time.Duration) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var total, success int64
	var totalDecisionTime int64
	providerCounts := make(map[string]int64)

	for _, d := range c.decisions {
		if d.timestamp.After(cutoff) {
			total++
			totalDecisionTime += d.decisionTimeMs
			providerCounts[d.provider]++
			if d.success {
				success++
			}
		}
	}

	avgDecisionTime := int64(0)
	successRate := 1.0
	if total > 0 {
		avgDecisionTime = totalDecisionTime / total
		successRate = float64(success) / float64(total)
	}

	return map[string]interface{}{
		"total_decisions":      total,
		"success_rate":         successRate,
		"avg_decision_time_ms": avgDecisionTime,
		"provider_distribution": providerCounts,
	}
}

type RealTimeStats struct {
	Provider       string
	SuccessRate1m  float64
	SuccessRate5m  float64
	SuccessRate1h  float64
	AvgLatency1m   int64
	AvgLatency5m   int64
	RequestCount1m int64
	RequestCount1h int64
	IsHealthy      bool
}

func (c *Collector) GetRealTimeStats(provider string) RealTimeStats {
	pm := c.getOrCreateProvider(provider)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	now := time.Now()
	cutoff1m := now.Add(-time.Minute)
	cutoff5m := now.Add(-5 * time.Minute)

	var count1m, count5m, count1h int64
	var success1m, success5m, success1h int64
	var latency1m, latency5m int64

	for _, r := range pm.lastHourData {
		count1h++
		if r.success {
			success1h++
		}

		if r.timestamp.After(cutoff5m) {
			count5m++
			latency5m += r.latencyMs
			if r.success {
				success5m++
			}
		}

		if r.timestamp.After(cutoff1m) {
			count1m++
			latency1m += r.latencyMs
			if r.success {
				success1m++
			}
		}
	}

	rate1m, rate5m, rate1h := 1.0, 1.0, 1.0
	avgLat1m, avgLat5m := int64(0), int64(0)

	if count1m > 0 {
		rate1m = float64(success1m) / float64(count1m)
		avgLat1m = latency1m / count1m
	}
	if count5m > 0 {
		rate5m = float64(success5m) / float64(count5m)
		avgLat5m = latency5m / count5m
	}
	if count1h > 0 {
		rate1h = float64(success1h) / float64(count1h)
	}

	isHealthy := rate5m >= 0.9 && (count5m < 10 || rate1m >= 0.8)

	return RealTimeStats{
		Provider:       provider,
		SuccessRate1m:  rate1m,
		SuccessRate5m:  rate5m,
		SuccessRate1h:  rate1h,
		AvgLatency1m:   avgLat1m,
		AvgLatency5m:   avgLat5m,
		RequestCount1m: count1m,
		RequestCount1h: count1h,
		IsHealthy:      isHealthy,
	}
}

func (c *Collector) Snapshot() map[string]interface{} {
	return map[string]interface{}{
		"providers": c.GetAllProviderStats(),
		"decisions": c.GetDecisionStats(time.Hour),
		"volume":    c.GetVolumeDistribution(time.Hour),
	}
}

func (c *Collector) Export(ctx context.Context, sink func([]Metric) error) error {
	metrics := make([]Metric, 0)
	now := time.Now()

	c.mu.RLock()
	for name, pm := range c.providers {
		pm.mu.RLock()
		labels := map[string]string{"provider": name}

		metrics = append(metrics,
			Metric{Name: "provider_requests_total", Type: MetricTypeCounter, Value: float64(pm.totalRequests), Labels: labels, Timestamp: now},
			Metric{Name: "provider_success_total", Type: MetricTypeCounter, Value: float64(pm.successCount), Labels: labels, Timestamp: now},
			Metric{Name: "provider_failure_total", Type: MetricTypeCounter, Value: float64(pm.failureCount), Labels: labels, Timestamp: now},
			Metric{Name: "provider_volume_total", Type: MetricTypeCounter, Value: pm.volumeProcessed, Labels: labels, Timestamp: now},
		)

		if pm.totalRequests > 0 {
			metrics = append(metrics,
				Metric{Name: "provider_success_rate", Type: MetricTypeGauge, Value: float64(pm.successCount) / float64(pm.totalRequests), Labels: labels, Timestamp: now},
				Metric{Name: "provider_avg_latency_ms", Type: MetricTypeGauge, Value: float64(pm.totalLatency / pm.totalRequests), Labels: labels, Timestamp: now},
			)
		}
		pm.mu.RUnlock()
	}
	c.mu.RUnlock()

	return sink(metrics)
}
