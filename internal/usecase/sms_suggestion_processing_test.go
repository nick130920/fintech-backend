package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

type suggestionCategoryRepo struct {
	repo.CategoryRepo
	categories []*entity.Category
	err        error
}

func (r suggestionCategoryRepo) GetAllAvailableForUser(uint) ([]*entity.Category, error) {
	return r.categories, r.err
}

type suggestionRoundTripper func(*http.Request) (*http.Response, error)

func (f suggestionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Intercept every provider request; tests never contact a real provider or read runtime credentials.
func newSuggestionProcessingTestUseCase(t *testing.T, results ...*webapi.BatchSMSBudgetResponse) (*BankNotificationPatternUseCase, *bytes.Buffer, *int) {
	t.Helper()
	t.Setenv("OPENROUTER_API_KEY", "synthetic-test-key")
	t.Setenv("GEMINI_API_KEY", "")
	calls := new(int)
	originalTransport := http.DefaultTransport
	http.DefaultTransport = suggestionRoundTripper(func(*http.Request) (*http.Response, error) {
		index := *calls
		(*calls)++
		if index >= len(results) {
			t.Fatal("unexpected provider request")
		}
		status := http.StatusForbidden
		body := "synthetic provider failure"
		if results[index] != nil {
			status = http.StatusOK
			content, err := json.Marshal(results[index])
			if err != nil {
				t.Fatal(err)
			}
			envelope, err := json.Marshal(map[string]any{
				"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			body = string(envelope)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	ai, err := webapi.NewAIServiceWithFallback()
	if err != nil {
		t.Fatal(err)
	}
	logs := new(bytes.Buffer)
	originalWriter, originalFlags, originalPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})
	uc := &BankNotificationPatternUseCase{
		aiService: ai,
		categoryRepo: suggestionCategoryRepo{categories: []*entity.Category{
			{ID: 1, Name: "Alimentación"}, {ID: 2, Name: "Otros"},
		}},
	}
	return uc, logs, calls
}

func syntheticSuggestionMessages(n int) []dto.SMSMessageForAnalysis {
	messages := make([]dto.SMSMessageForAnalysis, n)
	for i := range messages {
		messages[i] = dto.SMSMessageForAnalysis{Body: "Banco: compra realizada 10"}
	}
	return messages
}

func assertSuggestionCounts(t *testing.T, logs *bytes.Buffer, want string) {
	t.Helper()
	// Exact equality also guards against adding bodies, sender data, amounts, or provider errors.
	if got := logs.String(); got != "sms_budget_suggestions: "+want+"\n" {
		t.Errorf("aggregate log = %q; want counts only: %q", got, want)
	}
}

func TestProcessSMSBatchForSuggestions_TotalAIFailure(t *testing.T) {
	uc, logs, calls := newSuggestionProcessingTestUseCase(t, nil, nil)
	response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(26))
	if err == nil || response != nil {
		t.Errorf("total AI failure: response = %#v, error = %v; want nil response and error", response, err)
	}
	if err != nil && err.Error() != "all attempted AI chunks failed for SMS budget suggestions" {
		t.Errorf("error = %q; want unchanged sanitized generic error", err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Error("non-context failure must not acquire context sentinel identity")
	}
	if *calls != 2 {
		t.Errorf("provider calls = %d; want 2", *calls)
	}
	assertSuggestionCounts(t, logs, "received_messages=26 filtered_messages=26 attempted_chunks=2 successful_chunks=0 failed_chunks=2 accepted_expense_lines=0")
}

// Each failing chunk exhausts the provider's four attempts. A nil error means
// that chunk succeeds with one accepted expense. Callers use synctest for backoff.
func newSuggestionChunkErrorTestUseCase(t *testing.T, chunkErrors ...error) (*BankNotificationPatternUseCase, *bytes.Buffer, *int) {
	t.Helper()
	results := make([]*webapi.BatchSMSBudgetResponse, len(chunkErrors))
	for i := range results {
		results[i] = &webapi.BatchSMSBudgetResponse{Lines: []webapi.BatchSMSBudgetLine{
			{Line: 1, Amount: 10, TransactionType: "expense", Confidence: 0.9, CategoryKey: "food"},
		}}
	}
	uc, logs, _ := newSuggestionProcessingTestUseCase(t, results...)
	successTransport := http.DefaultTransport
	calls := new(int)
	chunk, attempt := 0, 0
	http.DefaultTransport = suggestionRoundTripper(func(req *http.Request) (*http.Response, error) {
		(*calls)++
		if chunk >= len(chunkErrors) {
			t.Fatal("unexpected provider request")
		}
		if err := chunkErrors[chunk]; err != nil {
			attempt++
			if attempt == 4 {
				chunk++
				attempt = 0
			}
			return nil, err
		}
		chunk++
		return successTransport.RoundTrip(req)
	})
	return uc, logs, calls
}

func TestProcessSMSBatchForSuggestions_AllChunkContextFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		causes []error
		want   []error
	}{
		{name: "canceled", causes: []error{context.Canceled, context.Canceled}, want: []error{context.Canceled}},
		{name: "deadline", causes: []error{context.DeadlineExceeded, context.DeadlineExceeded}, want: []error{context.DeadlineExceeded}},
		{name: "both", causes: []error{context.Canceled, context.DeadlineExceeded}, want: []error{context.Canceled, context.DeadlineExceeded}},
		{name: "context_then_other", causes: []error{context.Canceled, errors.New("provider unavailable")}, want: []error{context.Canceled}},
		{name: "other_then_context", causes: []error{errors.New("provider unavailable"), context.DeadlineExceeded}, want: []error{context.DeadlineExceeded}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				providerErrors := make([]error, len(tc.causes))
				for i, cause := range tc.causes {
					providerErrors[i] = fmt.Errorf("synthetic private provider detail: %w", cause)
				}
				uc, logs, calls := newSuggestionChunkErrorTestUseCase(t, providerErrors...)
				response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(26))
				if response != nil || err == nil {
					t.Fatalf("all chunks failed: response = %#v, error = %v", response, err)
				}
				for _, want := range tc.want {
					if !errors.Is(err, want) {
						t.Errorf("error = %v; want errors.Is(_, %v)", err, want)
					}
				}
				for _, raw := range providerErrors {
					if errors.Is(err, raw) {
						t.Error("error chain must not retain raw provider errors")
					}
				}
				if strings.Contains(err.Error(), "synthetic private provider detail") {
					t.Error("error must not expose raw provider content")
				}
				if *calls != 8 {
					t.Errorf("provider calls = %d; want 8 (four attempts per chunk)", *calls)
				}
				assertSuggestionCounts(t, logs, "received_messages=26 filtered_messages=26 attempted_chunks=2 successful_chunks=0 failed_chunks=2 accepted_expense_lines=0")
			})
		})
	}
}

