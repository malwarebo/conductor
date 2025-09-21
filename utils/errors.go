package utils

import (
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
	ErrInternalServer     = NewAPIError(http.StatusInternalServerError, "Internal server error")
	ErrServiceUnavailable = NewAPIError(http.StatusServiceUnavailable, "Service unavailable")
	ErrTooManyRequests    = NewAPIError(http.StatusTooManyRequests, "Too many requests")
)

func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

func WrapErrorWithCode(err error, code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: err.Error(),
	}
}
