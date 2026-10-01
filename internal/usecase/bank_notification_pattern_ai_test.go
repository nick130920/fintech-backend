package usecase

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

func TestStartSMSBatchSuggestionJob_RejectsUnavailableAIWithoutCreatingJob(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	aiService, err := webapi.NewAIServiceWithFallback()
	if err != nil {
		t.Fatalf("NewAIServiceWithFallback() error = %v", err)
	}
	uc := NewBankNotificationPatternUseCase(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, aiService, nil)

	_, err = uc.StartSMSBatchSuggestionJob(1, []dto.SMSMessageForAnalysis{{Body: "purchase"}})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("StartSMSBatchSuggestionJob() error = %v; want AppError", err)
	}
	if appErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("StartSMSBatchSuggestionJob() status = %d; want %d", appErr.StatusCode, http.StatusServiceUnavailable)
	}
}
