package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"golang.org/x/time/rate"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic: %v\n%s", err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	limiter *rate.Limiter
	next    http.Handler
}

func (rl *rateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rl.limiter.Allow() {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	rl.next.ServeHTTP(w, r)
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Every(time.Second), 100)
	return &rateLimiter{
		limiter: limiter,
		next:    next,
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				apiKey = authHeader[7:]
			}
		}

		if apiKey == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "API key required. Provide X-API-Key header or Authorization: Bearer <key>",
			})
			return
		}

		if len(apiKey) < 10 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid API key format",
			})
			return
		}

		// Define a custom type for context keys to avoid collisions
		type contextKey string
		const apiKeyContextKey contextKey = "apiKey"
		ctx := context.WithValue(r.Context(), apiKeyContextKey, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
