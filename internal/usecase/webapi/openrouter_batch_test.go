package webapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type batchPrivacyTransport func(*http.Request) (*http.Response, error)

func (f batchPrivacyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func batchModelEnvelope(t *testing.T, content string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"choices": []any{map[string]any{"message": map[string]string{"content": content}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func batchProviderErrorEnvelope(t *testing.T, message string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"error": map[string]any{"code": 503, "message": message}})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestOpenRouterBatchBudgetPrivacy(t *testing.T) {
	const secret = "SYNTHETIC_PRIVATE_SMS_PROVIDER_CONTENT"
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantClass string
		wantCalls int
	}{
		{"forbidden_body", 403, secret, "status 403", 1},
		{"server_error_body", 503, secret, "status 503", 1},
		{"rate_limit_exhaustion", 429, secret, "status 429", 4},
		{"provider_error", 200, batchProviderErrorEnvelope(t, secret), "provider error", 1},
		{"provider_rate_limit_exhaustion", 200, batchProviderErrorEnvelope(t, "rate limit: "+secret), "rate limit", 4},
		{"provider_429_exhaustion", 200, batchProviderErrorEnvelope(t, "429: "+secret), "rate limit", 4},
		{"malformed_envelope", 200, secret, "parse provider JSON", 1},
		{"malformed_model_json", 200, batchModelEnvelope(t, secret), "parse batch JSON", 1},
		{"model_type_error", 200, batchModelEnvelope(t, `{"lines":[{"amount":"`+secret+`"}]}`), "parse batch JSON", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Virtual time exercises every production retry without real backoff delays.
			synctest.Test(t, func(t *testing.T) {
				var logs bytes.Buffer
				logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&logs), zap.DebugLevel))
				calls := 0
				provider := &OpenRouterService{
					apiKey: "synthetic-test-key",
					logger: logger,
					batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
						calls++
						return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
					})},
				}
				// Include the actual fallback wrapper: it logs returned provider errors.
				service := &AIServiceWithFallback{primary: provider, logger: logger}
				result, err := service.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] "+secret)
				if err == nil || result != nil {
					t.Fatalf("result = %#v, error = %v; want nil result and sanitized error", result, err)
				}
				if !strings.Contains(err.Error(), tc.wantClass) {
					t.Errorf("error = %q; want classification %q", err, tc.wantClass)
				}
				if strings.Contains(err.Error(), secret) {
					t.Error("returned error contains private provider/model content")
				}
				if strings.Contains(logs.String(), secret) {
					t.Error("provider/fallback logs contain private provider/model content")
				}
				if !strings.Contains(logs.String(), "primario") {
					t.Error("test did not capture fallback wrapper warning")
				}
				if calls != tc.wantCalls {
					t.Errorf("provider calls = %d; want %d", calls, tc.wantCalls)
				}
			})
		})
	}
}

func TestOpenRouterBatchBudgetRetrySuccess(t *testing.T) {
	for _, firstStatus := range []int{http.StatusTooManyRequests, http.StatusOK} {
		name := "http_rate_limit"
		if firstStatus == http.StatusOK {
			name = "provider_rate_limit"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				provider := &OpenRouterService{
					logger: zap.NewNop(),
					batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
						calls++
						if calls == 1 {
							return &http.Response{StatusCode: firstStatus, Body: io.NopCloser(strings.NewReader(batchProviderErrorEnvelope(t, "rate limit")))}, nil
						}
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(batchModelEnvelope(t, "```json\n{\"lines\":[{\"line\":1,\"amount\":10,\"transaction_type\":\"expense\",\"confidence\":0.9,\"category_key\":\"food\"}]}\n```")))}, nil
					})},
				}
				result, err := provider.ExtractBudgetLinesFromSMSChunk(context.Background(), "[1] synthetic SMS")
				if err != nil || result == nil || len(result.Lines) != 1 {
					t.Fatalf("result = %#v, error = %v; want successful retry", result, err)
				}
				line := result.Lines[0]
				if calls != 2 || line.Line != 1 || line.Amount != 10 || line.TransactionType != "expense" || line.Confidence != 0.9 || line.CategoryKey != "food" {
					t.Errorf("calls = %d, line = %#v; retry changed successful parsing", calls, line)
				}
			})
		})
	}
}

