package routing

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/malwarebo/conductor/internal/circuitbreaker"
	"github.com/malwarebo/conductor/internal/metrics"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
)

type ScoringWeights struct {
	SuccessRate float64
	Cost        float64
	Latency     float64
	BIN         float64
	Health      float64
	Volume      float64
}

func DefaultWeights() ScoringWeights {
	return ScoringWeights{
		SuccessRate: 0.35,
		Cost:        0.20,
		Latency:     0.10,
		BIN:         0.20,
		Health:      0.10,
		Volume:      0.05,
	}
}

type ProviderCosts struct {
	FixedFee    float64
	PercentFee  float64
	MinFee      float64
	MaxFee      float64
}

type Engine struct {
	circuitBreakers  *circuitbreaker.Manager
	metricsCollector *metrics.Collector
	binStore         *stores.BINStore
	merchantStore    *stores.MerchantConfigStore
	ruleStore        *stores.RoutingRuleStore
	weights          ScoringWeights
	providerCosts    map[string]ProviderCosts
	availableProviders []string
}

type Config struct {
	CircuitBreakerConfig circuitbreaker.Config
	Weights              ScoringWeights
	ProviderCosts        map[string]ProviderCosts
	AvailableProviders   []string
}

func DefaultConfig() Config {
	return Config{
		CircuitBreakerConfig: circuitbreaker.DefaultConfig(),
		Weights:              DefaultWeights(),
		ProviderCosts: map[string]ProviderCosts{
			"stripe":    {FixedFee: 0.30, PercentFee: 0.029},
			"xendit":    {FixedFee: 0.20, PercentFee: 0.025},
			"razorpay":  {FixedFee: 0.00, PercentFee: 0.020},
			"airwallex": {FixedFee: 0.25, PercentFee: 0.028},
		},
		AvailableProviders: []string{"stripe", "xendit", "razorpay", "airwallex"},
	}
}

func NewEngine(binStore *stores.BINStore, merchantStore *stores.MerchantConfigStore, ruleStore *stores.RoutingRuleStore, cfg Config) *Engine {
	return &Engine{
		circuitBreakers:    circuitbreaker.NewManager(cfg.CircuitBreakerConfig),
		metricsCollector:   metrics.NewCollector(),
		binStore:           binStore,
		merchantStore:      merchantStore,
		ruleStore:          ruleStore,
		weights:            cfg.Weights,
		providerCosts:      cfg.ProviderCosts,
		availableProviders: cfg.AvailableProviders,
	}
}

func (e *Engine) Route(ctx context.Context, rc *models.RoutingContext) (*models.RoutingDecision, error) {
	start := time.Now()

	merchantConfig := e.merchantStore.GetOrDefault(ctx, rc.MerchantID)
	eligibleProviders := e.getEligibleProviders(ctx, rc, merchantConfig)

	if len(eligibleProviders) == 0 {
		return nil, ErrNoEligibleProviders
	}

	scores := e.scoreProviders(ctx, rc, eligibleProviders, merchantConfig)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	rulesApplied := e.applyRules(ctx, rc, scores)

	fallbacks := make([]string, 0, len(scores)-1)
	for i := 1; i < len(scores) && i <= 3; i++ {
		if scores[i].Eligible {
			fallbacks = append(fallbacks, scores[i].ProviderName)
		}
	}

	decision := &models.RoutingDecision{
		ID:                generateID(),
		TransactionID:     rc.TransactionID,
		MerchantID:        rc.MerchantID,
		SelectedProvider:  scores[0].ProviderName,
		FallbackProviders: fallbacks,
		Scores:            scores,
		RulesApplied:      rulesApplied,
		DecisionTimeMs:    time.Since(start).Milliseconds(),
		Reason:            e.buildReason(scores[0]),
		Attempt:           1,
		CreatedAt:         time.Now(),
	}

	return decision, nil
}

func (e *Engine) getEligibleProviders(ctx context.Context, rc *models.RoutingContext, config *models.MerchantRoutingConfig) []string {
	eligible := make([]string, 0)

	excluded := make(map[string]bool)
	for _, p := range config.ExcludedProviders {
		excluded[p] = true
	}

	preferred := make(map[string]bool)
	for _, p := range config.PreferredProviders {
		preferred[p] = true
	}

	if pref, ok := config.CurrencyPreferences[rc.Currency]; ok && pref != "" {
		return []string{pref}
	}

	for _, provider := range e.availableProviders {
		if excluded[provider] {
			continue
		}

		cb := e.circuitBreakers.Get(provider)
		if !cb.Allow() {
			continue
		}

		if !e.providerSupportsCurrency(provider, rc.Currency) {
			continue
		}

		eligible = append(eligible, provider)
	}

	if len(config.PreferredProviders) > 0 {
		prioritized := make([]string, 0)
		others := make([]string, 0)
		for _, p := range eligible {
			if preferred[p] {
				prioritized = append(prioritized, p)
			} else {
				others = append(others, p)
			}
		}
		eligible = append(prioritized, others...)
	}

	return eligible
}

