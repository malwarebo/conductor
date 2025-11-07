package utils

import (
	"context"
	"fmt"
	"net/http"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

func CreateAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func CreateAPIErrorWithDetails(code int, message, details string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

var (
	ErrInvalidRequest     = CreateAPIError(http.StatusBadRequest, "Invalid request")
	ErrUnauthorized       = CreateAPIError(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden          = CreateAPIError(http.StatusForbidden, "Forbidden")
	ErrNotFound           = CreateAPIError(http.StatusNotFound, "Resource not found")
	ErrConflict           = CreateAPIError(http.StatusConflict, "Resource conflict")
	ErrTooManyRequests    = CreateAPIError(http.StatusTooManyRequests, "Too many requests")
	ErrInternalServer     = CreateAPIError(http.StatusInternalServerError, "Internal server error")
	ErrServiceUnavailable = CreateAPIError(http.StatusServiceUnavailable, "Service unavailable")
	ErrGatewayTimeout     = CreateAPIError(http.StatusGatewayTimeout, "Gateway timeout")
)

var (
	ErrInvalidAmount        = CreateAPIError(http.StatusBadRequest, "Invalid amount")
	ErrInvalidCurrency      = CreateAPIError(http.StatusBadRequest, "Invalid currency")
	ErrInvalidPaymentMethod = CreateAPIError(http.StatusBadRequest, "Invalid payment method")
	ErrInvalidCustomerID    = CreateAPIError(http.StatusBadRequest, "Invalid customer ID")
	ErrInvalidPaymentID     = CreateAPIError(http.StatusBadRequest, "Invalid payment ID")
	ErrInvalidRefundAmount  = CreateAPIError(http.StatusBadRequest, "Invalid refund amount")
	ErrRefundExceedsPayment = CreateAPIError(http.StatusBadRequest, "Refund amount exceeds payment amount")
	ErrPaymentNotFound      = CreateAPIError(http.StatusNotFound, "Payment not found")
	ErrRefundNotFound       = CreateAPIError(http.StatusNotFound, "Refund not found")
	ErrSubscriptionNotFound = CreateAPIError(http.StatusNotFound, "Subscription not found")
	ErrPlanNotFound         = CreateAPIError(http.StatusNotFound, "Plan not found")
	ErrDisputeNotFound      = CreateAPIError(http.StatusNotFound, "Dispute not found")
)

var (
	ErrNoAvailableProvider = CreateAPIError(http.StatusServiceUnavailable, "No payment provider available")
	ErrProviderUnavailable = CreateAPIError(http.StatusServiceUnavailable, "Payment provider unavailable")
	ErrProviderTimeout     = CreateAPIError(http.StatusGatewayTimeout, "Payment provider timeout")
	ErrProviderError       = CreateAPIError(http.StatusBadGateway, "Payment provider error")
)

var (
	ErrDatabaseConnection  = CreateAPIError(http.StatusServiceUnavailable, "Database connection failed")
	ErrDatabaseQuery       = CreateAPIError(http.StatusInternalServerError, "Database query failed")
	ErrDatabaseTransaction = CreateAPIError(http.StatusInternalServerError, "Database transaction failed")
	ErrDatabaseTimeout     = CreateAPIError(http.StatusGatewayTimeout, "Database timeout")
)

var (
	ErrEncryptionFailed  = CreateAPIError(http.StatusInternalServerError, "Encryption failed")
	ErrDecryptionFailed  = CreateAPIError(http.StatusInternalServerError, "Decryption failed")
	ErrInvalidSignature  = CreateAPIError(http.StatusUnauthorized, "Invalid signature")
	ErrTokenExpired      = CreateAPIError(http.StatusUnauthorized, "Token expired")
	ErrInvalidToken      = CreateAPIError(http.StatusUnauthorized, "Invalid token")
	ErrRateLimitExceeded = CreateAPIError(http.StatusTooManyRequests, "Rate limit exceeded")
)

var (
	ErrFraudDetectionFailed = CreateAPIError(http.StatusInternalServerError, "Fraud detection failed")
	ErrFraudRiskHigh        = CreateAPIError(http.StatusForbidden, "Transaction blocked due to fraud risk")
	ErrFraudAnalysisTimeout = CreateAPIError(http.StatusGatewayTimeout, "Fraud analysis timeout")
)

var (
	ErrWebhookInvalidSignature = CreateAPIError(http.StatusUnauthorized, "Invalid webhook signature")
	ErrWebhookInvalidPayload   = CreateAPIError(http.StatusBadRequest, "Invalid webhook payload")
	ErrWebhookProcessingFailed = CreateAPIError(http.StatusInternalServerError, "Webhook processing failed")
)

func CreateWrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

func CreateWrapAPIError(err error, apiErr *APIError) error {
	return fmt.Errorf("%s: %w", apiErr.Message, err)
}

func CreateIsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"service unavailable",
		"gateway timeout",
		"too many requests",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errorStr, retryableErr) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:], substr))))
}

func CreateGetHTTPStatusFromError(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code
	}

	errorStr := err.Error()
	if contains(errorStr, "not found") {
		return http.StatusNotFound
	}
	if contains(errorStr, "unauthorized") {
		return http.StatusUnauthorized
	}
	if contains(errorStr, "forbidden") {
		return http.StatusForbidden
	}
	if contains(errorStr, "timeout") {
		return http.StatusGatewayTimeout
	}
	if contains(errorStr, "rate limit") {
		return http.StatusTooManyRequests
	}

	return http.StatusInternalServerError
}

func CreateLogError(ctx context.Context, err error, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	fields["error"] = err.Error()
	fields["message"] = message

	CreateLogger("conductor").Error(ctx, message, fields)
}

func CreateLogAPIError(ctx context.Context, err *APIError, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	fields["error_code"] = err.Code
	fields["error_message"] = err.Message
	fields["error_details"] = err.Details
	fields["message"] = message

	CreateLogger("conductor").Error(ctx, message, fields)
}
