package webapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/genai"
)

const budgetPrivateSentinel = "SYNTHETIC_PRIVATE_SMS_AMOUNT_SENDER_TOKEN_CREDENTIAL"
const budgetPrivateURL = "https://synthetic.invalid/private-request"

func budgetPrivacyLogger() (*zap.Logger, *bytes.Buffer) {
	logs := new(bytes.Buffer)
	return zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(logs), zap.DebugLevel)), logs
}

func assertBudgetPrivacy(t *testing.T, err error, logs *bytes.Buffer) {
	t.Helper()
	for _, private := range []string{budgetPrivateSentinel, budgetPrivateURL, "generativelanguage.googleapis.com", "openrouter.ai/api/"} {
		if err != nil && strings.Contains(err.Error(), private) {
			t.Errorf("returned error contains private content or request URL: %q", private)
		}
		if strings.Contains(logs.String(), private) {
			t.Errorf("logs contain private content or request URL: %q", private)
		}
	}
}

type budgetFailingBody struct{ err error }

func (b budgetFailingBody) Read([]byte) (int, error) { return 0, b.err }
func (b budgetFailingBody) Close() error             { return nil }

func TestBudgetPrivacyOpenRouterIO(t *testing.T) {
	for _, kind := range []string{"transport", "body_read"} {
		for _, cause := range []error{nil, context.Canceled, context.DeadlineExceeded} {
			name := kind
			if cause != nil {
				name += "/" + cause.Error()
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					logger, logs := budgetPrivacyLogger()
					rawErr := errors.New(budgetPrivateSentinel + " " + budgetPrivateURL)
					if cause != nil {
						rawErr = fmt.Errorf("%s %s: %w", budgetPrivateSentinel, budgetPrivateURL, cause)
					}
					calls := 0
					provider := &OpenRouterService{logger: logger, batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
						calls++
						if kind == "transport" {
							return nil, rawErr
						}
						return &http.Response{StatusCode: 200, Body: budgetFailingBody{rawErr}}, nil
					})}}
					result, err := provider.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+budgetPrivateSentinel)
					assertBudgetPrivacy(t, err, logs)
					wantClass := "transport failure"
					if kind == "body_read" {
						wantClass = "response body read failure"
					}
					if err == nil || !strings.Contains(err.Error(), wantClass) {
						t.Errorf("missing stable class %q", wantClass)
					}
					if errors.Is(err, rawErr) {
						t.Error("raw cause retained in error chain")
					}
					if result != nil || calls != 4 {
						t.Errorf("result = %#v, calls = %d; want nil and four retries", result, calls)
					}
					if cause != nil && !errors.Is(err, cause) {
						t.Error("context identity lost")
					}
				})
			})
		}
	}
}

func budgetTestGemini(t *testing.T, logger *zap.Logger, transport batchPrivacyTransport) *GeminiService {
	t.Helper()
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI, APIKey: "synthetic-test-key",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &GeminiService{client: client, logger: logger}
}

func budgetGeminiResponse(t *testing.T, content string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]string{"text": content}}, "role": "model"}, "finishReason": "STOP"}}})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}
}

func TestBudgetPrivacyGemini(t *testing.T) {
	for _, kind := range []string{"sdk_error", "malformed_model", "model_type_error", "canceled", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			logger, logs := budgetPrivacyLogger()
			var cause error
			provider := budgetTestGemini(t, logger, func(*http.Request) (*http.Response, error) {
				switch kind {
				case "sdk_error":
					return &http.Response{StatusCode: 403, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":403,"message":"` + budgetPrivateSentinel + `","status":"PERMISSION_DENIED"}}`))}, nil
				case "canceled", "deadline":
					cause = context.Canceled
					if kind == "deadline" {
						cause = context.DeadlineExceeded
					}
					return nil, fmt.Errorf("%s %s: %w", budgetPrivateSentinel, budgetPrivateURL, cause)
				case "model_type_error":
					return budgetGeminiResponse(t, `{"lines":[{"amount":"`+budgetPrivateSentinel+`"}]}`), nil
				default:
					return budgetGeminiResponse(t, budgetPrivateSentinel), nil
				}
			})
			result, err := provider.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+budgetPrivateSentinel)
			assertBudgetPrivacy(t, err, logs)
			wantClass := "SDK failure"
			if kind == "malformed_model" || kind == "model_type_error" {
				wantClass = "invalid batch JSON"
			}
			if err == nil || !strings.Contains(err.Error(), wantClass) {
				t.Errorf("missing stable class %q", wantClass)
			}
			if result != nil {
				t.Error("expected nil result")
			}
			if cause != nil && !errors.Is(err, cause) {
				t.Error("context identity lost")
			}
		})
	}
}