func TestProcessSMSBatchForSuggestions_PartialSuccessWithContextFailure(t *testing.T) {
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, successFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/success_first_%t", sentinel, successFirst), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					providerErr := fmt.Errorf("synthetic private provider detail: %w", sentinel)
					chunkErrors := []error{providerErr, nil}
					if successFirst {
						chunkErrors = []error{nil, providerErr}
					}
					uc, logs, calls := newSuggestionChunkErrorTestUseCase(t, chunkErrors...)
					response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(26))
					if err != nil || response == nil {
						t.Fatalf("partial success: response = %#v, error = %v", response, err)
					}
					if *calls != 5 || response.Suggestions.TotalExpense3m != 10 || len(response.Suggestions.ByCategory) != 1 || response.Suggestions.ByCategory[0].Count != 1 {
						t.Errorf("partial success: calls = %d, suggestions = %#v", *calls, response.Suggestions)
					}
					assertSuggestionCounts(t, logs, "received_messages=26 filtered_messages=26 attempted_chunks=2 successful_chunks=1 failed_chunks=1 accepted_expense_lines=1")
				})
			})
		}
	}
}

func TestProcessSMSBatchForSuggestions_PartialSuccess(t *testing.T) {
	accepted := &webapi.BatchSMSBudgetResponse{Lines: []webapi.BatchSMSBudgetLine{
		{Line: 1, Amount: 10, TransactionType: "expense", Confidence: 0.9, CategoryKey: "food"},
		{Line: 1, Amount: 10, TransactionType: "expense", Confidence: 0.9, CategoryKey: "food"}, // duplicate
		{Line: 2, Amount: 20, TransactionType: "income", Confidence: 0.9},
		{Line: 3, Amount: 30, TransactionType: "expense", Confidence: 0.1},
		{Line: 4, Amount: 0, TransactionType: "expense", Confidence: 0.9},
	}}
	for _, successFirst := range []bool{true, false} {
		name := "failure_then_success"
		results := []*webapi.BatchSMSBudgetResponse{nil, accepted}
		if successFirst {
			name = "success_then_failure"
			results = []*webapi.BatchSMSBudgetResponse{accepted, nil}
		}
		t.Run(name, func(t *testing.T) {
			uc, logs, calls := newSuggestionProcessingTestUseCase(t, results...)
			response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(26))
			if err != nil || response == nil {
				t.Fatalf("partial success: response = %#v, error = %v; want successful response", response, err)
			}
			if *calls != 2 || response.Suggestions.TotalExpense3m != 10 || len(response.Suggestions.ByCategory) != 1 {
				t.Fatalf("partial success: calls = %d, suggestions = %#v", *calls, response.Suggestions)
			}
			category := response.Suggestions.ByCategory[0]
			if category.CategoryID != 1 || category.Total != 10 || category.Count != 1 {
				t.Errorf("accepted category = %#v; want category 1 with one expense totaling 10", category)
			}
			assertSuggestionCounts(t, logs, "received_messages=26 filtered_messages=26 attempted_chunks=2 successful_chunks=1 failed_chunks=1 accepted_expense_lines=1")
		})
	}
}

