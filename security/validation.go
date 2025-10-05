package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

type Validator struct {
	errors []ValidationError
}

func CreateValidator() *Validator {
	return &Validator{
		errors: make([]ValidationError, 0),
	}
}

func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *Validator) GetErrors() []ValidationError {
	return v.errors
}

func (v *Validator) Validate() error {
	if v.HasErrors() {
		return fmt.Errorf("validation failed: %d errors", len(v.errors))
	}
	return nil
}

func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "is required")
	}
}

func (v *Validator) MinLength(field, value string, min int) {
	if len(value) < min {
		v.AddError(field, fmt.Sprintf("must be at least %d characters", min))
	}
}

func (v *Validator) MaxLength(field, value string, max int) {
	if len(value) > max {
		v.AddError(field, fmt.Sprintf("must be no more than %d characters", max))
	}
}

func (v *Validator) Email(field, value string) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.AddError(field, "must be a valid email address")
	}
}

func (v *Validator) Positive(field string, value int64) {
	if value <= 0 {
		v.AddError(field, "must be positive")
	}
}

func (v *Validator) Range(field string, value, min, max int64) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

func (v *Validator) Currency(field, value string) {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"IDR": true, "SGD": true, "MYR": true, "PHP": true,
		"THB": true, "VND": true, "AUD": true, "CAD": true,
	}

	if !validCurrencies[strings.ToUpper(value)] {
		v.AddError(field, "must be a valid currency code")
	}
}

func (v *Validator) UUID(field, value string) {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(strings.ToLower(value)) {
		v.AddError(field, "must be a valid UUID")
	}
}

func (v *Validator) IPAddress(field, value string) {
	ipRegex := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	if !ipRegex.MatchString(value) {
		v.AddError(field, "must be a valid IP address")
	}
}

func (v *Validator) CountryCode(field, value string) {
	countryRegex := regexp.MustCompile(`^[A-Z]{2}$`)
	if !countryRegex.MatchString(strings.ToUpper(value)) {
		v.AddError(field, "must be a valid 2-letter country code")
	}
}

func (v *Validator) Date(field, value string) {
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		v.AddError(field, "must be a valid date in YYYY-MM-DD format")
	}
}

func (v *Validator) DateTime(field, value string) {
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		v.AddError(field, "must be a valid datetime in RFC3339 format")
	}
}

func (v *Validator) PaymentMethod(field, value string) {
	validMethods := map[string]bool{
		"card": true, "bank_transfer": true, "ewallet": true,
		"crypto": true, "cash": true, "check": true,
	}

	if !validMethods[strings.ToLower(value)] {
		v.AddError(field, "must be a valid payment method")
	}
}

func (v *Validator) Status(field, value string, validStatuses []string) {
	validMap := make(map[string]bool)
	for _, status := range validStatuses {
		validMap[strings.ToLower(status)] = true
	}

	if !validMap[strings.ToLower(value)] {
		v.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(validStatuses, ", ")))
	}
}

func (v *Validator) Alphanumeric(field, value string) {
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumericRegex.MatchString(value) {
		v.AddError(field, "must contain only alphanumeric characters")
	}
}

func (v *Validator) NoSpecialChars(field, value string) {
	specialCharRegex := regexp.MustCompile(`[<>\"'%;()&+]`)
	if specialCharRegex.MatchString(value) {
		v.AddError(field, "must not contain special characters")
	}
}

func CreateGenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %v", err)
	}
	return hex.EncodeToString(bytes), nil
}

func CreateHashAPIKey(key string) string {
	return hex.EncodeToString([]byte(key))
}

func CreateValidateAPIKey(key string) error {
	if len(key) < 32 {
		return fmt.Errorf("API key must be at least 32 characters")
	}

	hexRegex := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !hexRegex.MatchString(key) {
		return fmt.Errorf("API key must contain only hexadecimal characters")
	}

	return nil
}

func CreateSanitizeInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	return input
}

func CreateValidateWebhookSignature(payload, signature, secret string) bool {
	expectedSignature := calculateHMAC(payload, secret)
	return signature == expectedSignature
}

func calculateHMAC(payload, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
