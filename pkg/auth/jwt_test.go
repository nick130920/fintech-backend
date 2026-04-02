package auth

import (
	"testing"
	"time"
)

func TestJWTManager_RefreshTokenRoundTrip(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute)

	testUserIDs := []uint{1, 255, 1000, 65535}
	email := "test@example.com"

	for _, userID := range testUserIDs {
		token, err := manager.GenerateRefreshToken(userID, email)
		if err != nil {
			t.Fatalf("failed to generate refresh token for user %d: %v", userID, err)
		}

		gotUserID, gotEmail, err := manager.ValidateRefreshToken(token)
		if err != nil {
			t.Fatalf("failed to validate refresh token for user %d: %v", userID, err)
		}

		if gotUserID != userID {
			t.Fatalf("unexpected user ID. want=%d got=%d", userID, gotUserID)
		}

		if gotEmail != email {
			t.Fatalf("unexpected email. want=%s got=%s", email, gotEmail)
		}
	}
}

func TestJWTManager_AccessTokenRoundTrip(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute)

	token, err := manager.GenerateToken(42, "user@example.com", "Ada", "Lovelace")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("unexpected user ID. want=42 got=%d", claims.UserID)
	}

	if claims.Email != "user@example.com" {
		t.Fatalf("unexpected email. want=user@example.com got=%s", claims.Email)
	}
}