// Embedding provides the unrelated interface methods without exercising them.
type budgetContractViolator struct {
	AIService
	err   error
	calls int
}

func (p *budgetContractViolator) ExtractBudgetLinesFromSMSChunk(context.Context, string) (*BatchSMSBudgetResponse, error) {
	p.calls++
	return nil, p.err
}

func TestBudgetPrivacyFallbackWrapper(t *testing.T) {
	for _, kind := range []string{"no_fallback", "raw_fallback", "wrapped_safe_error", "gemini_sdk_error", "gemini_malformed", "fallback_success", "canceled", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				logger, logs := budgetPrivacyLogger()
				rawErr := errors.New(budgetPrivateSentinel + " " + budgetPrivateURL)
				primaryErr := rawErr
				var cause error
				if kind == "wrapped_safe_error" {
					primaryErr = fmt.Errorf("%s: %w", budgetPrivateSentinel, newBudgetBatchError("safe local class", nil))
				}
				if kind == "canceled" || kind == "deadline" {
					cause = context.Canceled
					if kind == "deadline" {
						cause = context.DeadlineExceeded
					}
					primaryErr = fmt.Errorf("%s: %w", budgetPrivateSentinel, cause)
				}
				primaryCalls := 0
				primary := &OpenRouterService{logger: logger, batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
					primaryCalls++
					return nil, primaryErr
				})}}
				service := &AIServiceWithFallback{primary: primary, logger: logger}
				fallbackCalls := 0
				switch kind {
				case "raw_fallback", "wrapped_safe_error", "canceled", "deadline":
					service.fallback = &budgetContractViolator{err: rawErr}
				case "gemini_sdk_error", "gemini_malformed", "fallback_success":
					service.fallback = budgetTestGemini(t, logger, func(*http.Request) (*http.Response, error) {
						fallbackCalls++
						if kind == "gemini_sdk_error" {
							return nil, rawErr
						}
						if kind == "gemini_malformed" {
							return budgetGeminiResponse(t, budgetPrivateSentinel), nil
						}
						return budgetGeminiResponse(t, `{"lines":[{"line":1,"amount":10,"transaction_type":"expense"}]}`), nil
					})
				}
				result, err := service.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+budgetPrivateSentinel)
				if err == nil || result != nil {
					t.Errorf("result = %#v, error = %v; want fail-closed primary failure", result, err)
				}
				assertBudgetPrivacy(t, err, logs)
				if !strings.Contains(logs.String(), "primario") {
					t.Error("primary warning not captured")
				}
				if primaryCalls != 4 {
					t.Errorf("primary calls = %d; want 4", primaryCalls)
				}
				if fallback, ok := service.fallback.(*budgetContractViolator); ok && fallback.calls != 0 {
					t.Error("non-ZDR fallback was invoked")
				}
				if fallbackCalls != 0 {
					t.Errorf("Gemini calls = %d; want 0", fallbackCalls)
				}
				if cause != nil && !errors.Is(err, cause) {
					t.Error("context identity lost in wrapper")
				}
			})
		})
	}
}

func TestBudgetPrivacyRejectNonOpenRouterPrimary(t *testing.T) {
	for _, kind := range []string{"gemini", "unknown", "unavailable", "nil", "typed_nil"} {
		t.Run(kind, func(t *testing.T) {
			logger, logs := budgetPrivacyLogger()
			geminiCalls := 0
			gemini := budgetTestGemini(t, logger, func(*http.Request) (*http.Response, error) {
				geminiCalls++
				return budgetGeminiResponse(t, `{"lines":[]}`), nil
			})
			unknown := &budgetContractViolator{err: errors.New(budgetPrivateSentinel)}
			service := &AIServiceWithFallback{fallback: gemini, logger: logger}
			switch kind {
			case "gemini":
				service.primary = gemini
			case "unknown":
				service.primary = unknown
			case "unavailable":
				service.primary = unavailableAIService{}
			case "typed_nil":
				service.primary = (*OpenRouterService)(nil)
			}
			result, err := service.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+budgetPrivateSentinel)
			if result != nil || !errors.Is(err, apperrors.ErrAIServiceUnavailable) {
				t.Errorf("result = %#v, error = %v; want unavailable strict budget route", result, err)
			}
			if geminiCalls != 0 || unknown.calls != 0 {
				t.Errorf("Gemini/unknown calls = %d/%d; want 0/0", geminiCalls, unknown.calls)
			}
			assertBudgetPrivacy(t, err, logs)
		})
	}
}
