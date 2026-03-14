package webapi

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// AIService defines the interface for AI-based transaction extraction.
type AIService interface {
	ExtractTransactionFromSMS(ctx context.Context, smsContent string) (*TransactionExtraction, error)
}

// AIServiceWithFallback wraps multiple AI services and provides fallback capability.
type AIServiceWithFallback struct {
	primary   AIService
	fallback  AIService
	logger    *zap.Logger
	usedService string
}

// NewAIServiceWithFallback creates a new AI service with fallback support.
// It tries to initialize OpenRouter first, then Gemini as fallback.
func NewAIServiceWithFallback() (*AIServiceWithFallback, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error al crear el logger: %w", err)
	}

	service := &AIServiceWithFallback{
		logger: logger,
	}

	// Try to initialize OpenRouter as primary
	openRouter, err := NewOpenRouterService()
	if err != nil {
		logger.Warn("OpenRouter no disponible, intentando Gemini como primario",
			zap.Error(err),
		)
	} else {
		service.primary = openRouter
		logger.Info("OpenRouter configurado como servicio primario de IA")
	}

	// Try to initialize Gemini as fallback (or primary if OpenRouter failed)
	gemini, err := NewGeminiService()
	if err != nil {
		logger.Warn("Gemini no disponible",
			zap.Error(err),
		)
	} else {
		if service.primary == nil {
			service.primary = gemini
			logger.Info("Gemini configurado como servicio primario de IA (OpenRouter no disponible)")
		} else {
			service.fallback = gemini
			logger.Info("Gemini configurado como servicio de fallback")
		}
	}

	// If neither service is available, return error
	if service.primary == nil {
		return nil, fmt.Errorf("ningún servicio de IA está disponible (configure OPENROUTER_API_KEY o GEMINI_API_KEY)")
	}

	return service, nil
}

// ExtractTransactionFromSMS extracts transaction data using the primary service,
// falling back to the secondary service if the primary fails.
func (s *AIServiceWithFallback) ExtractTransactionFromSMS(ctx context.Context, smsContent string) (*TransactionExtraction, error) {
	// Try primary service first
	extraction, err := s.primary.ExtractTransactionFromSMS(ctx, smsContent)
	if err == nil {
		s.usedService = s.getPrimaryServiceName()
		return extraction, nil
	}

	s.logger.Warn("Servicio primario de IA falló, intentando fallback",
		zap.Error(err),
	)

	// If fallback is available, try it
	if s.fallback != nil {
		extraction, fallbackErr := s.fallback.ExtractTransactionFromSMS(ctx, smsContent)
		if fallbackErr == nil {
			s.usedService = s.getFallbackServiceName()
			s.logger.Info("Fallback de IA exitoso",
				zap.String("service", s.usedService),
			)
			return extraction, nil
		}

		s.logger.Error("Fallback de IA también falló",
			zap.Error(fallbackErr),
		)
		return nil, fmt.Errorf("ambos servicios de IA fallaron: primario=%v, fallback=%v", err, fallbackErr)
	}

	return nil, fmt.Errorf("servicio primario de IA falló y no hay fallback disponible: %w", err)
}

// GetUsedService returns the name of the service that was last used successfully.
func (s *AIServiceWithFallback) GetUsedService() string {
	if s.usedService == "" {
		return s.getPrimaryServiceName()
	}
	return s.usedService
}

func (s *AIServiceWithFallback) getPrimaryServiceName() string {
	switch s.primary.(type) {
	case *OpenRouterService:
		return "OpenRouter AI (Mistral)"
	case *GeminiService:
		return "Google Gemini"
	default:
		return "Unknown AI Service"
	}
}

func (s *AIServiceWithFallback) getFallbackServiceName() string {
	if s.fallback == nil {
		return "None"
	}
	switch s.fallback.(type) {
	case *OpenRouterService:
		return "OpenRouter AI (Mistral)"
	case *GeminiService:
		return "Google Gemini"
	default:
		return "Unknown AI Service"
	}
}

// HasFallback returns true if a fallback service is configured.
func (s *AIServiceWithFallback) HasFallback() bool {
	return s.fallback != nil
}
