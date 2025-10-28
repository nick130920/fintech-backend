package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/option"
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

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Gemini: %w", err)
	}

	return &GeminiService{client: client}, nil
}

// Close cierra el cliente de Gemini.
func (s *GeminiService) Close() {
	s.client.Close()
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
	model := s.client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.0) // Queremos respuestas consistentes

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

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido con Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("la respuesta de Gemini está vacía")
	}

	// Extraer y limpiar la respuesta JSON
	rawJSON, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("tipo de respuesta inesperado de Gemini")
	}

	// Limpiar el string de cualquier caracter no deseado (como ```json)
	jsonStr := strings.TrimSpace(string(rawJSON))
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
