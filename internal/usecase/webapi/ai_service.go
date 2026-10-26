package webapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"go.uber.org/zap"
)

// AIService defines the interface for AI-based transaction extraction.
type AIService interface {
	ExtractTransactionFromSMS(ctx context.Context, smsContent string) (*TransactionExtraction, error)
	// ExtractBudgetLinesFromSMSChunk analyzes many numbered SMS in one request (budget suggestions only).
	ExtractBudgetLinesFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSBudgetResponse, error)
	// ExtractTransactionsFromSMSChunk analyzes numbered SMS in one request (crear movimientos).
	ExtractTransactionsFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSTransactionResponse, error)
}

// AIServiceWithFallback wraps multiple AI services and provides fallback capability.
type AIServiceWithFallback struct {
	primary     AIService
	fallback    AIService
	logger      *zap.Logger
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

	if service.primary == nil {
		service.primary = unavailableAIService{}
		logger.Warn("No AI provider configured; AI-dependent endpoints are unavailable")
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

// ExtractBudgetLinesFromSMSChunk requires OpenRouter's strict ZDR budget route.
// Direct providers, including Gemini promoted to primary, cannot satisfy it.
func (s *AIServiceWithFallback) ExtractBudgetLinesFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSBudgetResponse, error) {
	primary, ok := s.primary.(*OpenRouterService)
	if !ok || primary == nil {
		return nil, apperrors.ErrAIServiceUnavailable
	}
	resp, err := primary.ExtractBudgetLinesFromSMSChunk(ctx, numberedSMSBlock)
	if err == nil {
		s.usedService = s.getPrimaryServiceName()
		return resp, nil
	}
	// Providers are a trust boundary: never log or return their raw errors.
	err = sanitizeBudgetBatchError(err)
	s.logger.Warn("Batch extracción: primario falló; ZDR obligatorio, sin fallback", zap.Error(err))
	return nil, err
}

// budgetBatchError retains only a fixed local classification and a context sentinel.
// In particular, it never unwraps the original provider error, which may contain
// SMS, model output, credentials, or request URLs.
type budgetBatchError struct {
	class      string
	contextErr error
}

func (e *budgetBatchError) Error() string { return e.class }
func (e *budgetBatchError) Unwrap() error { return e.contextErr }

func newBudgetBatchError(class string, err error) error {
	var contextErr error
	switch {
	case errors.Is(err, context.Canceled):
		contextErr = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		contextErr = context.DeadlineExceeded
	}
	return &budgetBatchError{class: class, contextErr: contextErr}
}

func sanitizeBudgetBatchError(err error) error {
	// Only errors constructed locally can carry a trusted classification. Do not
	// use errors.As: an untrusted wrapper could add private text around one.
	if safe, ok := err.(*budgetBatchError); ok {
		return safe
	}
	class := "budget batch: provider failure"
	switch {
	case errors.Is(err, context.Canceled):
		class = "budget batch: canceled"
	case errors.Is(err, context.DeadlineExceeded):
		class = "budget batch: deadline exceeded"
	}
	return newBudgetBatchError(class, err)
}

// ExtractTransactionsFromSMSChunk runs batch transaction extraction on primary with fallback.
func (s *AIServiceWithFallback) ExtractTransactionsFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSTransactionResponse, error) {
	resp, err := s.primary.ExtractTransactionsFromSMSChunk(ctx, numberedSMSBlock)
	if err == nil {
		s.usedService = s.getPrimaryServiceName()
		return resp, nil
	}
	s.logger.Warn("Batch transacciones: primario falló, intentando fallback", zap.Error(err))
	if s.fallback != nil {
		resp, fallbackErr := s.fallback.ExtractTransactionsFromSMSChunk(ctx, numberedSMSBlock)
		if fallbackErr == nil {
			s.usedService = s.getFallbackServiceName()
			return resp, nil
		}
		return nil, fmt.Errorf("batch transacciones IA: primario=%v, fallback=%v", err, fallbackErr)
	}
	return nil, err
}

// GetUsedService returns the name of the service that was last used successfully.
// HasProvider reports whether an AI provider is configured.
func (s *AIServiceWithFallback) HasProvider() bool {
	_, unavailable := s.primary.(unavailableAIService)
	return s.primary != nil && !unavailable
}

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

type unavailableAIService struct{}

func (unavailableAIService) ExtractTransactionFromSMS(context.Context, string) (*TransactionExtraction, error) {
	return nil, apperrors.ErrAIServiceUnavailable
}

func (unavailableAIService) ExtractBudgetLinesFromSMSChunk(context.Context, string) (*BatchSMSBudgetResponse, error) {
	return nil, apperrors.ErrAIServiceUnavailable
}

func (unavailableAIService) ExtractTransactionsFromSMSChunk(context.Context, string) (*BatchSMSTransactionResponse, error) {
	return nil, apperrors.ErrAIServiceUnavailable
}
