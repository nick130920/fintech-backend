package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

// GeminiService se encarga de la comunicación con la API de Google Gemini.
type GeminiService struct {
	client *genai.Client
}

// NewGeminiService crea un nuevo servicio de Gemini.
// La API Key se lee de la variable de entorno "GEMINI_API_KEY".
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

	return &GeminiService{client: client}, nil
}

// TransactionInfo contiene la información extraída de un SMS.
type TransactionInfo struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Merchant    string  `json:"merchant"`
	Date        string  `json:"date"` // Formato YYYY-MM-DD
}

// ExtractTransactionInfoFromSMS analiza un texto de SMS y extrae la información de la transacción.
func (s *GeminiService) ExtractTransactionInfoFromSMS(ctx context.Context, smsContent string) (*TransactionInfo, error) {
	prompt := fmt.Sprintf(`
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
	`, smsContent)

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr[float32](0.0), // Queremos respuestas consistentes
		ResponseMIMEType: "application/json",
	}

	resp, err := s.client.Models.GenerateContent(ctx, "gemini-1.5-flash", genai.Text(prompt), config)
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido con Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("la respuesta de Gemini está vacía")
	}

	// Extraer y limpiar la respuesta JSON
	jsonStr := resp.Text()
	if jsonStr == "" {
		return nil, fmt.Errorf("el contenido de la respuesta de Gemini está vacío")
	}

	// Limpiar el string de cualquier caracter no deseado (como ```json)
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	// Decodificar el JSON a nuestra estructura
	var info TransactionInfo
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return nil, fmt.Errorf("error al decodificar el JSON de Gemini: %w (JSON recibido: %s)", err, jsonStr)
	}

	return &info, nil
}
