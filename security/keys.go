package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	UserID     string     `json:"user_id"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	IsActive   bool       `json:"is_active"`
	RateLimit  int        `json:"rate_limit"`
}

type APIKeyManager struct {
	keys map[string]*APIKey
}

func CreateAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys: make(map[string]*APIKey),
	}
}

func (m *APIKeyManager) GenerateKey(name, userID string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}

	key := fmt.Sprintf("gopay_%s", hex.EncodeToString(keyBytes))
	keyHash := m.hashKey(key)
	keyPrefix := key[:12]

	apiKey := &APIKey{
		ID:        fmt.Sprintf("key_%d", time.Now().Unix()),
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    scopes,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		IsActive:  true,
		RateLimit: 1000,
	}

	m.keys[keyHash] = apiKey

	return apiKey, key, nil
}

func (m *APIKeyManager) ValidateKey(key string) (*APIKey, error) {
	keyHash := m.hashKey(key)
	apiKey, exists := m.keys[keyHash]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	if !apiKey.IsActive {
		return nil, fmt.Errorf("API key is inactive")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	now := time.Now()
	apiKey.LastUsedAt = &now

	return apiKey, nil
}

func (m *APIKeyManager) RevokeKey(keyID string) error {
	for _, key := range m.keys {
		if key.ID == keyID {
			key.IsActive = false
			return nil
		}
	}
	return fmt.Errorf("API key not found")
}

func (m *APIKeyManager) RotateKey(keyID string) (*APIKey, string, error) {
	for _, key := range m.keys {
		if key.ID == keyID {
			key.IsActive = false
			return m.GenerateKey(key.Name, key.UserID, key.Scopes, key.ExpiresAt)
		}
	}
	return nil, "", fmt.Errorf("API key not found")
}

func (m *APIKeyManager) ListKeys(userID string) []*APIKey {
	var userKeys []*APIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			userKeys = append(userKeys, key)
		}
	}
	return userKeys
}

func (m *APIKeyManager) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (m *APIKeyManager) CheckScope(apiKey *APIKey, requiredScope string) bool {
	for _, scope := range apiKey.Scopes {
		if scope == requiredScope || scope == "*" {
			return true
		}
	}
	return false
}
