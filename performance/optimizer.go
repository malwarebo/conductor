package performance

import (
	"context"
	"runtime"
	"sync"
	"time"
)

type PerformanceOptimizer struct {
	cacheHitRate    float64
	avgResponseTime time.Duration
	throughput      float64
	errorRate       float64
	mu              sync.RWMutex
}

func CreatePerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{}
}

func (po *PerformanceOptimizer) UpdateMetrics(cacheHitRate float64, avgResponseTime time.Duration, throughput float64, errorRate float64) {
	po.mu.Lock()
	defer po.mu.Unlock()

	po.cacheHitRate = cacheHitRate
	po.avgResponseTime = avgResponseTime
	po.throughput = throughput
	po.errorRate = errorRate
}

func (po *PerformanceOptimizer) GetMetrics() map[string]interface{} {
	po.mu.RLock()
	defer po.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"cache_hit_rate":    po.cacheHitRate,
		"avg_response_time": po.avgResponseTime.String(),
		"throughput":        po.throughput,
		"error_rate":        po.errorRate,
		"memory_alloc":      m.Alloc,
		"memory_sys":        m.Sys,
		"gc_cycles":         m.NumGC,
		"goroutines":        runtime.NumGoroutine(),
	}
}

func (po *PerformanceOptimizer) OptimizeCache(ctx context.Context, cacheSize int, ttl time.Duration) map[string]interface{} {
	optimizations := make(map[string]interface{})

	if po.cacheHitRate < 0.7 {
		optimizations["cache_ttl"] = ttl * 2
		optimizations["cache_size"] = cacheSize * 2
		optimizations["recommendation"] = "Increase cache TTL and size for better hit rate"
	}

	if po.avgResponseTime > 500*time.Millisecond {
		optimizations["response_time"] = "Consider adding more caching layers"
		optimizations["recommendation"] = "High response time detected, optimize database queries"
	}

	if po.errorRate > 0.05 {
		optimizations["error_rate"] = "High error rate detected"
		optimizations["recommendation"] = "Review error handling and retry logic"
	}

	return optimizations
}

func (po *PerformanceOptimizer) GetRecommendations() []string {
	recommendations := make([]string, 0)

	if po.cacheHitRate < 0.7 {
		recommendations = append(recommendations, "Consider increasing cache TTL and size")
	}

	if po.avgResponseTime > 500*time.Millisecond {
		recommendations = append(recommendations, "Optimize database queries and add caching")
	}

	if po.throughput < 100 {
		recommendations = append(recommendations, "Consider horizontal scaling")
	}

	if po.errorRate > 0.05 {
		recommendations = append(recommendations, "Review error handling and circuit breakers")
	}

	return recommendations
}
