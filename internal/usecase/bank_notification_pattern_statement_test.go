package usecase

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

type fakeOCRExtractor struct {
	text string
	err  error
}

func (f *fakeOCRExtractor) ExtractText(ctx context.Context, filename string, content []byte) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

func TestAnalyzeStatementDocument_RequiresOCRProvider(t *testing.T) {
	uc := &BankNotificationPatternUseCase{aiService: newAvailableFakeAIService(t)}

	_, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.pdf", []byte("pdf-bytes"))
	assertStatementOCRError(t, err, "OCR provider not configured in server")
}

func TestAnalyzeStatementDocument_PropagatesOCRFailure(t *testing.T) {
	uc := &BankNotificationPatternUseCase{
		aiService:  newAvailableFakeAIService(t),
		ocrService: &fakeOCRExtractor{err: errors.New("provider unavailable")},
	}

	_, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.pdf", []byte("pdf-bytes"))
	assertStatementOCRError(t, err, "failed extracting text with OCR provider")
}

func newAvailableFakeAIService(t *testing.T) *webapi.AIServiceWithFallback {
	t.Helper()
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("GEMINI_API_KEY", "")

	service, err := webapi.NewAIServiceWithFallback()
	if err != nil {
		t.Fatalf("NewAIServiceWithFallback() error = %v", err)
	}
	if !service.HasProvider() {
		t.Fatal("NewAIServiceWithFallback() has no available provider")
	}
	return service
}

func assertStatementOCRError(t *testing.T, err error, wantDetails string) {
	t.Helper()

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("AnalyzeStatementDocument() error = %v; want AppError", err)
	}
	if appErr.Code != apperrors.ErrCodeInternal {
		t.Fatalf("AnalyzeStatementDocument() code = %q; want %q", appErr.Code, apperrors.ErrCodeInternal)
	}
	if appErr.Details != wantDetails {
		t.Fatalf("AnalyzeStatementDocument() details = %q; want %q", appErr.Details, wantDetails)
	}
}

func TestAnalyzeStatementDocument_RequiresAIProviderBeforeOCR(t *testing.T) {
	uc := &BankNotificationPatternUseCase{
		ocrService: &fakeOCRExtractor{text: "extracto sin montos detectables"},
	}

	_, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.jpg", []byte("image-bytes"))
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("AnalyzeStatementDocument() error = %v; want AppError", err)
	}
	if appErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("AnalyzeStatementDocument() status = %d; want %d", appErr.StatusCode, http.StatusServiceUnavailable)
	}
}
