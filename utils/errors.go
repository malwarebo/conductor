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

func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func NewAPIErrorWithDetails(code int, message, details string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

var (
	ErrInvalidRequest     = NewAPIError(http.StatusBadRequest, "Invalid request")
	ErrUnauthorized       = NewAPIError(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden          = NewAPIError(http.StatusForbidden, "Forbidden")
	ErrNotFound           = NewAPIError(http.StatusNotFound, "Resource not found")
	ErrConflict           = NewAPIError(http.StatusConflict, "Resource conflict")
	ErrTooManyRequests    = NewAPIError(http.StatusTooManyRequests, "Too many requests")
	ErrInternalServer     = NewAPIError(http.StatusInternalServerError, "Internal server error")
	ErrServiceUnavailable = NewAPIError(http.StatusServiceUnavailable, "Service unavailable")
	ErrGatewayTimeout     = NewAPIError(http.StatusGatewayTimeout, "Gateway timeout")
)

var (
	ErrInvalidAmount        = NewAPIError(http.StatusBadRequest, "Invalid amount")
	ErrInvalidCurrency      = NewAPIError(http.StatusBadRequest, "Invalid currency")
	ErrInvalidPaymentMethod = NewAPIError(http.StatusBadRequest, "Invalid payment method")
	ErrInvalidCustomerID    = NewAPIError(http.StatusBadRequest, "Invalid customer ID")
	ErrInvalidPaymentID     = NewAPIError(http.StatusBadRequest, "Invalid payment ID")
	ErrInvalidRefundAmount  = NewAPIError(http.StatusBadRequest, "Invalid refund amount")
	ErrRefundExceedsPayment = NewAPIError(http.StatusBadRequest, "Refund amount exceeds payment amount")
	ErrPaymentNotFound      = NewAPIError(http.StatusNotFound, "Payment not found")
	ErrRefundNotFound       = NewAPIError(http.StatusNotFound, "Refund not found")
	ErrSubscriptionNotFound = NewAPIError(http.StatusNotFound, "Subscription not found")
	ErrPlanNotFound         = NewAPIError(http.StatusNotFound, "Plan not found")
	ErrDisputeNotFound      = NewAPIError(http.StatusNotFound, "Dispute not found")
)

var (
	ErrNoAvailableProvider = NewAPIError(http.StatusServiceUnavailable, "No payment provider available")
	ErrProviderUnavailable = NewAPIError(http.StatusServiceUnavailable, "Payment provider unavailable")
	ErrProviderTimeout     = NewAPIError(http.StatusGatewayTimeout, "Payment provider timeout")
	ErrProviderError       = NewAPIError(http.StatusBadGateway, "Payment provider error")
)

var (
	ErrDatabaseConnection  = NewAPIError(http.StatusServiceUnavailable, "Database connection failed")
	ErrDatabaseQuery       = NewAPIError(http.StatusInternalServerError, "Database query failed")
	ErrDatabaseTransaction = NewAPIError(http.StatusInternalServerError, "Database transaction failed")
	ErrDatabaseTimeout     = NewAPIError(http.StatusGatewayTimeout, "Database timeout")
)

var (
	ErrEncryptionFailed  = NewAPIError(http.StatusInternalServerError, "Encryption failed")
	ErrDecryptionFailed  = NewAPIError(http.StatusInternalServerError, "Decryption failed")
	ErrInvalidSignature  = NewAPIError(http.StatusUnauthorized, "Invalid signature")
	ErrTokenExpired      = NewAPIError(http.StatusUnauthorized, "Token expired")
	ErrInvalidToken      = NewAPIError(http.StatusUnauthorized, "Invalid token")
	ErrRateLimitExceeded = NewAPIError(http.StatusTooManyRequests, "Rate limit exceeded")
)

var (
	ErrFraudDetectionFailed = NewAPIError(http.StatusInternalServerError, "Fraud detection failed")
	ErrFraudRiskHigh        = NewAPIError(http.StatusForbidden, "Transaction blocked due to fraud risk")
	ErrFraudAnalysisTimeout = NewAPIError(http.StatusGatewayTimeout, "Fraud analysis timeout")
)

var (
	ErrWebhookInvalidSignature = NewAPIError(http.StatusUnauthorized, "Invalid webhook signature")
	ErrWebhookInvalidPayload   = NewAPIError(http.StatusBadRequest, "Invalid webhook payload")
	ErrWebhookProcessingFailed = NewAPIError(http.StatusInternalServerError, "Webhook processing failed")
)

func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

func WrapAPIError(err error, apiErr *APIError) error {
	return fmt.Errorf("%s: %w", apiErr.Message, err)
}

func IsRetryableError(err error) bool {
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

func GetHTTPStatusFromError(err error) int {
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

func LogError(ctx context.Context, err error, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	fields["error"] = err.Error()
	fields["message"] = message

	Error(ctx, message, fields)
}

func LogAPIError(ctx context.Context, err *APIError, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	fields["error_code"] = err.Code
	fields["error_message"] = err.Message
	fields["error_details"] = err.Details
	fields["message"] = message

	Error(ctx, message, fields)
}