func (e *Engine) providerSupportsCurrency(provider, currency string) bool {
	supported := map[string][]string{
		"stripe":    {"USD", "EUR", "GBP", "CAD", "AUD", "JPY", "CHF", "HKD", "SGD", "NZD"},
		"xendit":    {"IDR", "PHP", "VND", "THB", "MYR", "SGD"},
		"razorpay":  {"INR", "USD", "EUR", "GBP", "SGD", "AED", "AUD", "CAD", "HKD", "JPY", "MYR", "SAR"},
		"airwallex": {"USD", "EUR", "GBP", "AUD", "NZD", "HKD", "SGD", "CNY", "JPY", "CHF", "CAD", "ILS", "KRW"},
	}

	currencies, ok := supported[provider]
	if !ok {
		return true
	}

	for _, c := range currencies {
		if c == currency {
			return true
		}
	}
	return false
}

func (e *Engine) scoreProviders(ctx context.Context, rc *models.RoutingContext, providers []string, config *models.MerchantRoutingConfig) []models.ProviderScore {
	scores := make([]models.ProviderScore, 0, len(providers))

	for _, provider := range providers {
		score := e.calculateScore(ctx, provider, rc, config)
		scores = append(scores, score)
	}

	return scores
}

func (e *Engine) calculateScore(ctx context.Context, provider string, rc *models.RoutingContext, config *models.MerchantRoutingConfig) models.ProviderScore {
	score := models.ProviderScore{
		ProviderName: provider,
		Eligible:     true,
	}

	stats := e.metricsCollector.GetRealTimeStats(provider)
	score.SuccessRate = e.normalizeSuccessRate(stats.SuccessRate5m)

	costPct := e.calculateCostPercent(provider, rc.Amount)
	score.CostScore = e.normalizeCost(costPct)

	score.LatencyScore = e.normalizeLatency(stats.AvgLatency5m)

	score.BINScore = e.calculateBINScore(ctx, provider, rc.CardBIN)

	cb := e.circuitBreakers.Get(provider)
	score.HealthScore = e.normalizeHealth(cb.SuccessRate(), cb.State())

	score.VolumeScore = e.calculateVolumeScore(ctx, provider, config)

	if stats.SuccessRate5m < config.MinSuccessRate && stats.RequestCount1h > 10 {
		score.Eligible = false
		score.Reason = "success rate below threshold"
	}

	if config.MaxCostPercent > 0 && costPct > config.MaxCostPercent {
		score.Eligible = false
		score.Reason = "cost exceeds maximum"
	}

	score.Score = (score.SuccessRate * e.weights.SuccessRate) +
		(score.CostScore * e.weights.Cost) +
		(score.LatencyScore * e.weights.Latency) +
		(score.BINScore * e.weights.BIN) +
		(score.HealthScore * e.weights.Health) +
		(score.VolumeScore * e.weights.Volume)

	return score
}

func (e *Engine) normalizeSuccessRate(rate float64) float64 {
	if rate >= 0.99 {
		return 1.0
	}
	if rate <= 0.80 {
		return 0.0
	}
	return (rate - 0.80) / 0.19
}

func (e *Engine) normalizeCost(costPct float64) float64 {
	if costPct <= 0.015 {
		return 1.0
	}
	if costPct >= 0.04 {
		return 0.0
	}
	return 1.0 - ((costPct - 0.015) / 0.025)
}

func (e *Engine) normalizeLatency(latencyMs int64) float64 {
	if latencyMs <= 200 {
		return 1.0
	}
	if latencyMs >= 2000 {
		return 0.0
	}
	return 1.0 - (float64(latencyMs-200) / 1800.0)
}

func (e *Engine) normalizeHealth(successRate float64, state circuitbreaker.State) float64 {
	if state == circuitbreaker.StateOpen {
		return 0.0
	}
	if state == circuitbreaker.StateHalfOpen {
		return 0.5
	}
	return successRate
}

func (e *Engine) calculateCostPercent(provider string, amount float64) float64 {
	costs, ok := e.providerCosts[provider]
	if !ok {
		return 0.03
	}

	totalCost := costs.FixedFee + (amount * costs.PercentFee)
	if costs.MinFee > 0 && totalCost < costs.MinFee {
		totalCost = costs.MinFee
	}
	if costs.MaxFee > 0 && totalCost > costs.MaxFee {
		totalCost = costs.MaxFee
	}

	if amount == 0 {
		return 0
	}
	return totalCost / amount
}

