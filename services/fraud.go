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
	"github.com/malwarebo/conductor/stores"
	"github.com/malwarebo/conductor/utils"
)

const (
	openAIAPIURL = "https://api.openai.com/v1/chat/completions"
	systemPrompt = `You are an expert fraud detection analyst for an e-commerce platform.
Analyze the following transaction data. Your goal is to identify suspicious patterns.
High-risk indicators include: mismatch between billing and shipping countries, new users making large purchases, unusual IP addresses, or high transaction velocity.
Based on the data, provide a fraud assessment. Your response MUST be a valid JSON object with three keys:
1. "is_fraudulent": a boolean (true or false).
2. "fraud_score": an integer between 0 (low risk) and 100 (high risk).
3. "reason": a brief, clear explanation for your assessment.`
)

type FraudService interface {
	AnalyzeTransaction(ctx context.Context, request *models.FraudAnalysisRequest) (*models.FraudAnalysisResponse, error)
	GetStatsByDateRange(startDate, endDate time.Time) (*models.FraudStatsResponse, error)
}

type fraudService struct {
	repo       stores.FraudRepository
	openAIKey  string
	httpClient *http.Client
	ipAnalyzer *utils.IPAnalyzer
	cache      map[string]*models.FraudAnalysisResult
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func CreateFraudService(repo stores.FraudRepository, openAIKey string) FraudService {
	return &fraudService{
		repo:      repo,
		openAIKey: openAIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ipAnalyzer: utils.CreateIPAnalyzer(),
		cache:      make(map[string]*models.FraudAnalysisResult),
	}
}

func (s *fraudService) AnalyzeTransaction(ctx context.Context, request *models.FraudAnalysisRequest) (*models.FraudAnalysisResponse, error) {
	cacheKey := fmt.Sprintf("%s_%s_%s_%s", request.TransactionID, request.UserID, request.IPAddress, request.BillingCountry)

	if cached, exists := s.cache[cacheKey]; exists {
		utils.CreateLogger("conductor").Info(ctx, "Using cached fraud analysis result", map[string]interface{}{
			"transaction_id": request.TransactionID,
			"cache_key":      cacheKey,
		})
		return &models.FraudAnalysisResponse{
			Allow:  cached.Allow,
			Reason: cached.Reason,
		}, nil
	}

	ipRiskLevel := s.ipAnalyzer.AnalyzeIP(ctx, request.IPAddress)
	ipRiskScore := s.ipAnalyzer.GetRiskScore(ipRiskLevel)
	ipRiskDescription := s.ipAnalyzer.GetRiskDescription(ipRiskLevel)

	anonymizedData := map[string]interface{}{
		"transaction_amount":   request.TransactionAmount,
		"billing_country":      request.BillingCountry,
		"shipping_country":     request.ShippingCountry,
		"transaction_velocity": request.TransactionVelocity,
		"countries_match":      request.BillingCountry == request.ShippingCountry,
		"amount_category":      categorizeAmount(request.TransactionAmount),
		"ip_risk_level":        ipRiskLevel,
		"ip_risk_score":        ipRiskScore,
		"ip_risk_description":  ipRiskDescription,
	}

	userMessageData, err := json.Marshal(anonymizedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
	}

	assessment, err := s.callOpenAI(ctx, string(userMessageData))
	if err != nil {
		// If OpenAI fails, use fallback logic
		log.Printf("OpenAI API failed, using fallback logic: %v", err)
		assessment = s.fallbackFraudDetection(request)
	}

	allow := !assessment.IsFraudulent || assessment.FraudScore < 70

	result := &models.FraudAnalysisResult{
		TransactionID:       request.TransactionID,
		UserID:              request.UserID,
		TransactionAmount:   request.TransactionAmount,
		BillingCountry:      request.BillingCountry,
		ShippingCountry:     request.ShippingCountry,
		IPAddress:           request.IPAddress,
		TransactionVelocity: request.TransactionVelocity,
		IsFraudulent:        assessment.IsFraudulent,
		FraudScore:          assessment.FraudScore,
		Reason:              assessment.Reason,
		Allow:               allow,
	}

	if err := s.repo.SaveAnalysisResult(result); err != nil {
		utils.CreateLogger("conductor").Error(ctx, "Failed to save fraud analysis result", map[string]interface{}{
			"error": err.Error(),
		})
	}

	s.cache[cacheKey] = result

	response := &models.FraudAnalysisResponse{
		Allow:  allow,
		Reason: assessment.Reason,
	}

	return response, nil
}

func (s *fraudService) callOpenAI(ctx context.Context, transactionData string) (*models.OpenAIFraudAssessment, error) {
	requestBody := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: transactionData,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openAIAPIURL, bytes.NewBuffer(jsonData))
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

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	var assessment models.OpenAIFraudAssessment
	if err := json.Unmarshal([]byte(openAIResp.Choices[0].Message.Content), &assessment); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI fraud assessment: %w", err)
	}

	return &assessment, nil
}

func (s *fraudService) fallbackFraudDetection(request *models.FraudAnalysisRequest) *models.OpenAIFraudAssessment {
	fraudScore := 0
	reasons := []string{}

	if request.BillingCountry != request.ShippingCountry {
		fraudScore += 25
		reasons = append(reasons, "billing and shipping countries don't match")
	}

	if request.TransactionAmount > 1000 {
		fraudScore += 30
		reasons = append(reasons, "high transaction amount")
	}

	if request.TransactionVelocity > 5 {
		fraudScore += 35
		reasons = append(reasons, "high transaction velocity")
	}

	if request.TransactionAmount > 5000 {
		fraudScore += 20
		reasons = append(reasons, "extremely high transaction amount")
	}

	isFraudulent := fraudScore >= 50
	reason := "Low risk transaction"
	if isFraudulent {
		if len(reasons) > 0 {
			reason = fmt.Sprintf("High risk: %s", joinReasons(reasons))
		} else {
			reason = "High risk transaction detected"
		}
	}

	return &models.OpenAIFraudAssessment{
		IsFraudulent: isFraudulent,
		FraudScore:   fraudScore,
		Reason:       reason,
	}
}

func (s *fraudService) GetStatsByDateRange(startDate, endDate time.Time) (*models.FraudStatsResponse, error) {
	return s.repo.GetStatsByDateRange(startDate, endDate)
}

func categorizeAmount(amount float64) string {
	if amount < 50 {
		return "low"
	} else if amount < 500 {
		return "medium"
	} else if amount < 2000 {
		return "high"
	}
	return "very_high"
}

func categorizeIPAddress(ip string) string {
	if ip == "" {
		return "unknown"
	}
	return "normal"
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	if len(reasons) == 1 {
		return reasons[0]
	}

	result := ""
	for i, reason := range reasons {
		if i == len(reasons)-1 {
			result += " and " + reason
		} else if i == 0 {
			result = reason
		} else {
			result += ", " + reason
		}
	}
	return result
}
