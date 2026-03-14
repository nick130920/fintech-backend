package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/genai"
)

const (
	geminiModel = "gemini-2.0-flash"
)

// GeminiService handles interactions with the Google Gemini API using the official SDK.
type GeminiService struct {
	client *genai.Client
	logger *zap.Logger
}

// NewGeminiService creates a new GeminiService using the official Google genai SDK.
func NewGeminiService() (*GeminiService, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno GEMINI_API_KEY no está configurada")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error al crear el logger: %w", err)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("error al crear cliente de Gemini: %w", err)
	}

	return &GeminiService{
		client: client,
		logger: logger,
	}, nil
}

// ExtractTransactionFromSMS analyzes an SMS and extracts transaction details using Gemini.
func (s *GeminiService) ExtractTransactionFromSMS(ctx context.Context, smsContent string) (*TransactionExtraction, error) {
	prompt := s.buildExtractionPrompt(smsContent)

	result, err := s.client.Models.GenerateContent(
		ctx,
		geminiModel,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			Temperature:      genai.Ptr(float32(0.0)),
			MaxOutputTokens:  1024,
			ResponseMIMEType: "application/json",
		},
	)
	if err != nil {
		s.logger.Error("Error calling Gemini API",
			zap.Error(err),
		)
		return nil, fmt.Errorf("error al llamar a Gemini: %w", err)
	}

	// Extract text from response
	responseText := result.Text()
	responseText = s.cleanJSONResponse(responseText)

	// Parse the JSON response
	var extraction TransactionExtraction
	if err := json.Unmarshal([]byte(responseText), &extraction); err != nil {
		s.logger.Error("Error parsing Gemini response",
			zap.String("response", responseText),
			zap.Error(err),
		)
		return nil, fmt.Errorf("error al parsear respuesta de Gemini: %w (respuesta: %s)", err, responseText)
	}

	extraction.RawMessage = smsContent
	s.logger.Info("Transaction extracted successfully via Gemini",
		zap.Float64("amount", extraction.Amount),
		zap.String("type", extraction.TransactionType),
		zap.Float64("confidence", extraction.Confidence),
	)

	return &extraction, nil
}

// buildExtractionPrompt creates the prompt for transaction extraction.
func (s *GeminiService) buildExtractionPrompt(smsContent string) string {
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

// cleanJSONResponse removes markdown formatting if present.
func (s *GeminiService) cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	return content
}
