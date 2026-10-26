package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
)

func TestBudgetPrivacyOtherSMSRoutesUnchanged(t *testing.T) {
	for _, kind := range []string{"single_fallback", "transaction_batch_fallback", "single_gemini_primary", "transaction_batch_gemini_primary"} {
		t.Run(kind, func(t *testing.T) {
			logger, _ := budgetPrivacyLogger()
			primaryCalls, geminiCalls := 0, 0
			client := &http.Client{Transport: batchPrivacyTransport(func(req *http.Request) (*http.Response, error) {
				primaryCalls++
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["model"] != "openrouter/free" || body["provider"] != nil {
					t.Errorf("unrelated route changed: model = %v, provider = %v", body["model"], body["provider"])
				}
				return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("synthetic rejection"))}, nil
			})}
			primary := &OpenRouterService{logger: logger, httpClient: client, batchClient: client}
			gemini := budgetTestGemini(t, logger, func(*http.Request) (*http.Response, error) {
				geminiCalls++
				content := `{"success":true,"amount":10,"date":"2026-03-01","transaction_type":"expense"}`
				if strings.HasPrefix(kind, "transaction_batch") {
					content = `{"lines":[{"line":1,"success":true,"amount":10,"date":"2026-03-01","transaction_type":"expense"}]}`
				}
				return budgetGeminiResponse(t, content), nil
			})
			service := &AIServiceWithFallback{primary: primary, fallback: gemini, logger: logger}
			wantPrimary := 1
			if strings.HasSuffix(kind, "gemini_primary") {
				service.primary, service.fallback = gemini, nil
				wantPrimary = 0
			}
			if strings.HasPrefix(kind, "single") {
				result, err := service.ExtractTransactionFromSMS(context.Background(), "synthetic purchase on 2026-03-01")
				if err != nil || result == nil || result.Amount != 10 || result.Date != "2026-03-01" {
					t.Fatalf("single SMS result = %#v, error = %v", result, err)
				}
			} else {
				result, err := service.ExtractTransactionsFromSMSChunk(context.Background(), "[1] synthetic purchase on 2026-03-01")
				if err != nil || result == nil || len(result.Lines) != 1 || result.Lines[0].Date != "2026-03-01" {
					t.Fatalf("transaction batch result = %#v, error = %v", result, err)
				}
			}
			if primaryCalls != wantPrimary || geminiCalls != 1 {
				t.Errorf("OpenRouter/Gemini calls = %d/%d; want %d/1", primaryCalls, geminiCalls, wantPrimary)
			}
		})
	}
}

func TestBudgetPrivacyRealProviderPipeline(t *testing.T) {
	for _, kind := range []string{"transport_failure", "read_failure", "transport_retry_success", "read_retry_success", "no_eligible_route"} {
		t.Run(kind, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				logger, logs := budgetPrivacyLogger()
				rawErr := errors.New(budgetPrivateSentinel + " " + budgetPrivateURL)
				primaryCalls, fallbackCalls := 0, 0
				primary := &OpenRouterService{logger: logger, batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
					primaryCalls++
					if strings.Contains(kind, "retry_success") && primaryCalls == 2 {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(batchModelEnvelope(t, `{"lines":[]}`)))}, nil
					}
					if kind == "no_eligible_route" {
						return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("No eligible ZDR route"))}, nil
					}
					if strings.HasPrefix(kind, "transport") {
						return nil, rawErr
					}
					return &http.Response{StatusCode: 200, Body: budgetFailingBody{rawErr}}, nil
				})}}
				fallback := budgetTestGemini(t, logger, func(*http.Request) (*http.Response, error) {
					fallbackCalls++
					return budgetGeminiResponse(t, `{"lines":[]}`), nil
				})
				service := &AIServiceWithFallback{primary: primary, fallback: fallback, logger: logger}
				result, err := service.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+budgetPrivateSentinel)
				assertBudgetPrivacy(t, err, logs)
				wantPrimary, wantFallback := 4, 0
				if strings.Contains(kind, "retry_success") {
					wantPrimary = 2
				} else if kind == "no_eligible_route" {
					wantPrimary = 1
				}
				if primaryCalls != wantPrimary || fallbackCalls != wantFallback {
					t.Errorf("calls primary/fallback = %d/%d; want %d/%d", primaryCalls, fallbackCalls, wantPrimary, wantFallback)
				}
				if strings.Contains(kind, "success") {
					if err != nil || result == nil {
						t.Fatalf("expected success, got %v", err)
					}
				} else if err == nil || result != nil {
					t.Fatal("expected sanitized failure")
				}
			})
		})
	}
}