func TestProcessSMSBatchForSuggestions_ProcessedWithoutAcceptedExpenses(t *testing.T) {
	for _, lines := range [][]webapi.BatchSMSBudgetLine{
		nil,
		{{Line: 1, Amount: 10, TransactionType: "income", Confidence: 0.9}},
	} {
		uc, logs, calls := newSuggestionProcessingTestUseCase(t, nil, &webapi.BatchSMSBudgetResponse{Lines: lines})
		response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(26))
		if err != nil || response == nil {
			t.Fatalf("processed empty result: response = %#v, error = %v", response, err)
		}
		if *calls != 2 || response.Suggestions.TotalExpense3m != 0 || response.Suggestions.ByCategory == nil || len(response.Suggestions.ByCategory) != 0 {
			t.Errorf("processed empty result: calls = %d, suggestions = %#v", *calls, response.Suggestions)
		}
		assertSuggestionCounts(t, logs, "received_messages=26 filtered_messages=26 attempted_chunks=2 successful_chunks=1 failed_chunks=1 accepted_expense_lines=0")
	}
}

func TestProcessSMSBatchForSuggestions_CategoryFailures(t *testing.T) {
	repositoryErr := errors.New("synthetic category repository error")
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "repository_error", err: repositoryErr},
		{name: "no_categories"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc, logs, calls := newSuggestionProcessingTestUseCase(t)
			uc.categoryRepo = suggestionCategoryRepo{err: tc.err}
			response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(1))
			if err == nil || response != nil {
				t.Errorf("categories unavailable: response = %#v, error = %v; want nil response and error", response, err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Errorf("error = %v; want wrapped repository error", err)
			}
			if tc.err == nil && (err == nil || !strings.Contains(err.Error(), "categories")) {
				t.Errorf("error = %v; want clear missing categories error", err)
			}
			if *calls != 0 {
				t.Errorf("provider calls = %d; want 0", *calls)
			}
			assertSuggestionCounts(t, logs, "received_messages=1 filtered_messages=1 attempted_chunks=0 successful_chunks=0 failed_chunks=0 accepted_expense_lines=0")
		})
	}
}