func (e *Engine) calculateBINScore(ctx context.Context, provider, bin string) float64 {
	if bin == "" || len(bin) < 6 {
		return 0.5
	}

	if e.binStore == nil {
		return 0.5
	}

	bestProvider, bestRate, err := e.binStore.GetBestProvider(ctx, bin[:6], 10)
	if err != nil || bestProvider == "" {
		return 0.5
	}

	if bestProvider == provider {
		return 1.0
	}

	stats, err := e.binStore.GetProviderStats(ctx, bin[:6])
	if err != nil {
		return 0.5
	}

	if providerStats, ok := stats[provider]; ok {
		if bestRate == 0 {
			return 0.5
		}
		return providerStats.SuccessRate / bestRate
	}

	return 0.3
}

func (e *Engine) calculateVolumeScore(ctx context.Context, provider string, config *models.MerchantRoutingConfig) float64 {
	if len(config.VolumeTargets) == 0 {
		return 1.0
	}

	target, ok := config.VolumeTargets[provider]
	if !ok {
		return 1.0
	}

	distribution := e.metricsCollector.GetVolumeDistribution(time.Hour)
	current := distribution[provider]

	if current < target*0.8 {
		return 1.0
	}
	if current > target*1.2 {
		return 0.5
	}
	return 0.8
}

func (e *Engine) applyRules(ctx context.Context, rc *models.RoutingContext, scores []models.ProviderScore) []string {
	if e.ruleStore == nil {
		return nil
	}

	rules, err := e.ruleStore.GetAll(ctx)
	if err != nil {
		return nil
	}

	applied := make([]string, 0)

	for _, rule := range rules {
		if !e.ruleMatches(&rule, rc) {
			continue
		}

		applied = append(applied, rule.Name)

		for i := range scores {
			if scores[i].ProviderName == rule.TargetProvider {
				scores[i].Score += rule.Weight
				break
			}
		}
	}

	return applied
}

func (e *Engine) ruleMatches(rule *models.RoutingRule, rc *models.RoutingContext) bool {
	c := rule.Conditions

	if len(c.Currencies) > 0 && !contains(c.Currencies, rc.Currency) {
		return false
	}

	if len(c.Countries) > 0 && !contains(c.Countries, rc.Country) {
		return false
	}

	if len(c.CardBrands) > 0 && !contains(c.CardBrands, rc.CardBrand) {
		return false
	}

	if len(c.BINPrefixes) > 0 {
		matched := false
		for _, prefix := range c.BINPrefixes {
			if strings.HasPrefix(rc.CardBIN, prefix) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if c.MinAmount != nil && rc.Amount < *c.MinAmount {
		return false
	}

	if c.MaxAmount != nil && rc.Amount > *c.MaxAmount {
		return false
	}

	if len(c.PaymentMethods) > 0 && !contains(c.PaymentMethods, rc.PaymentMethod) {
		return false
	}

	if len(c.CustomerSegments) > 0 && !contains(c.CustomerSegments, rc.CustomerSegment) {
		return false
	}

	if len(c.TimeRanges) > 0 {
		now := time.Now()
		hour := now.Hour()
		weekday := int(now.Weekday())

		matched := false
		for _, tr := range c.TimeRanges {
			if hour >= tr.StartHour && hour < tr.EndHour {
				if len(tr.DaysOfWeek) == 0 || containsInt(tr.DaysOfWeek, weekday) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (e *Engine) buildReason(score models.ProviderScore) string {
	parts := make([]string, 0, 3)

	if score.SuccessRate >= 0.9 {
		parts = append(parts, "high success rate")
	}
	if score.CostScore >= 0.8 {
		parts = append(parts, "cost efficient")
	}
	if score.BINScore >= 0.9 {
		parts = append(parts, "optimal for card type")
	}
	if score.HealthScore >= 0.95 {
		parts = append(parts, "healthy")
	}

	if len(parts) == 0 {
		return "best available option"
	}
	return strings.Join(parts, ", ")
}

func (e *Engine) RecordResult(provider string, success bool, latencyMs int64, amount, cost float64) {
	cb := e.circuitBreakers.Get(provider)
	if success {
		cb.RecordSuccess()
	} else {
		cb.RecordFailure()
	}

	e.metricsCollector.RecordRequest(provider, success, latencyMs, amount, cost)
}

func (e *Engine) RecordBINResult(ctx context.Context, bin, provider string, success bool, latencyMs int64) {
	if e.binStore != nil && len(bin) >= 6 {
		e.binStore.UpdateProviderStats(ctx, bin[:6], provider, success, latencyMs)
	}
}

func (e *Engine) GetCircuitBreakerStats() map[string]map[string]interface{} {
	return e.circuitBreakers.AllStats()
}

func (e *Engine) GetMetricsSnapshot() map[string]interface{} {
	return e.metricsCollector.Snapshot()
}

func (e *Engine) GetProviderHealth(provider string) bool {
	cb := e.circuitBreakers.Get(provider)
	return cb.IsHealthy()
}

func (e *Engine) GetHealthyProviders() []string {
	return e.circuitBreakers.HealthyProviders()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateID() string {
	return time.Now().Format("20060102150405") + randomSuffix()
}

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(b)
}
