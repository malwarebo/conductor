package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type JWTManager struct {
	secretKey string
	issuer    string
	audience  string
}

type Claims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	APIKey    string   `json:"api_key"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Issuer    string   `json:"iss"`
	Audience  string   `json:"aud"`
}

func CreateJWTManager(secretKey, issuer, audience string) *JWTManager {
	return &JWTManager{
		secretKey: secretKey,
		issuer:    issuer,
		audience:  audience,
	}
}

func (j *JWTManager) GenerateToken(userID, email string, roles []string, apiKey string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		APIKey:    apiKey,
		ExpiresAt: now.Add(duration).Unix(),
		IssuedAt:  now.Unix(),
		Issuer:    j.issuer,
		Audience:  j.audience,
	}

	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	message := headerB64 + "." + claimsB64
	signature := j.sign(message)

	return message + "." + signature, nil
}

func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	message := parts[0] + "." + parts[1]
	expectedSignature := j.sign(message)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %v", err)
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

func (j *JWTManager) RefreshToken(tokenString string, duration time.Duration) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token for refresh: %v", err)
	}

	return j.GenerateToken(claims.UserID, claims.Email, claims.Roles, claims.APIKey, duration)
}

func (j *JWTManager) sign(message string) string {
	h := hmac.New(sha256.New, []byte(j.secretKey))
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
