package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJWTManager_RefreshTokenRoundTrip(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute)

	testUserIDs := []uint{1, 255, 1000, 65535}
	email := "test@example.com"

	jtis := map[string]struct{}{}
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

		_, _, _, jti, err := manager.ValidateRefreshTokenFull(token)
		if err != nil {
			t.Fatalf("ValidateRefreshTokenFull user %d: %v", userID, err)
		}
		if len(jti) != 32 {
			t.Fatalf("expected 32-char hex jti, got len=%d %q", len(jti), jti)
		}
		if _, dup := jtis[jti]; dup {
			t.Fatalf("duplicate jti %q for user %d", jti, userID)
		}
		jtis[jti] = struct{}{}
	}
}

func TestGenerateRefreshTokenPayloadHasUserIDAndJTI(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute)
	token, err := manager.GenerateRefreshToken(42, "u@example.com")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		t.Fatal("expected JWT with 3 segments")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["user_id"]; !ok {
		t.Fatalf("missing user_id in payload: %s", string(raw))
	}
	if jti, ok := payload["jti"].(string); !ok || len(jti) != 32 {
		t.Fatalf("expected 32-char hex jti claim, got %v", payload["jti"])
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
