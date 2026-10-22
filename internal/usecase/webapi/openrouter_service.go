package webapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1/chat/completions"
	// Modelo gratuito de Mistral
	defaultModel = "openrouter/free"
	// SMS budget extraction must use a ZDR-eligible route, never the free router.
	smsBudgetModel = "deepseek/deepseek-v4.1-flash"
	// Lotes grandes pueden tardar >2 min en generar JSON (tokens + lectura del body).
	openRouterBatchHTTPTimeout = 5 * time.Minute
)

// OpenRouterService handles interactions with the OpenRouter API.
type OpenRouterService struct {
	apiKey      string
	httpClient  *http.Client
	batchClient *http.Client // timeouts largos para lotes de SMS
	logger      *zap.Logger
}

// OpenRouterRequest represents the request body for OpenRouter API.
type OpenRouterRequest struct {
	Model          string              `json:"model"`
	Messages       []OpenRouterMsg     `json:"messages"`
	Temperature    float32             `json:"temperature,omitempty"`
	ResponseFormat *ResponseFormat     `json:"response_format,omitempty"`
	Provider       *OpenRouterProvider `json:"provider,omitempty"`
}

// OpenRouterProvider restricts routing for the strict SMS budget path.
type OpenRouterProvider struct {
	ZDR bool `json:"zdr"`
}

// OpenRouterMsg represents a message in the OpenRouter API.
type OpenRouterMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat for JSON mode
type ResponseFormat struct {
	Type string `json:"type"`
}

