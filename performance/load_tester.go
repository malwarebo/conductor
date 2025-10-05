package performance

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type LoadTestConfig struct {
	Concurrency    int
	Duration       time.Duration
	RampUpDuration time.Duration
	TargetRPS      float64
}

type LoadTestResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	Throughput         float64
	ErrorRate          float64
	Duration           time.Duration
}

type LoadTester struct {
	client  *http.Client
	results []LoadTestResult
	mu      sync.RWMutex
}

func CreateLoadTester() *LoadTester {
	return &LoadTester{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		results: make([]LoadTestResult, 0),
	}
}

func (lt *LoadTester) RunLoadTest(ctx context.Context, config LoadTestConfig, requestFunc func() (*http.Request, error)) LoadTestResult {
	start := time.Now()

	var wg sync.WaitGroup
	requestChan := make(chan struct{}, config.Concurrency)
	latencies := make([]time.Duration, 0)
	successCount := int64(0)
	failedCount := int64(0)

	var mu sync.Mutex

	rampUpDuration := config.RampUpDuration / time.Duration(config.Concurrency)

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			time.Sleep(time.Duration(workerID) * rampUpDuration)

			for {
				select {
				case <-ctx.Done():
					return
				case <-requestChan:
					req, err := requestFunc()
					if err != nil {
						mu.Lock()
						failedCount++
						mu.Unlock()
						continue
					}

					reqStart := time.Now()
					resp, err := lt.client.Do(req)
					latency := time.Since(reqStart)

					mu.Lock()
					latencies = append(latencies, latency)
					if err != nil || resp.StatusCode >= 400 {
						failedCount++
					} else {
						successCount++
					}
					mu.Unlock()

					if resp != nil {
						resp.Body.Close()
					}
				}
			}
		}(i)
	}

	rateLimiter := time.NewTicker(time.Duration(1e9/config.TargetRPS) * time.Nanosecond)
	defer rateLimiter.Stop()

	endTime := start.Add(config.Duration)

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			break
		case <-rateLimiter.C:
			select {
			case requestChan <- struct{}{}:
			default:
			}
		}
	}

	close(requestChan)
	wg.Wait()

	actualDuration := time.Since(start)

	result := lt.calculateResult(latencies, successCount, failedCount, actualDuration)

	lt.mu.Lock()
	lt.results = append(lt.results, result)
	lt.mu.Unlock()

	return result
}

func (lt *LoadTester) calculateResult(latencies []time.Duration, successCount, failedCount int64, duration time.Duration) LoadTestResult {
	if len(latencies) == 0 {
		return LoadTestResult{
			TotalRequests:      0,
			SuccessfulRequests: 0,
			FailedRequests:     0,
			Duration:           duration,
		}
	}

	totalRequests := successCount + failedCount
	avgLatency := time.Duration(0)
	minLatency := latencies[0]
	maxLatency := latencies[0]

	for _, latency := range latencies {
		avgLatency += latency
		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	avgLatency = avgLatency / time.Duration(len(latencies))

	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	p95Latency := latencies[p95Index]
	p99Latency := latencies[p99Index]

	throughput := float64(totalRequests) / duration.Seconds()
	errorRate := float64(failedCount) / float64(totalRequests)

	return LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successCount,
		FailedRequests:     failedCount,
		AverageLatency:     avgLatency,
		MinLatency:         minLatency,
		MaxLatency:         maxLatency,
		P95Latency:         p95Latency,
		P99Latency:         p99Latency,
		Throughput:         throughput,
		ErrorRate:          errorRate,
		Duration:           duration,
	}
}

func (lt *LoadTester) GetResults() []LoadTestResult {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.results
}

func (lt *LoadTester) GetSummary() map[string]interface{} {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.results) == 0 {
		return map[string]interface{}{}
	}

	totalRequests := int64(0)
	totalSuccessful := int64(0)
	totalFailed := int64(0)
	totalDuration := time.Duration(0)

	for _, result := range lt.results {
		totalRequests += result.TotalRequests
		totalSuccessful += result.SuccessfulRequests
		totalFailed += result.FailedRequests
		totalDuration += result.Duration
	}

	avgThroughput := float64(totalRequests) / totalDuration.Seconds()
	overallErrorRate := float64(totalFailed) / float64(totalRequests)

	return map[string]interface{}{
		"total_requests":      totalRequests,
		"successful_requests": totalSuccessful,
		"failed_requests":     totalFailed,
		"avg_throughput":      avgThroughput,
		"overall_error_rate":  overallErrorRate,
		"test_count":          len(lt.results),
	}
}
