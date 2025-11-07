package security

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	manager := CreateJWTManager("test-secret-key-32-bytes-long!!", "gopay-test", "gopay-api")

	tests := []struct {
		name   string
		userID string
		email  string
		roles  []string
	}{
		{
			name:   "Single role",
			userID: "user123",
			email:  "user@test.com",
			roles:  []string{"admin"},
		},
		{
			name:   "Multiple roles",
			userID: "user456",
			email:  "admin@test.com",
			roles:  []string{"admin", "user"},
		},
		{
			name:   "No roles",
			userID: "user789",
			email:  "test@test.com",
			roles:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.userID, tt.email, tt.roles, "api_key_test", 24*time.Hour)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}
			if token == "" {
				t.Error("GenerateToken() returned empty token")
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := CreateJWTManager("test-secret-key-32-bytes-long!!", "gopay-test", "gopay-api")

	t.Run("Valid token", func(t *testing.T) {
		userID := "user123"
		email := "user@test.com"
		roles := []string{"admin"}

		token, err := manager.GenerateToken(userID, email, roles, "api_key_test", 24*time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		claims, err := manager.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("ValidateToken() userID = %v, want %v", claims.UserID, userID)
		}
		if claims.Email != email {
			t.Errorf("ValidateToken() email = %v, want %v", claims.Email, email)
		}
		if len(claims.Roles) != len(roles) {
			t.Errorf("ValidateToken() roles length = %v, want %v", len(claims.Roles), len(roles))
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := manager.ValidateToken("invalid.token.here")
		if err == nil {
			t.Error("ValidateToken() expected error for invalid token")
		}
	})

	t.Run("Tampered token", func(t *testing.T) {
		token, err := manager.GenerateToken("user123", "user@test.com", []string{"admin"}, "api_key_test", 24*time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		tamperedToken := token[:len(token)-5] + "XXXXX"
		_, err = manager.ValidateToken(tamperedToken)
		if err == nil {
			t.Error("ValidateToken() expected error for tampered token")
		}
	})

	t.Run("Wrong secret", func(t *testing.T) {
		manager1 := CreateJWTManager("secret1-key-32-bytes-long!!!!", "gopay-test", "gopay-api")
		manager2 := CreateJWTManager("secret2-key-32-bytes-long!!!!", "gopay-test", "gopay-api")

		token, err := manager1.GenerateToken("user123", "user@test.com", []string{"admin"}, "api_key_test", 24*time.Hour)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		_, err = manager2.ValidateToken(token)
		if err == nil {
			t.Error("ValidateToken() expected error for wrong secret")
		}
	})
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	manager := CreateJWTManager("test-secret-key-32-bytes-long!!", "gopay-test", "gopay-api")

	token, err := manager.GenerateToken("user123", "user@test.com", []string{"admin"}, "api_key_test", -1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() expected error for expired token")
	}
}

func TestJWTManager_EmptyUserID(t *testing.T) {
	manager := CreateJWTManager("test-secret-key-32-bytes-long!!", "gopay-test", "gopay-api")

	token, err := manager.GenerateToken("", "user@test.com", []string{"admin"}, "api_key_test", 24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != "" {
		t.Errorf("ValidateToken() userID = %v, want empty string", claims.UserID)
	}
}
