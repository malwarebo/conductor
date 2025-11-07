package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/utils"
)

const (
	routingSystemPrompt = `You are an expert payment routing analyst for a multi-provider payment orchestration system.
Analyze the following transaction data and provider performance metrics to determine the optimal payment provider.

Available providers:
- Stripe: Best for USD, EUR, GBP. High success rates, premium pricing
- Xendit: Best for IDR, SGD, MYR, PHP, THB, VND. Regional expertise, competitive pricing

Consider these factors in your analysis:
1. Currency optimization (use regional providers for local currencies)
2. Success rate (higher is better)
3. Cost efficiency (lower fees)
4. Transaction amount (some providers have better rates for high amounts)
5. Geographic location (use local providers when possible)
6. Historical performance data

Your response MUST be a valid JSON object with these keys:
1. "recommended_provider": string (stripe, xendit, or fallback)
2. "confidence_score": integer between 0-100
3. "reasoning": string explaining your decision
4. "alternative_provider": string (backup option)
5. "estimated_success_rate": float between 0.0-1.0
6. "estimated_cost": float (estimated processing cost)`
)

type RoutingService interface {
	SelectOptimalProvider(ctx context.Context, request *models.RoutingRequest) (*models.RoutingResponse, error)
	GetProviderStats(ctx context.Context) (*models.ProviderStatsResponse, error)
}

type routingService struct {
	openAIKey     string
	httpClient    *http.Client
	providerStats map[string]*models.ProviderStats
	cache         map[string]*models.RoutingResponse
}

type RoutingOpenAIRequest struct {
	Model    string           `json:"model"`
	Messages []RoutingMessage `json:"messages"`
}

type RoutingOpenAIResponse struct {
	Choices []RoutingChoice `json:"choices"`
}

type RoutingChoice struct {
	Message RoutingMessage `json:"message"`
}

type RoutingMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func CreateRoutingService(openAIKey string) RoutingService {
	return &routingService{
		openAIKey: openAIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		providerStats: make(map[string]*models.ProviderStats),
		cache:         make(map[string]*models.RoutingResponse),
	}
}

func (s *routingService) SelectOptimalProvider(ctx context.Context, request *models.RoutingRequest) (*models.RoutingResponse, error) {
	cacheKey := fmt.Sprintf("%s_%s_%s_%s_%s",
		request.Currency,
		request.Amount,
		request.Country,
		request.CustomerSegment,
		request.TransactionType)

	if cached, exists := s.cache[cacheKey]; exists {
		utils.CreateLogger("conductor").Info(ctx, "Using cached routing decision", map[string]interface{}{
			"cache_key": cacheKey,
			"provider":  cached.RecommendedProvider,
		})
		return cached, nil
	}

	routingData := s.prepareRoutingData(request)

	response, err := s.callOpenAIRouting(ctx, routingData)
	if err != nil {
		log.Printf("OpenAI routing failed, using fallback logic: %v", err)
		response = s.fallbackRouting(request)
	}

	s.cache[cacheKey] = response

	utils.CreateRecordRoutingMetrics(ctx, response.RecommendedProvider, response.ConfidenceScore, response.EstimatedSuccessRate)

	return response, nil
}

func (s *routingService) prepareRoutingData(request *models.RoutingRequest) map[string]interface{} {
	stripeStats := s.getProviderStats("stripe")
	xenditStats := s.getProviderStats("xendit")

	return map[string]interface{}{
		"transaction_amount": request.Amount,
		"currency":           request.Currency,
		"country":            request.Country,
		"customer_segment":   request.CustomerSegment,
		"transaction_type":   request.TransactionType,
		"time_of_day":        time.Now().Hour(),
		"day_of_week":        time.Now().Weekday().String(),

		"stripe_stats": map[string]interface{}{
			"success_rate":         stripeStats.SuccessRate,
			"avg_response_time":    stripeStats.AvgResponseTime,
			"cost_per_transaction": stripeStats.CostPerTransaction,
			"supported_currencies": []string{"USD", "EUR", "GBP"},
		},
		"xendit_stats": map[string]interface{}{
			"success_rate":         xenditStats.SuccessRate,
			"avg_response_time":    xenditStats.AvgResponseTime,
			"cost_per_transaction": xenditStats.CostPerTransaction,
			"supported_currencies": []string{"IDR", "SGD", "MYR", "PHP", "THB", "VND"},
		},

		"amount_category":  s.categorizeAmount(request.Amount),
		"is_high_value":    request.Amount > 1000,
		"is_international": request.Country != "US",
	}
}

func (s *routingService) callOpenAIRouting(ctx context.Context, routingData map[string]interface{}) (*models.RoutingResponse, error) {
	jsonData, err := json.Marshal(routingData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal routing data: %w", err)
	}

	requestBody := RoutingOpenAIRequest{
		Model: "gpt-4o",
		Messages: []RoutingMessage{
			{
				Role:    "system",
				Content: routingSystemPrompt,
			},
			{
				Role:    "user",
				Content: string(jsonData),
			},
		},
	}

	jsonReq, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.openAIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp RoutingOpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	var routingResponse models.RoutingResponse
	if err := json.Unmarshal([]byte(openAIResp.Choices[0].Message.Content), &routingResponse); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI routing response: %w", err)
	}

	return &routingResponse, nil
}

func (s *routingService) fallbackRouting(request *models.RoutingRequest) *models.RoutingResponse {
	var recommendedProvider string
	var confidenceScore int
	var reasoning string

	switch request.Currency {
	case "USD", "EUR", "GBP":
		recommendedProvider = "stripe"
		confidenceScore = 85
		reasoning = "Currency matches Stripe's primary markets"
	case "IDR", "SGD", "MYR", "PHP", "THB", "VND":
		recommendedProvider = "xendit"
		confidenceScore = 90
		reasoning = "Currency matches Xendit's Southeast Asian focus"
	default:
		recommendedProvider = "stripe"
		confidenceScore = 70
		reasoning = "Fallback to Stripe for unsupported currency"
	}

	return &models.RoutingResponse{
		RecommendedProvider:  recommendedProvider,
		ConfidenceScore:      confidenceScore,
		Reasoning:            reasoning,
		AlternativeProvider:  "xendit",
		EstimatedSuccessRate: 0.95,
		EstimatedCost:        s.estimateCost(request.Amount, recommendedProvider),
	}
}

func (s *routingService) getProviderStats(provider string) *models.ProviderStats {
	if stats, exists := s.providerStats[provider]; exists {
		return stats
	}

	defaultStats := &models.ProviderStats{
		SuccessRate:        0.95,
		AvgResponseTime:    500,   // ms
		CostPerTransaction: 0.029, // 2.9%
	}

	s.providerStats[provider] = defaultStats
	return defaultStats
}

func (s *routingService) GetProviderStats(ctx context.Context) (*models.ProviderStatsResponse, error) {
	return &models.ProviderStatsResponse{
		Stripe: s.getProviderStats("stripe"),
		Xendit: s.getProviderStats("xendit"),
	}, nil
}

func (s *routingService) categorizeAmount(amount float64) string {
	if amount < 50 {
		return "low"
	} else if amount < 500 {
		return "medium"
	} else if amount < 2000 {
		return "high"
	}
	return "very_high"
}

func (s *routingService) estimateCost(amount float64, provider string) float64 {
	switch provider {
	case "stripe":
		return amount*0.029 + 0.30 // 2.9% + $0.30
	case "xendit":
		return amount*0.025 + 0.20 // 2.5% + $0.20
	default:
		return amount * 0.030 // 3% fallback
	}
}
