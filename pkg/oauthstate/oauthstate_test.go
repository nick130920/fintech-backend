package oauthstate

import (
	"strings"
	"testing"
	"time"
)

func TestSignVerify_OK(t *testing.T) {
	const secret = "oauth-state-test-secret"
	st, err := Sign(42, 10*time.Minute, secret)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := Verify(st, secret)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 42 {
		t.Fatalf("uid want 42 got %d", uid)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	st, err := Sign(1, time.Hour, "secret-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Verify(st, "secret-b")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerify_Expired(t *testing.T) {
	st, err := Sign(99, -time.Hour, "secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Verify(st, "secret")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestVerify_InvalidFormat(t *testing.T) {
	_, err := Verify("not-a-state", "secret")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSign_EmptySecret(t *testing.T) {
	_, err := Sign(1, time.Minute, "")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}
