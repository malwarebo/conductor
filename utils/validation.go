package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

func (ve ValidationErrors) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"error":   "Validation failed",
		"details": ve,
	}
}

func ValidateString(value, fieldName string, minLen, maxLen int, required bool) *ValidationError {
	if required && strings.TrimSpace(value) == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	if value != "" {
		if utf8.RuneCountInString(value) < minLen {
			return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must be at least %d characters", minLen)}
		}
		if utf8.RuneCountInString(value) > maxLen {
			return &ValidationError{Field: fieldName, Message: fmt.Sprintf("must be at most %d characters", maxLen)}
		}
	}

	return nil
}

func ValidateAmount(amount int64, fieldName string) *ValidationError {
	if amount <= 0 {
		return &ValidationError{Field: fieldName, Message: "must be greater than 0"}
	}
	if amount > 100000000 {
		return &ValidationError{Field: fieldName, Message: "must be less than 100,000,000"}
	}
	return nil
}

func ValidateCurrency(currency, fieldName string) *ValidationError {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true,
		"IDR": true, "SGD": true, "MYR": true,
		"PHP": true, "THB": true, "VND": true,
	}

	if currency == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	if !validCurrencies[strings.ToUpper(currency)] {
		return &ValidationError{Field: fieldName, Message: "is not a supported currency"}
	}

	return nil
}

func ValidateEmail(email, fieldName string) *ValidationError {
	if email == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: fieldName, Message: "is not a valid email address"}
	}

	return nil
}

func ValidateUUID(id, fieldName string) *ValidationError {
	if id == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(strings.ToLower(id)) {
		return &ValidationError{Field: fieldName, Message: "is not a valid UUID"}
	}

	return nil
}

func ValidateCountryCode(code, fieldName string) *ValidationError {
	if code == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	if len(code) != 2 {
		return &ValidationError{Field: fieldName, Message: "must be a 2-letter country code"}
	}

	return nil
}

func ValidateIPAddress(ip, fieldName string) *ValidationError {
	if ip == "" {
		return &ValidationError{Field: fieldName, Message: "is required"}
	}

	ipRegex := regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	if !ipRegex.MatchString(ip) {
		return &ValidationError{Field: fieldName, Message: "is not a valid IP address"}
	}

	return nil
}

func ValidateRequestSize(r *http.Request, maxSize int64) error {
	if r.ContentLength > maxSize {
		return &APIError{
			Code:    http.StatusRequestEntityTooLarge,
			Message: "Request body too large",
		}
	}
	return nil
}

func ValidateJSONRequest(w http.ResponseWriter, r *http.Request, maxSize int64) error {
	if err := ValidateRequestSize(r, maxSize); err != nil {
		return err
	}

	if r.Header.Get("Content-Type") != "application/json" {
		return &APIError{
			Code:    http.StatusUnsupportedMediaType,
			Message: "Content-Type must be application/json",
		}
	}

	return nil
}

func WriteValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	if validationErr, ok := err.(ValidationErrors); ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(validationErr.ToJSON())
	} else if apiErr, ok := err.(*APIError); ok {
		w.WriteHeader(apiErr.Code)
		json.NewEncoder(w).Encode(apiErr)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
	}
}
