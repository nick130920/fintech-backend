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
	Model          string          `json:"model"`
	Messages       []OpenRouterMsg `json:"messages"`
	Temperature    float32         `json:"temperature,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
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
		Timeout: 120 * time.Second,
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
	response, err := s.callOpenRouterBatch(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter batch: %w", err)
	}
	var out BatchSMSBudgetResponse
	if err := json.Unmarshal([]byte(response), &out); err != nil {
		s.logger.Error("Error parsing batch budget response", zap.String("response", response), zap.Error(err))
		return nil, fmt.Errorf("parse batch JSON: %w", err)
	}
	return &out, nil
}

func (s *OpenRouterService) buildBatchBudgetPrompt(block string) string {
	return fmt.Sprintf(`Eres un experto en SMS bancarios de Latinoamérica (México, Colombia, etc.).

Analiza el BLOQUE siguiente. Cada línea empieza con [N] donde N es el número de línea.

BLOQUE:
%s

TAREA:
- Para cada línea [N] que sea claramente una NOTIFICACIÓN BANCARIA de movimiento de dinero:
  - Si el usuario GASTÓ / compró / pagó / débito / retiro / envió dinero → transaction_type "expense"
  - Si el usuario RECIBIÓ depósito / abono claro → transaction_type "income"
- IGNORA: OTP, códigos 2FA, publicidad, mensajes personales, recordatorios sin monto claro.
- Incluye en "lines" SOLO líneas con monto numérico claro y confidence >= 0.35.
- amount: número decimal (ej. 1500.50; normaliza miles/decimales locales).
- Una entrada por línea como máximo (usa el número N de [N]).

Responde ÚNICAMENTE JSON válido, sin markdown:
{"lines":[{"line":1,"amount":100.5,"transaction_type":"expense","confidence":0.9}]}`, block)
}

func (s *OpenRouterService) callOpenRouterBatch(ctx context.Context, prompt string) (string, error) {
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
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://money-flow-app.com")
	req.Header.Set("X-Title", "Money Flow App")

	resp, err := s.batchClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", err
	}
	if openRouterResp.Error != nil {
		return "", fmt.Errorf("%s", openRouterResp.Error.Message)
	}
	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("sin choices")
	}
	content := s.cleanJSONResponse(openRouterResp.Choices[0].Message.Content)
	s.logger.Info("OpenRouter batch OK",
		zap.Int("prompt_tokens", openRouterResp.Usage.PromptTokens),
		zap.Int("lines_in_response_estimate", len(content)),
	)
	return content, nil
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