// Inspect numbered input as well as results: aggregate totals alone cannot prove
// that every submitted message reaches exactly one chunk.
func newCompleteSuggestionBatchTestUseCase(t *testing.T, n int) (*BankNotificationPatternUseCase, *bytes.Buffer, *int, []dto.SMSMessageForAnalysis, map[string]int) {
	t.Helper()
	messages := syntheticSuggestionMessages(n)
	for i := range messages {
		messages[i].Body += fmt.Sprintf(" synthetic-message-%03d", i)
	}
	results := make([]*webapi.BatchSMSBudgetResponse, (n+24)/25)
	for chunk := range results {
		results[chunk] = &webapi.BatchSMSBudgetResponse{}
		for line := 1; line <= 25 && chunk*25+line <= n; line++ {
			results[chunk].Lines = append(results[chunk].Lines, webapi.BatchSMSBudgetLine{
				Line: line, Amount: 10, TransactionType: "expense", Confidence: 0.9, CategoryKey: "food",
			})
		}
	}
	uc, logs, calls := newSuggestionProcessingTestUseCase(t, results...)
	transport := http.DefaultTransport
	seen := make(map[string]int)
	http.DefaultTransport = suggestionRoundTripper(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal("cannot read synthetic request")
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		var payload struct {
			Model    string `json:"model"`
			Provider struct {
				ZDR bool `json:"zdr"`
			} `json:"provider"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if json.Unmarshal(body, &payload) != nil || len(payload.Messages) != 1 {
			t.Fatal("invalid synthetic request envelope")
		}
		if payload.Model != "deepseek/deepseek-v4.1-flash" || !payload.Provider.ZDR {
			t.Error("budget request lost strict model/ZDR routing")
		}
		_, block, ok := strings.Cut(payload.Messages[0].Content, "BLOQUE:\n")
		if !ok {
			t.Fatal("numbered SMS block missing")
		}
		block, _, _ = strings.Cut(block, "\nPara cada línea")
		lines := strings.Split(strings.TrimSpace(block), "\n")
		wantSize := 25
		if remaining := n - *calls*25; remaining < wantSize {
			wantSize = remaining
		}
		if len(lines) != wantSize {
			t.Errorf("chunk size = %d; want %d", len(lines), wantSize)
		}
		for i, line := range lines {
			prefix := fmt.Sprintf("[%d] ", i+1)
			if !strings.HasPrefix(line, prefix) {
				t.Error("invalid chunk-local line numbering")
			}
			seen[strings.TrimPrefix(line, prefix)]++
		}
		return transport.RoundTrip(req)
	})
	return uc, logs, calls, messages, seen
}

func TestProcessSMSBatchForSuggestions_ProcessesEveryMessageExactlyOnce(t *testing.T) {
	for _, n := range []int{101, 126, 500} {
		t.Run(fmt.Sprintf("messages_%d", n), func(t *testing.T) {
			uc, logs, calls, messages, seen := newCompleteSuggestionBatchTestUseCase(t, n)
			response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, messages)
			if err != nil || response == nil {
				t.Fatalf("complete batch error = %v", err)
			}
			chunks := (n + 24) / 25
			if *calls != chunks {
				t.Errorf("provider calls = %d; want %d", *calls, chunks)
			}
			for i, message := range messages {
				if seen[message.Body] != 1 {
					t.Errorf("message index %d processed %d times; want once", i, seen[message.Body])
				}
			}
			if len(seen) != n || response.Suggestions.TotalExpense3m != float64(n*10) || len(response.Suggestions.ByCategory) != 1 || response.Suggestions.ByCategory[0].Count != n {
				t.Error("complete batch must aggregate every expense exactly once")
			}
			assertSuggestionCounts(t, logs, fmt.Sprintf("received_messages=%d filtered_messages=%d attempted_chunks=%d successful_chunks=%d failed_chunks=0 accepted_expense_lines=%d", n, n, chunks, chunks, n))
		})
	}
}

func TestProcessSMSBatchForSuggestions_RejectsOverLimitBeforeProcessing(t *testing.T) {
	uc, logs, calls := newSuggestionProcessingTestUseCase(t)
	uc.categoryRepo = suggestionCategoryRepo{err: errors.New("category lookup must not run")}
	response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, syntheticSuggestionMessages(501))
	var appErr *apperrors.AppError
	if response != nil || !errors.As(err, &appErr) || appErr.Code != apperrors.ErrCodeInvalidRequest || appErr.StatusCode != http.StatusBadRequest || appErr.Internal != nil {
		t.Errorf("over-limit batch must return a safe invalid-request 400; got %T", err)
	}
	if *calls != 0 {
		t.Errorf("provider calls = %d; want zero", *calls)
	}
	assertSuggestionCounts(t, logs, "received_messages=501 filtered_messages=0 attempted_chunks=0 successful_chunks=0 failed_chunks=0 accepted_expense_lines=0")
}

func TestProcessSMSBatchForSuggestions_NoFilteredMessages(t *testing.T) {
	uc, logs, calls := newSuggestionProcessingTestUseCase(t)
	// No AI processing requires no categories, either.
	uc.categoryRepo = nil
	response, err := uc.ProcessSMSBatchForSuggestions(context.Background(), 1, []dto.SMSMessageForAnalysis{{Body: "Synthetic unrelated message"}, {Body: "   "}})
	if err != nil || response == nil {
		t.Fatalf("no filtered messages: response = %#v, error = %v", response, err)
	}
	if *calls != 0 || response.Suggestions.TotalExpense3m != 0 || response.Suggestions.ByCategory == nil || len(response.Suggestions.ByCategory) != 0 {
		t.Errorf("no filtered messages: calls = %d, suggestions = %#v", *calls, response.Suggestions)
	}
	assertSuggestionCounts(t, logs, "received_messages=2 filtered_messages=0 attempted_chunks=0 successful_chunks=0 failed_chunks=0 accepted_expense_lines=0")
}
