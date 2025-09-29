package models

import "time"

type RoutingRequest struct {
	Currency        string                 `json:"currency" validate:"required"`
	Amount          float64                `json:"amount" validate:"required,min=0.01"`
	Country         string                 `json:"country" validate:"required"`
	CustomerSegment string                 `json:"customer_segment"` // premium, standard, enterprise
	TransactionType string                 `json:"transaction_type"` // one_time, subscription, refund
	MerchantID      string                 `json:"merchant_id,omitempty"`
	CustomerID      string                 `json:"customer_id,omitempty"`
	IPAddress       string                 `json:"ip_address,omitempty"`
	UserAgent       string                 `json:"user_agent,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type RoutingResponse struct {
	RecommendedProvider  string  `json:"recommended_provider"`
	ConfidenceScore      int     `json:"confidence_score"` // 0-100
	Reasoning            string  `json:"reasoning"`
	AlternativeProvider  string  `json:"alternative_provider"`
	EstimatedSuccessRate float64 `json:"estimated_success_rate"` // 0.0-1.0
	EstimatedCost        float64 `json:"estimated_cost"`
	RoutingTime          int64   `json:"routing_time_ms"`
	CacheHit             bool    `json:"cache_hit"`
	FallbackUsed         bool    `json:"fallback_used"`
}

type ProviderStats struct {
	SuccessRate        float64   `json:"success_rate"` // 0.0-1.0
	AvgResponseTime    int64     `json:"avg_response_time_ms"`
	CostPerTransaction float64   `json:"cost_per_transaction"`
	TotalTransactions  int64     `json:"total_transactions"`
	FailedTransactions int64     `json:"failed_transactions"`
	LastUpdated        time.Time `json:"last_updated"`
}

type ProviderStatsResponse struct {
	Stripe *ProviderStats `json:"stripe"`
	Xendit *ProviderStats `json:"xendit"`
}

type RoutingHistory struct {
	ID                  string    `json:"id" gorm:"primaryKey"`
	TransactionID       string    `json:"transaction_id"`
	Currency            string    `json:"currency"`
	Amount              float64   `json:"amount"`
	Country             string    `json:"country"`
	RecommendedProvider string    `json:"recommended_provider"`
	ActualProvider      string    `json:"actual_provider"`
	Success             bool      `json:"success"`
	ResponseTime        int64     `json:"response_time_ms"`
	Cost                float64   `json:"cost"`
	ConfidenceScore     int       `json:"confidence_score"`
	Reasoning           string    `json:"reasoning"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type RoutingMetrics struct {
	TotalDecisions       int64            `json:"total_decisions"`
	CacheHitRate         float64          `json:"cache_hit_rate"`
	AvgConfidenceScore   float64          `json:"avg_confidence_score"`
	SuccessRate          float64          `json:"success_rate"`
	AvgResponseTime      int64            `json:"avg_response_time_ms"`
	CostSavings          float64          `json:"cost_savings"`
	ProviderDistribution map[string]int64 `json:"provider_distribution"`
}

type RoutingConfig struct {
	EnableAIRouting               bool   `json:"enable_ai_routing"`
	CacheTTL                      int64  `json:"cache_ttl_seconds"`
	MinConfidenceScore            int    `json:"min_confidence_score"`
	FallbackProvider              string `json:"fallback_provider"`
	EnableCostOptimization        bool   `json:"enable_cost_optimization"`
	EnableSuccessRateOptimization bool   `json:"enable_success_rate_optimization"`
}
