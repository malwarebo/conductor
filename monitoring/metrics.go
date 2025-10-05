package monitoring

import (
	"sync"
	"time"
)

type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
)

type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

type MetricsCollector struct {
	metrics map[string]*Metric
	mu      sync.RWMutex
}

func CreateMetricsCollector() *MetricsCollector {
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

func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) (*Metric, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.getKey(name, labels)
	metric, exists := mc.metrics[key]
	return metric, exists
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

func (mc *MetricsCollector) GetMetricsByType(metricType MetricType) []*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var result []*Metric
	for _, metric := range mc.metrics {
		if metric.Type == metricType {
			result = append(result, metric)
		}
	}
	return result
}

func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = make(map[string]*Metric)
}

func (mc *MetricsCollector) getKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ":" + k + "=" + v
	}
	return key
}

type PrometheusExporter struct {
	collector *MetricsCollector
}

func CreatePrometheusExporter(collector *MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
	}
}

func (pe *PrometheusExporter) Export() string {
	metrics := pe.collector.GetAllMetrics()
	var output string

	for _, metric := range metrics {
		output += "# HELP " + metric.Name + " " + metric.Name + "\n"
		output += "# TYPE " + metric.Name + " " + string(metric.Type) + "\n"

		labels := ""
		if len(metric.Labels) > 0 {
			labels = "{"
			first := true
			for k, v := range metric.Labels {
				if !first {
					labels += ","
				}
				labels += k + "=\"" + v + "\""
				first = false
			}
			labels += "}"
		}

		output += metric.Name + labels + " " + formatFloat(metric.Value) + "\n"
	}

	return output
}

func formatFloat(value float64) string {
	if value == float64(int(value)) {
		return string(rune(int(value)))
	}
	return string(rune(value))
}
