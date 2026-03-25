package models

import "time"

type BINData struct {
	BIN           string            `json:"bin" gorm:"primaryKey"`
	CardBrand     string            `json:"card_brand"`
	CardType      string            `json:"card_type"`
	IssuingBank   string            `json:"issuing_bank"`
	IssuingCountry string           `json:"issuing_country"`
	Category      string            `json:"category"`
	ProviderStats map[string]*BINProviderStats `json:"provider_stats" gorm:"-"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type BINProviderStats struct {
	BIN             string    `json:"bin" gorm:"primaryKey"`
	ProviderName    string    `json:"provider_name" gorm:"primaryKey"`
	SuccessRate     float64   `json:"success_rate"`
	AvgResponseTime int64     `json:"avg_response_time_ms"`
	TotalAttempts   int64     `json:"total_attempts"`
	SuccessCount    int64     `json:"success_count"`
	LastUpdated     time.Time `json:"last_updated"`
}

func (b *BINProviderStats) TableName() string {
	return "bin_provider_stats"
}

type MerchantRoutingConfig struct {
	MerchantID          string                 `json:"merchant_id" gorm:"primaryKey"`
	PreferredProviders  []string               `json:"preferred_providers" gorm:"serializer:json"`
	ExcludedProviders   []string               `json:"excluded_providers" gorm:"serializer:json"`
	CurrencyPreferences map[string]string      `json:"currency_preferences" gorm:"serializer:json"`
	MinSuccessRate      float64                `json:"min_success_rate"`
	MaxCostPercent      float64                `json:"max_cost_percent"`
	EnableSmartRouting  bool                   `json:"enable_smart_routing"`
	EnableRetry         bool                   `json:"enable_retry"`
	MaxRetryAttempts    int                    `json:"max_retry_attempts"`
	VolumeTargets       map[string]float64     `json:"volume_targets" gorm:"serializer:json"`
	Metadata            map[string]interface{} `json:"metadata" gorm:"serializer:json"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type RoutingRule struct {
	ID             string                 `json:"id" gorm:"primaryKey"`
	Name           string                 `json:"name"`
	Priority       int                    `json:"priority"`
	Enabled        bool                   `json:"enabled"`
	Conditions     RoutingConditions      `json:"conditions" gorm:"serializer:json"`
	TargetProvider string                 `json:"target_provider"`
	Weight         float64                `json:"weight"`
	Metadata       map[string]interface{} `json:"metadata" gorm:"serializer:json"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type RoutingConditions struct {
	Currencies      []string `json:"currencies,omitempty"`
	Countries       []string `json:"countries,omitempty"`
	CardBrands      []string `json:"card_brands,omitempty"`
	BINPrefixes     []string `json:"bin_prefixes,omitempty"`
	MinAmount       *float64 `json:"min_amount,omitempty"`
	MaxAmount       *float64 `json:"max_amount,omitempty"`
	PaymentMethods  []string `json:"payment_methods,omitempty"`
	CustomerSegments []string `json:"customer_segments,omitempty"`
	TimeRanges      []TimeRange `json:"time_ranges,omitempty"`
}

type TimeRange struct {
	StartHour int      `json:"start_hour"`
	EndHour   int      `json:"end_hour"`
	DaysOfWeek []int   `json:"days_of_week,omitempty"`
}

type ProviderScore struct {
	ProviderName    string  `json:"provider_name"`
	Score           float64 `json:"score"`
	SuccessRate     float64 `json:"success_rate"`
	CostScore       float64 `json:"cost_score"`
	LatencyScore    float64 `json:"latency_score"`
	BINScore        float64 `json:"bin_score"`
	HealthScore     float64 `json:"health_score"`
	VolumeScore     float64 `json:"volume_score"`
	Eligible        bool    `json:"eligible"`
	Reason          string  `json:"reason,omitempty"`
}

type RoutingDecision struct {
	ID                  string           `json:"id"`
	TransactionID       string           `json:"transaction_id"`
	MerchantID          string           `json:"merchant_id"`
	SelectedProvider    string           `json:"selected_provider"`
	FallbackProviders   []string         `json:"fallback_providers"`
	Scores              []ProviderScore  `json:"scores"`
	RulesApplied        []string         `json:"rules_applied"`
	DecisionTimeMs      int64            `json:"decision_time_ms"`
	Reason              string           `json:"reason"`
	Attempt             int              `json:"attempt"`
	PreviousAttempts    []AttemptResult  `json:"previous_attempts,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
}

type AttemptResult struct {
	Provider      string    `json:"provider"`
	Success       bool      `json:"success"`
	ErrorCode     string    `json:"error_code,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	ResponseTimeMs int64    `json:"response_time_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

type RoutingContext struct {
	TransactionID   string
	MerchantID      string
	Amount          float64
	Currency        string
	Country         string
	PaymentMethod   string
	CardBIN         string
	CardBrand       string
	CustomerID      string
	CustomerSegment string
	Metadata        map[string]interface{}
}
