package routing

import (
	"context"
	"errors"
	"time"

	"github.com/malwarebo/conductor/models"
)

var (
	ErrNoEligibleProviders = errors.New("no eligible providers available")
	ErrMaxRetriesExceeded  = errors.New("maximum retry attempts exceeded")
	ErrAllProvidersFailed  = errors.New("all providers failed")
)

type RetryStrategy int

const (
	RetryStrategySameProvider RetryStrategy = iota
	RetryStrategyNextProvider
	RetryStrategySmartFallback
)

type RetryConfig struct {
	MaxAttempts       int
	Strategy          RetryStrategy
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	RetryableCodes    []string
	NonRetryableCodes []string
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		Strategy:          RetryStrategySmartFallback,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          2 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []string{"timeout", "rate_limit", "temporary_failure", "network_error"},
		NonRetryableCodes: []string{"invalid_card", "insufficient_funds", "expired_card", "fraud", "declined"},
	}
}

type RetryManager struct {
	engine *Engine
	config RetryConfig
}

func NewRetryManager(engine *Engine, config RetryConfig) *RetryManager {
	return &RetryManager{
		engine: engine,
		config: config,
	}
}

type PaymentFunc func(ctx context.Context, provider string) (*PaymentResult, error)

type PaymentResult struct {
	Success      bool
	ProviderID   string
	ErrorCode    string
	ErrorMessage string
	ResponseTime int64
}

func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, decision *models.RoutingDecision, fn PaymentFunc) (*PaymentResult, *models.RoutingDecision, error) {
	providers := rm.buildProviderList(decision)
	attempts := make([]models.AttemptResult, 0)

	for attempt := 0; attempt < rm.config.MaxAttempts && len(providers) > 0; attempt++ {
		provider := providers[0]

		if attempt > 0 {
			delay := rm.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, decision, ctx.Err()
			case <-time.After(delay):
			}
		}

		start := time.Now()
		result, err := fn(ctx, provider)
		responseTime := time.Since(start).Milliseconds()

		attemptResult := models.AttemptResult{
			Provider:       provider,
			ResponseTimeMs: responseTime,
			Timestamp:      time.Now(),
		}

		if err != nil {
			attemptResult.Success = false
			attemptResult.ErrorMessage = err.Error()
		} else if result != nil {
			attemptResult.Success = result.Success
			attemptResult.ErrorCode = result.ErrorCode
			attemptResult.ErrorMessage = result.ErrorMessage
		}

		attempts = append(attempts, attemptResult)

		if result != nil && result.Success {
			decision.PreviousAttempts = attempts
			decision.Attempt = attempt + 1
			return result, decision, nil
		}

		if result != nil && !rm.isRetryable(result.ErrorCode) {
			decision.PreviousAttempts = attempts
			return result, decision, nil
		}

		providers = rm.selectNextProviders(providers, result)
	}

	decision.PreviousAttempts = attempts
	return nil, decision, ErrAllProvidersFailed
}

func (rm *RetryManager) buildProviderList(decision *models.RoutingDecision) []string {
	providers := []string{decision.SelectedProvider}
	providers = append(providers, decision.FallbackProviders...)
	return providers
}

func (rm *RetryManager) selectNextProviders(current []string, lastResult *PaymentResult) []string {
	if len(current) <= 1 {
		return nil
	}

	switch rm.config.Strategy {
	case RetryStrategySameProvider:
		return current
	case RetryStrategyNextProvider:
		return current[1:]
	case RetryStrategySmartFallback:
		if lastResult != nil && rm.isSoftDecline(lastResult.ErrorCode) {
			return current[1:]
		}
		if lastResult != nil && rm.isHardDecline(lastResult.ErrorCode) {
			return nil
		}
		return current[1:]
	}

	return current[1:]
}

func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	delay := rm.config.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rm.config.BackoffMultiplier)
		if delay > rm.config.MaxDelay {
			delay = rm.config.MaxDelay
			break
		}
	}
	return delay
}

func (rm *RetryManager) isRetryable(errorCode string) bool {
	for _, code := range rm.config.NonRetryableCodes {
		if code == errorCode {
			return false
		}
	}

	for _, code := range rm.config.RetryableCodes {
		if code == errorCode {
			return true
		}
	}

	return true
}

func (rm *RetryManager) isSoftDecline(errorCode string) bool {
	softDeclines := map[string]bool{
		"insufficient_funds": true,
		"temporary_failure":  true,
		"rate_limit":         true,
		"network_error":      true,
		"timeout":            true,
	}
	return softDeclines[errorCode]
}

func (rm *RetryManager) isHardDecline(errorCode string) bool {
	hardDeclines := map[string]bool{
		"invalid_card":   true,
		"expired_card":   true,
		"fraud":          true,
		"stolen_card":    true,
		"lost_card":      true,
		"do_not_honor":   true,
	}
	return hardDeclines[errorCode]
}

type ErrorClassifier struct {
	providerMappings map[string]map[string]string
}

func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		providerMappings: map[string]map[string]string{
			"stripe": {
				"card_declined":          "declined",
				"expired_card":           "expired_card",
				"incorrect_cvc":          "invalid_card",
				"processing_error":       "temporary_failure",
				"insufficient_funds":     "insufficient_funds",
				"fraudulent":             "fraud",
				"rate_limit":             "rate_limit",
			},
			"razorpay": {
				"BAD_REQUEST_ERROR":      "invalid_card",
				"GATEWAY_ERROR":          "temporary_failure",
				"SERVER_ERROR":           "temporary_failure",
				"insufficient_balance":   "insufficient_funds",
			},
			"xendit": {
				"INVALID_ACCOUNT":        "invalid_card",
				"INSUFFICIENT_BALANCE":   "insufficient_funds",
				"API_VALIDATION_ERROR":   "invalid_card",
				"SERVER_ERROR":           "temporary_failure",
			},
			"airwallex": {
				"invalid_request":        "invalid_card",
				"insufficient_funds":     "insufficient_funds",
				"authentication_failed":  "fraud",
				"processing_error":       "temporary_failure",
			},
		},
	}
}

func (ec *ErrorClassifier) Classify(provider, providerErrorCode string) string {
	if mapping, ok := ec.providerMappings[provider]; ok {
		if normalized, ok := mapping[providerErrorCode]; ok {
			return normalized
		}
	}
	return "unknown_error"
}

func (ec *ErrorClassifier) IsRetryable(provider, providerErrorCode string) bool {
	normalized := ec.Classify(provider, providerErrorCode)
	retryable := map[string]bool{
		"temporary_failure": true,
		"rate_limit":        true,
		"network_error":     true,
		"timeout":           true,
	}
	return retryable[normalized]
}
