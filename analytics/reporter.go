package analytics

import (
	"time"
)

type PaymentMetric struct {
	Date        time.Time `json:"date"`
	TotalAmount float64   `json:"total_amount"`
	Count       int       `json:"count"`
	Currency    string    `json:"currency"`
	Provider    string    `json:"provider"`
}

type RevenueReport struct {
	Period           string          `json:"period"`
	TotalRevenue     float64         `json:"total_revenue"`
	TransactionCount int             `json:"transaction_count"`
	AverageAmount    float64         `json:"average_amount"`
	Currency         string          `json:"currency"`
	Breakdown        []PaymentMetric `json:"breakdown"`
}

type AnalyticsReporter struct {
	metrics []PaymentMetric
}

func CreateAnalyticsReporter() *AnalyticsReporter {
	return &AnalyticsReporter{
		metrics: make([]PaymentMetric, 0),
	}
}

func (ar *AnalyticsReporter) RecordPayment(amount float64, currency, provider string) {
	metric := PaymentMetric{
		Date:        time.Now(),
		TotalAmount: amount,
		Count:       1,
		Currency:    currency,
		Provider:    provider,
	}
	ar.metrics = append(ar.metrics, metric)
}

func (ar *AnalyticsReporter) GetRevenueReport(period string) RevenueReport {
	var filtered []PaymentMetric

	switch period {
	case "daily":
		filtered = ar.getMetricsForPeriod(24 * time.Hour)
	case "weekly":
		filtered = ar.getMetricsForPeriod(7 * 24 * time.Hour)
	case "monthly":
		filtered = ar.getMetricsForPeriod(30 * 24 * time.Hour)
	default:
		filtered = ar.metrics
	}

	report := RevenueReport{
		Period:    period,
		Breakdown: filtered,
	}

	for _, metric := range filtered {
		report.TotalRevenue += metric.TotalAmount
		report.TransactionCount += metric.Count
	}

	if report.TransactionCount > 0 {
		report.AverageAmount = report.TotalRevenue / float64(report.TransactionCount)
	}

	return report
}

func (ar *AnalyticsReporter) getMetricsForPeriod(duration time.Duration) []PaymentMetric {
	cutoff := time.Now().Add(-duration)
	var filtered []PaymentMetric

	for _, metric := range ar.metrics {
		if metric.Date.After(cutoff) {
			filtered = append(filtered, metric)
		}
	}

	return filtered
}

func (ar *AnalyticsReporter) GetProviderStats() map[string]ProviderStats {
	stats := make(map[string]ProviderStats)

	for _, metric := range ar.metrics {
		if _, exists := stats[metric.Provider]; !exists {
			stats[metric.Provider] = ProviderStats{
				Provider: metric.Provider,
			}
		}

		stat := stats[metric.Provider]
		stat.TotalAmount += metric.TotalAmount
		stat.TransactionCount += metric.Count
		stats[metric.Provider] = stat
	}

	for provider, stat := range stats {
		if stat.TransactionCount > 0 {
			stat.AverageAmount = stat.TotalAmount / float64(stat.TransactionCount)
			stats[provider] = stat
		}
	}

	return stats
}

func (ar *AnalyticsReporter) GetCurrencyStats() map[string]CurrencyStats {
	stats := make(map[string]CurrencyStats)

	for _, metric := range ar.metrics {
		if _, exists := stats[metric.Currency]; !exists {
			stats[metric.Currency] = CurrencyStats{
				Currency: metric.Currency,
			}
		}

		stat := stats[metric.Currency]
		stat.TotalAmount += metric.TotalAmount
		stat.TransactionCount += metric.Count
		stats[metric.Currency] = stat
	}

	return stats
}

func (ar *AnalyticsReporter) GetTrends(days int) []TrendData {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	trends := make(map[string]TrendData)

	for _, metric := range ar.metrics {
		if metric.Date.Before(cutoff) {
			continue
		}

		dateKey := metric.Date.Format("2006-01-02")
		if _, exists := trends[dateKey]; !exists {
			trends[dateKey] = TrendData{
				Date: metric.Date,
			}
		}

		trend := trends[dateKey]
		trend.TotalAmount += metric.TotalAmount
		trend.TransactionCount += metric.Count
		trends[dateKey] = trend
	}

	var result []TrendData
	for _, trend := range trends {
		result = append(result, trend)
	}

	return result
}

type ProviderStats struct {
	Provider         string  `json:"provider"`
	TotalAmount      float64 `json:"total_amount"`
	TransactionCount int     `json:"transaction_count"`
	AverageAmount    float64 `json:"average_amount"`
}

type CurrencyStats struct {
	Currency         string  `json:"currency"`
	TotalAmount      float64 `json:"total_amount"`
	TransactionCount int     `json:"transaction_count"`
}

type TrendData struct {
	Date             time.Time `json:"date"`
	TotalAmount      float64   `json:"total_amount"`
	TransactionCount int       `json:"transaction_count"`
}
