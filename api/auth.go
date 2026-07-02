package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/malwarebo/conductor/security"
	"github.com/malwarebo/conductor/services"
)

type AuthHandler struct {
	jwtManager    *security.JWTManager
	tenantService *services.TenantService
	tokenDuration time.Duration
}

func CreateAuthHandler(jwtManager *security.JWTManager, tenantService *services.TenantService, tokenDuration time.Duration) *AuthHandler {
	if tokenDuration <= 0 {
		tokenDuration = 24 * time.Hour
	}
	return &AuthHandler{
		jwtManager:    jwtManager,
		tenantService: tenantService,
		tokenDuration: tokenDuration,
	}
}

type tokenRequest struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// HandleToken exchanges a tenant API key/secret pair for a short-lived JWT that
// the rest of the /v1 API expects in the Authorization header.
func (h *AuthHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.APIKey == "" || req.APISecret == "" {
		writeAuthError(w, http.StatusBadRequest, "api_key and api_secret are required")
		return
	}

	tenant, err := h.tenantService.ValidateCredentials(r.Context(), req.APIKey, req.APISecret)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.jwtManager.GenerateToken(tenant.ID, tenant.Name, []string{"standard"}, tenant.APIKey, h.tokenDuration)
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.tokenDuration.Seconds()),
	})
}

func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
