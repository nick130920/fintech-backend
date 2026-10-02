package webapi

import (
	"context"
	"errors"
	"testing"

	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

func TestNewAIServiceWithFallback_UsesUnavailableServiceWithoutCredentials(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	service, err := NewAIServiceWithFallback()
	if err != nil {
		t.Fatalf("NewAIServiceWithFallback() error = %v; want nil", err)
	}
	if service == nil {
		t.Fatal("NewAIServiceWithFallback() returned nil service")
	}
	if service.HasProvider() {
		t.Fatal("HasProvider() = true; want false without configured credentials")
	}

	_, err = service.ExtractTransactionFromSMS(context.Background(), "purchase")
	if !errors.Is(err, apperrors.ErrAIServiceUnavailable) {
		t.Fatalf("ExtractTransactionFromSMS() error = %v; want AI service unavailable", err)
	}
}

func TestNewAIServiceWithFallback_UsesOpenRouterWhenConfigured(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("GEMINI_API_KEY", "")

	service, err := NewAIServiceWithFallback()
	if err != nil {
		t.Fatalf("NewAIServiceWithFallback() error = %v; want nil", err)
	}
	if !service.HasProvider() {
		t.Fatal("HasProvider() = false; want true with OpenRouter configured")
	}
	if service.GetUsedService() != "OpenRouter AI (Mistral)" {
		t.Fatalf("GetUsedService() = %q; want OpenRouter primary", service.GetUsedService())
	}
}
