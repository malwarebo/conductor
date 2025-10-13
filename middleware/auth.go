package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/malwarebo/gopay/security"
)

type AuthMiddleware struct {
	jwtManager    *security.JWTManager
	rateLimiter   *security.TieredRateLimiter
	encryption    *security.EncryptionManager
	webhookSecret string
}

func CreateAuthMiddleware(jwtManager *security.JWTManager, rateLimiter *security.TieredRateLimiter, encryption *security.EncryptionManager, webhookSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:    jwtManager,
		rateLimiter:   rateLimiter,
		encryption:    encryption,
		webhookSecret: webhookSecret,
	}
}

func (am *AuthMiddleware) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := am.jwtManager.ValidateToken(token)
		if err != nil {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_roles", claims.Roles)
		ctx = context.WithValue(ctx, "api_key", claims.APIKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthMiddleware) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID == nil {
			userID = r.RemoteAddr
		}

		tier := am.getUserTier(r.Context())
		key := fmt.Sprintf("%s_%s", userID, r.URL.Path)

		if !am.rateLimiter.Allow(key, tier) {
			am.writeErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) WebhookMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/webhooks/") {
			next.ServeHTTP(w, r)
			return
		}

		signature := r.Header.Get("X-Signature")
		if signature == "" {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Webhook signature required")
			return
		}

		body, err := am.readRequestBody(r)
		if err != nil {
			am.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
			return
		}

		if !am.verifyWebhookSignature(body, signature) {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Invalid webhook signature")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) EncryptionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			body, err := am.readRequestBody(r)
			if err != nil {
				am.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
				return
			}

			encrypted, err := am.encryption.Encrypt(string(body))
			if err != nil {
				am.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encrypt data")
				return
			}

			ctx := context.WithValue(r.Context(), "encrypted_data", encrypted)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) HeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) getUserTier(ctx context.Context) string {
	roles := ctx.Value("user_roles")
	if roles == nil {
		return "default"
	}

	userRoles, ok := roles.([]string)
	if !ok {
		return "default"
	}

	for _, role := range userRoles {
		switch role {
		case "admin":
			return "premium"
		case "premium":
			return "premium"
		case "standard":
			return "standard"
		}
	}

	return "default"
}

func (am *AuthMiddleware) readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("request body is nil")
	}

	body := make([]byte, r.ContentLength)
	_, err := r.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return body, nil
}

func (am *AuthMiddleware) verifyWebhookSignature(body []byte, signature string) bool {
	expectedSignature := am.calculateSignature(body)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (am *AuthMiddleware) calculateSignature(body []byte) string {
	h := hmac.New(sha256.New, []byte(am.webhookSecret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (am *AuthMiddleware) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error":     message,
		"status":    fmt.Sprintf("%d", statusCode),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}