func TestOpenRouterBatchBudgetStrictZDR(t *testing.T) {
	for _, kind := range []string{"http_rate_limit", "provider_rate_limit", "transport", "body_read", "no_eligible_route", "no_eligible_route_envelope", "empty_success"} {
		t.Run(kind, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := context.WithValue(context.Background(), struct{}{}, "synthetic-context")
				const block = "[1] synthetic purchase on 2026-03-01"
				calls := 0
				start := time.Now()
				provider := &OpenRouterService{logger: zap.NewNop()}
				provider.batchClient = &http.Client{Transport: batchPrivacyTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Context() != ctx {
						t.Error("request context identity changed")
					}
					wantDelay := []time.Duration{0, 3 * time.Second, 11 * time.Second, 26 * time.Second}[calls-1]
					if elapsed := time.Since(start); elapsed != wantDelay {
						t.Errorf("attempt %d elapsed = %v; want %v", calls, elapsed, wantDelay)
					}
					var body struct {
						Model          string          `json:"model"`
						Provider       map[string]any  `json:"provider"`
						Messages       []OpenRouterMsg `json:"messages"`
						ResponseFormat ResponseFormat  `json:"response_format"`
					}
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body.Model != "deepseek/deepseek-v4.1-flash" || body.Provider["zdr"] != true || len(body.Provider) != 1 {
						t.Errorf("attempt %d routing = model %q, provider %#v; want exact budget model and provider.zdr=true", calls, body.Model, body.Provider)
					}
					if len(body.Messages) != 1 || body.Messages[0].Role != "user" || body.Messages[0].Content != provider.buildBatchBudgetPrompt(block) || body.ResponseFormat.Type != "json_object" {
						t.Error("budget prompt/body semantics changed")
					}
					status, response := http.StatusOK, batchModelEnvelope(t, `{"lines":[]}`)
					switch kind {
					case "http_rate_limit":
						status, response = http.StatusTooManyRequests, "synthetic rate limit"
					case "provider_rate_limit":
						response = batchProviderErrorEnvelope(t, "rate limit")
					case "transport":
						return nil, errors.New("synthetic transport failure")
					case "body_read":
						return &http.Response{StatusCode: http.StatusOK, Body: budgetFailingBody{errors.New("synthetic read failure")}}, nil
					case "no_eligible_route":
						status, response = http.StatusNotFound, "No eligible ZDR route"
					case "no_eligible_route_envelope":
						response = `{"error":{"code":404,"message":"No eligible ZDR route"}}`
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(response))}, nil
				})}
				result, err := provider.ExtractBudgetLinesFromSMSChunk(ctx, block)
				wantCalls := 4
				if strings.HasPrefix(kind, "no_eligible_route") || kind == "empty_success" {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Errorf("calls = %d; want %d", calls, wantCalls)
				}
				if kind == "empty_success" {
					if err != nil || result == nil || len(result.Lines) != 0 {
						t.Errorf("result = %#v, error = %v; want legitimate zero-expense success", result, err)
					}
				} else if err == nil || result != nil {
					t.Errorf("result = %#v, error = %v; want fail-closed error", result, err)
				}
			})
		})
	}
}

func TestOpenRouterBatchBudgetRetryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	provider := &OpenRouterService{
		logger: zap.NewNop(),
		batchClient: &http.Client{Transport: batchPrivacyTransport(func(*http.Request) (*http.Response, error) {
			calls++
			cancel()
			return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("synthetic private body"))}, nil
		})},
	}
	result, err := provider.ExtractBudgetLinesFromSMSChunk(ctx, "[1] synthetic SMS")
	if result != nil || !errors.Is(err, context.Canceled) || calls != 1 {
		t.Errorf("result = %#v, error = %v, calls = %d; want cancellation before retry", result, err, calls)
	}
}
