package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/malwarebo/conductor/services"
)

type TenantMiddleware struct {
	tenantService *services.TenantService
	auditService  *services.AuditService
}

func CreateTenantMiddleware(tenantService *services.TenantService, auditService *services.AuditService) *TenantMiddleware {
	return &TenantMiddleware{
		tenantService: tenantService,
		auditService:  auditService,
	}
}

func (tm *TenantMiddleware) TenantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := tm.extractAPIKey(r)
		if apiKey == "" {
			tm.writeErrorResponse(w, http.StatusUnauthorized, "API key required")
			return
		}

		tenant, err := tm.tenantService.GetByAPIKey(r.Context(), apiKey)
		if err != nil {
			tm.writeErrorResponse(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		ctx := context.WithValue(r.Context(), "tenant_id", tenant.ID)
		ctx = context.WithValue(ctx, "tenant", tenant)
		ctx = context.WithValue(ctx, "api_key", apiKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (tm *TenantMiddleware) AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tm.auditService == nil {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		tenantID := ""
		if tid := r.Context().Value("tenant_id"); tid != nil {
			tenantID = tid.(string)
		}

		userID := ""
		if uid := r.Context().Value("user_id"); uid != nil {
			userID = uid.(string)
		}

		go tm.auditService.LogAPIRequest(
			context.Background(),
			tenantID,
			userID,
			r.Method,
			r.URL.Path,
			getClientIP(r),
			r.UserAgent(),
			nil,
			rw.statusCode,
			rw.statusCode < 400,
			"",
		)

		_ = start
	})
}

func (tm *TenantMiddleware) IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")
		if idempotencyKey != "" {
			ctx := context.WithValue(r.Context(), "idempotency_key", idempotencyKey)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func (tm *TenantMiddleware) extractAPIKey(r *http.Request) string {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

func (tm *TenantMiddleware) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func isPublicPath(path string) bool {
	publicPaths := []string{
		"/v1/health",
		"/v1/webhooks/stripe",
		"/v1/webhooks/xendit",
	}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}