// OpenRouterResponse represents the response from OpenRouter API.
type OpenRouterResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// TransactionExtraction holds the structured data extracted from an SMS by AI.
type TransactionExtraction struct {
	Success         bool    `json:"success"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Merchant        string  `json:"merchant"`
	Date            string  `json:"date"`
	TransactionType string  `json:"transaction_type"` // "expense", "income", "transfer"
	Confidence      float64 `json:"confidence"`
	Currency        string  `json:"currency"`
	RawMessage      string  `json:"raw_message,omitempty"`
}

// NewOpenRouterService creates a new OpenRouterService.
func NewOpenRouterService() (*OpenRouterService, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno OPENROUTER_API_KEY no está configurada")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error al crear el logger: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	batchClient := &http.Client{
		Timeout: openRouterBatchHTTPTimeout,
	}

	return &OpenRouterService{
		apiKey:      apiKey,
		httpClient:  httpClient,
		batchClient: batchClient,
		logger:      logger,
	}, nil
}

// ExtractTransactionFromSMS analyzes an SMS and extracts transaction details using AI.
func (s *OpenRouterService) ExtractTransactionFromSMS(ctx context.Context, smsContent string) (*TransactionExtraction, error) {
	prompt := s.buildExtractionPrompt(smsContent)

	response, err := s.callOpenRouter(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error al llamar a OpenRouter: %w", err)
	}

	// Parse the JSON response
	var extraction TransactionExtraction
	if err := json.Unmarshal([]byte(response), &extraction); err != nil {
		s.logger.Error("Error parsing AI response",
			zap.String("response", response),
			zap.Error(err),
		)
		return nil, fmt.Errorf("error al parsear respuesta de IA: %w (respuesta: %s)", err, response)
	}

	extraction.RawMessage = smsContent
	s.logger.Info("Transaction extracted successfully",
		zap.Float64("amount", extraction.Amount),
		zap.String("type", extraction.TransactionType),
		zap.Float64("confidence", extraction.Confidence),
	)

	return &extraction, nil
}

// ExtractBudgetLinesFromSMSChunk analyzes many numbered SMS in a single IA request.
func (s *OpenRouterService) ExtractBudgetLinesFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSBudgetResponse, error) {
	prompt := s.buildBatchBudgetPrompt(numberedSMSBlock)
	response, err := s.callOpenRouterBatchWithPrivacy(ctx, prompt, true)
	if err != nil {
		return nil, err
	}
	var out BatchSMSBudgetResponse
	if err := json.Unmarshal([]byte(response), &out); err != nil {
		// Decoder errors can contain model-controlled field names or values.
		s.logger.Error("Error parsing batch budget response", zap.String("error_class", "invalid_batch_json"))
		return nil, newBudgetBatchError("parse batch JSON: invalid batch response", nil)
	}
	return &out, nil
}

// ExtractTransactionsFromSMSChunk analyzes many numbered SMS in a single IA request (crear transacciones).
func (s *OpenRouterService) ExtractTransactionsFromSMSChunk(ctx context.Context, numberedSMSBlock string) (*BatchSMSTransactionResponse, error) {
	prompt := s.buildBatchTransactionPrompt(numberedSMSBlock)
	response, err := s.callOpenRouterBatch(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter batch transacciones: %w", err)
	}
	var out BatchSMSTransactionResponse
	if err := json.Unmarshal([]byte(response), &out); err != nil {
		s.logger.Error("Error parsing batch transaction response", zap.String("response", response), zap.Error(err))
		return nil, fmt.Errorf("parse batch transacciones JSON: %w", err)
	}
	return &out, nil
}

func (s *OpenRouterService) buildBatchTransactionPrompt(block string) string {
	return fmt.Sprintf(`Eres un experto en SMS bancarios de Latinoamérica (Colombia, México, etc.).

Analiza el BLOQUE. Cada línea empieza con [N] y es UN SMS distinto.

BLOQUE:
%s

Para cada línea [N]:
- Si NO es una notificación de movimiento bancario real (publicidad, mora, recordatorio de pago, OTP, código de app): devuelve success=false, amount=0, confidence=0.
- Si SÍ es movimiento bancario con monto claro: success=true, confidence 0.35-1.

REGLAS (igual que extracción individual):
- transaction_type = "expense" si: pagaste, compraste, débito, transferiste saliente, enviaste
- transaction_type = "income" si: recibiste transferencia, abono, depósito
- transaction_type = "transfer" si es transferencia ambigua
- Monto numérico (1.000,50 → 1000.50)
- Moneda: COP para bancos colombianos, MXN para mexicanos
- date: YYYY-MM-DD si aparece en el SMS, si no aproxima o cadena vacía
- description: corta; merchant: comercio o persona si aplica

Responde SOLO JSON válido:
{"lines":[{"line":1,"success":true,"amount":100.5,"description":"...","merchant":"...","date":"2026-03-01","transaction_type":"expense","confidence":0.9,"currency":"COP"}]}`, block)
}

func (s *OpenRouterService) buildBatchBudgetPrompt(block string) string {
	return fmt.Sprintf(`Eres un experto en SMS bancarios de Latinoamérica (México, Colombia, etc.).

Analiza el BLOQUE. Cada línea empieza con [N].

BLOQUE:
%s

Para cada línea [N] con NOTIFICACIÓN BANCARIA clara y monto:
- transaction_type: "expense" (gasto/débito/compra/pago) | "income" (depósito recibido) | ignora la línea si no aplica.
- category_key SOLO si expense — clasifica el gasto según el comercio/contexto del SMS, usando EXACTAMENTE una de estas claves en inglés minúsculas:
  food → super, restaurante, OXXO, comida, uber eats, rappi
  transport → gasolina, uber, taxi, metro, peaje, estacionamiento
  entertainment → cine, netflix, spotify, juegos, bar, evento
  utilities → luz, agua, gas, internet, teléfono, recibo CFE
  health → farmacia, hospital, médico, seguro médico
  shopping → ropa, amazon, tienda departamental, electrónica (no comida)
  education → curso, colegiatura, libro escolar
  other → transferencias persona, retiros ATM sin comercio claro, comisiones, no clasificable

Reglas: una entrada por [N]; confidence 0.35-1; amount numérico.

JSON solo:
{"lines":[{"line":1,"amount":100.5,"transaction_type":"expense","confidence":0.9,"category_key":"food"}]}`, block)
}

func (s *OpenRouterService) callOpenRouterBatch(ctx context.Context, prompt string) (string, error) {
	return s.callOpenRouterBatchWithPrivacy(ctx, prompt, false)
}

func (s *OpenRouterService) callOpenRouterBatchWithPrivacy(ctx context.Context, prompt string, budgetOnly bool) (string, error) {
	// Keep transaction-creation batch behavior unchanged. Budget failures alone
	// discard untrusted causes, retaining only context sentinel identity.
	failure := func(class string, err error) error {
		if budgetOnly {
			return newBudgetBatchError(class, err)
		}
		return err
	}
	safeFailure := func(format string, args ...any) error {
		err := fmt.Errorf(format, args...)
		return failure(err.Error(), err)
	}
	reqBody := OpenRouterRequest{
		Model: defaultModel,
		Messages: []OpenRouterMsg{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}
	if budgetOnly {
		reqBody.Model = smsBudgetModel
		reqBody.Provider = &OpenRouterProvider{ZDR: true}
	}
	// Marshal once so every retry preserves the mandatory budget routing policy.
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", failure("budget batch: request encoding failure", err)
	}

	// Preserve the existing retry schedule; budget failures never downgrade routing.
	backoffs := []time.Duration{0, 3 * time.Second, 8 * time.Second, 15 * time.Second}
	var lastErr error

	for attempt, wait := range backoffs {
		if wait > 0 {
			select {
			case <-ctx.Done():
				return "", failure("budget batch: context ended", ctx.Err())
			case <-time.After(wait):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL, bytes.NewReader(jsonBody))
		if err != nil {
			return "", failure("budget batch: request creation failure", err)
		}
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HTTP-Referer", "https://money-flow-app.com")
		req.Header.Set("X-Title", "Money Flow App")

		resp, err := s.batchClient.Do(req)
		if err != nil {
			lastErr = failure("budget batch: transport failure", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = failure("budget batch: response body read failure", readErr)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			s.logger.Warn("OpenRouter batch 429 (rate limit), reintentando",
				zap.Int("intento", attempt+1),
				zap.Int("max_intentos", len(backoffs)),
			)
			lastErr = safeFailure("status 429: rate limit")
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return "", safeFailure("status %d: provider HTTP error", resp.StatusCode)
		}

		var openRouterResp OpenRouterResponse
		if err := json.Unmarshal(body, &openRouterResp); err != nil {
			return "", safeFailure("parse provider JSON: invalid response envelope")
		}
		if openRouterResp.Error != nil {
			// Algunos errores de proveedor vienen como 200 + error JSON
			msg := strings.ToLower(openRouterResp.Error.Message)
			if strings.Contains(msg, "rate") || strings.Contains(msg, "429") || strings.Contains(msg, "limit") {
				s.logger.Warn("OpenRouter batch error tipo rate limit en body, reintentando",
					zap.Int("intento", attempt+1),
					zap.String("error_class", "rate_limit"),
				)
				lastErr = safeFailure("provider error: rate limit (code %d)", openRouterResp.Error.Code)
				continue
			}
			return "", safeFailure("provider error (code %d)", openRouterResp.Error.Code)
		}
		if len(openRouterResp.Choices) == 0 {
			return "", safeFailure("sin choices")
		}
		content := s.cleanJSONResponse(openRouterResp.Choices[0].Message.Content)
		if budgetOnly {
			s.logger.Info("OpenRouter budget batch OK")
		} else {
			s.logger.Info("OpenRouter batch OK",
				zap.Int("prompt_tokens", openRouterResp.Usage.PromptTokens),
				zap.Int("lines_in_response_estimate", len(content)),
			)
		}
		return content, nil
	}

	if lastErr != nil {
		if budgetOnly {
			// lastErr is already sanitized, including its context sentinel.
			return "", newBudgetBatchError("OpenRouter batch tras reintentos: "+lastErr.Error(), lastErr)
		}
		return "", fmt.Errorf("OpenRouter batch tras reintentos: %w", lastErr)
	}
	return "", safeFailure("OpenRouter batch: sin respuesta tras reintentos")
}

// buildExtractionPrompt creates the prompt for transaction extraction.
func (s *OpenRouterService) buildExtractionPrompt(smsContent string) string {
	return fmt.Sprintf(`Eres un asistente experto en analizar notificaciones bancarias de SMS de bancos latinoamericanos (Colombia, México, etc.).

Analiza el siguiente SMS bancario y extrae la información de la transacción.

SMS: "%s"

REGLAS DE CLASIFICACIÓN:
- transaction_type = "expense" cuando: pagaste, compra, débito, retiro, transferiste, enviaste
- transaction_type = "income" cuando: recibiste, consignación, depósito, abono, ingreso
- transaction_type = "transfer" cuando: es una transferencia sin dirección clara
- success = true si el SMS contiene un monto y es claramente una notificación bancaria de movimiento
- success = false SOLO si el SMS no es una notificación de transacción bancaria

EXTRACCIÓN:
- Monto: solo el número (sin $, puntos ni comas; 1.000,00 → 1000.00)
- Moneda: COP para bancos colombianos (Bancolombia, Davivienda, Nequi, etc.), MXN para mexicanos
- Fecha: formato YYYY-MM-DD
- Merchant: comercio o persona que recibió/envió el dinero, o null
- Confidence: 0.9 o superior si ves monto y banco claramente

Responde ÚNICAMENTE con un JSON válido, sin texto adicional ni markdown:
{
  "success": true,
  "amount": 0.00,
  "description": "descripción corta de la transacción",
  "merchant": "nombre del comercio o persona, o null",
  "date": "YYYY-MM-DD",
  "transaction_type": "expense",
  "confidence": 0.95,
  "currency": "COP"
}`, smsContent)
}

// callOpenRouter makes a request to the OpenRouter API.
func (s *OpenRouterService) callOpenRouter(ctx context.Context, prompt string) (string, error) {
	reqBody := OpenRouterRequest{
		Model: defaultModel,
		Messages: []OpenRouterMsg{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.0,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error al serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error al crear request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://money-flow-app.com")
	req.Header.Set("X-Title", "Money Flow App")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al hacer request a OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("OpenRouter API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return "", fmt.Errorf("OpenRouter API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("error al parsear respuesta de OpenRouter: %w", err)
	}

	if openRouterResp.Error != nil {
		return "", fmt.Errorf("OpenRouter error: %s", openRouterResp.Error.Message)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("OpenRouter no devolvió ninguna respuesta")
	}

	content := openRouterResp.Choices[0].Message.Content
	content = s.cleanJSONResponse(content)

	s.logger.Info("OpenRouter API call successful",
		zap.Int("prompt_tokens", openRouterResp.Usage.PromptTokens),
		zap.Int("completion_tokens", openRouterResp.Usage.CompletionTokens),
	)

	return content, nil
}

// cleanJSONResponse removes markdown formatting if present.
func (s *OpenRouterService) cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	return content
}
