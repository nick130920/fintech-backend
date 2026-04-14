package usecase

import (
	"context"
	"errors"
	"testing"
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
	uc := &BankNotificationPatternUseCase{}

	_, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.pdf", []byte("pdf-bytes"))
	if err == nil {
		t.Fatalf("expected error when OCR provider is missing")
	}
}

func TestAnalyzeStatementDocument_PropagatesOCRFailure(t *testing.T) {
	uc := &BankNotificationPatternUseCase{
		ocrService: &fakeOCRExtractor{err: errors.New("provider unavailable")},
	}

	_, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.pdf", []byte("pdf-bytes"))
	if err == nil {
		t.Fatalf("expected error when OCR extraction fails")
	}
}

func TestAnalyzeStatementDocument_EmptySuggestionFlowWithoutDigits(t *testing.T) {
	uc := &BankNotificationPatternUseCase{
		ocrService: &fakeOCRExtractor{text: "extracto sin montos detectables"},
	}

	resp, err := uc.AnalyzeStatementDocument(context.Background(), 10, "statement.jpg", []byte("image-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Suggestions.ByCategory) != 0 || resp.Suggestions.TotalExpense3m != 0 {
		t.Fatalf("expected empty suggestions response, got %#v", resp)
	}
}
