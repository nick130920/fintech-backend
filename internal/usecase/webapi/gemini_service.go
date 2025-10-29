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

// PatternGenerationResult holds the structured result from the Gemini API.
type PatternGenerationResult struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	AmountRegex      string   `json:"amount_regex"`
	DateRegex        string   `json:"date_regex"`
	DescriptionRegex string   `json:"description_regex"`
	MerchantRegex    string   `json:"merchant_regex"`
	KeywordsTrigger  []string `json:"keywords_trigger"`
}

// GeminiService handles interactions with the Google Gemini API.
type GeminiService struct {
	client *genai.Client
	logger *zap.Logger
}

// NewGeminiService creates a new GeminiService.
func NewGeminiService(ctx context.Context) (*GeminiService, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno GEMINI_API_KEY no está configurada")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Gemini: %w", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error al crear el logger: %w", err)
	}

	return &GeminiService{client: client, logger: logger}, nil
}

// TransactionInfo holds the structured data extracted from an SMS.
type TransactionInfo struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Merchant    string  `json:"merchant"`
	Date        string  `json:"date"` // Format YYYY-MM-DD
}

// buildPrompt constructs the prompt to be sent to the Gemini API.
func (s *GeminiService) buildPrompt(message string) string {
	return fmt.Sprintf(`
Analyze the following bank transaction notification message and extract key information.
Based on your analysis, generate a JSON object with the following fields:
- "name": A short, descriptive name for this type of transaction (e.g., "Transferencia a Cuenta", "Pago con Tarjeta", "Retiro en Cajero").
- "description": A brief explanation of what this pattern does (e.g., "Detecta transferencias salientes a otras cuentas.").
- "amount_regex": A regex to capture the transaction amount (the numeric value).
- "date_regex": A regex to capture the date (e.g., dd/mm/yyyy). If no date is present, return an empty string.
- "description_regex": A regex to capture the main description of the transaction.
- "merchant_regex": A regex to capture the merchant or destination account. If not applicable, return an empty string.
- "keywords_trigger": An array of keywords that reliably indicate this type of transaction.

The JSON object must be clean, without any markdown formatting like backticks.

Here is the message:
"%s"
`, message)
}

// extractJSON extracts the JSON part from the Gemini API response.
func (s *GeminiService) extractJSON(resp *genai.GenerateContentResponse) string {
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return ""
	}

	jsonStr := resp.Text()
	if jsonStr == "" {
		return ""
	}

	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	return jsonStr
}

// ExtractTransactionInfoFromSMS analyzes an SMS and extracts transaction details.
func (s *GeminiService) ExtractTransactionInfoFromSMS(ctx context.Context, smsContent string) (*TransactionInfo, error) {
	prompt := genai.Text(fmt.Sprintf(`
        Analiza el siguiente SMS de una notificación bancaria y extrae la información en formato JSON.
        El SMS es: "%s"

        El JSON de salida debe tener la siguiente estructura y tipos de datos:
        {
            "amount": float64,      // El monto de la transacción.
            "description": "string", // Una descripción corta y limpia de la compra.
            "merchant": "string",    // El nombre del comercio.
            "date": "string"         // La fecha en formato YYYY-MM-DD. Si no hay fecha, usa la fecha actual.
        }

        Ejemplo de salida:
        {
            "amount": 150.00,
            "description": "Compra en OXXO",
            "merchant": "OXXO",
            "date": "2024-01-15"
        }
    `, smsContent))

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr[float32](0.0),
		ResponseMIMEType: "application/json",
	}

	resp, err := s.client.Models.GenerateContent(ctx, "gemini-2.5-flash", prompt, config)
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido con Gemini: %w", err)
	}

	jsonStr := s.extractJSON(resp)
	if jsonStr == "" {
		return nil, fmt.Errorf("el contenido de la respuesta de Gemini está vacío")
	}

	var info TransactionInfo
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return nil, fmt.Errorf("error al decodificar el JSON de Gemini: %w (JSON recibido: %s)", err, jsonStr)
	}

	return &info, nil
}

// GeneratePatternFromMessage analyzes a message and generates regex patterns.
func (s *GeminiService) GeneratePatternFromMessage(ctx context.Context, messageContent string) (*PatternGenerationResult, error) {
	prompt := genai.Text(s.buildPrompt(messageContent))

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr[float32](0.0),
		ResponseMIMEType: "application/json",
	}

	resp, err := s.client.Models.GenerateContent(ctx, "gemini-2.5-flash", prompt, config)
	if err != nil {
		return nil, fmt.Errorf("error al generar patrón con Gemini: %w", err)
	}

	jsonStr := s.extractJSON(resp)
	if jsonStr == "" {
		return nil, fmt.Errorf("el contenido de la respuesta de Gemini para generar patrón está vacío")
	}

	var result PatternGenerationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("error al decodificar el JSON de Gemini para el patrón: %w (JSON recibido: %s)", err, jsonStr)
	}

	s.logger.Info("Successfully parsed pattern generation result from Gemini API")
	return &result, nil
}
