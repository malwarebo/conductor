package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

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
