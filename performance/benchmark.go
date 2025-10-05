package performance

import (
	"sync"
	"time"
)

type BenchmarkResult struct {
	Name        string
	Duration    time.Duration
	Operations  int64
	Throughput  float64
	MemoryUsage uint64
	ErrorRate   float64
	Errors      int64
}

type BenchmarkSuite struct {
	results []BenchmarkResult
	mu      sync.RWMutex
}

func CreateBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		results: make([]BenchmarkResult, 0),
	}
}

func (bs *BenchmarkSuite) RunBenchmark(name string, duration time.Duration, operation func() error) BenchmarkResult {
	start := time.Now()
	operations := int64(0)
	errors := int64(0)

	end := start.Add(duration)

	for time.Now().Before(end) {
		if err := operation(); err != nil {
			errors++
		}
		operations++
	}

	actualDuration := time.Since(start)
	throughput := float64(operations) / actualDuration.Seconds()
	errorRate := float64(errors) / float64(operations)

	result := BenchmarkResult{
		Name:       name,
		Duration:   actualDuration,
		Operations: operations,
		Throughput: throughput,
		ErrorRate:  errorRate,
		Errors:     errors,
	}

	bs.mu.Lock()
	bs.results = append(bs.results, result)
	bs.mu.Unlock()

	return result
}

func (bs *BenchmarkSuite) RunConcurrentBenchmark(name string, duration time.Duration, concurrency int, operation func() error) BenchmarkResult {
	start := time.Now()
	operations := int64(0)
	errors := int64(0)
	var mu sync.Mutex

	var wg sync.WaitGroup
	end := start.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(end) {
				if err := operation(); err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
				}
				mu.Lock()
				operations++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start)
	throughput := float64(operations) / actualDuration.Seconds()
	errorRate := float64(errors) / float64(operations)

	result := BenchmarkResult{
		Name:       name,
		Duration:   actualDuration,
		Operations: operations,
		Throughput: throughput,
		ErrorRate:  errorRate,
		Errors:     errors,
	}

	bs.mu.Lock()
	bs.results = append(bs.results, result)
	bs.mu.Unlock()

	return result
}

func (bs *BenchmarkSuite) GetResults() []BenchmarkResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.results
}

func (bs *BenchmarkSuite) GetSummary() map[string]interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if len(bs.results) == 0 {
		return map[string]interface{}{}
	}

	totalOps := int64(0)
	totalErrors := int64(0)
	totalDuration := time.Duration(0)

	for _, result := range bs.results {
		totalOps += result.Operations
		totalErrors += result.Errors
		totalDuration += result.Duration
	}

	avgThroughput := float64(totalOps) / totalDuration.Seconds()
	overallErrorRate := float64(totalErrors) / float64(totalOps)

	return map[string]interface{}{
		"total_operations":   totalOps,
		"total_errors":       totalErrors,
		"total_duration":     totalDuration.String(),
		"avg_throughput":     avgThroughput,
		"overall_error_rate": overallErrorRate,
		"benchmark_count":    len(bs.results),
	}
}
